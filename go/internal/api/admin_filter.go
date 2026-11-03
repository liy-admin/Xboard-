package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 管理端列表的过滤与排序，对应 PHP 的 QueryOperators trait 与各控制器的 applyFilters / applySorting。
//
// 过滤项 {id, value, logic?}：
//   - value 为数组          → id IN (...)
//   - value 为 "op:val"     → eq / gt / gte / lt / lte / like / notlike / null / notnull（未知运算符按 like）
//   - 其他                  → id LIKE %value%
//   - logic = "or"          → 以 OR 连接（默认 AND）
//   - id 形如 relation.col  → 对关联表做子查询（仅 plan / invite_user / group）
//   - {"any": [...]}        → 条件组，组内各项以 OR 连接，整体作为一个条件与其他条件 AND
//   - {"all": [...]}        → 条件组，组内各项以 AND 连接（用来在 any 里表达「A 且 B」）
//     条件组可以嵌套（最多 maxFilterDepth 层、共 maxFilterLeaves 个叶子条件）。
//
// 运算符：eq neq gt gte lt lte like notlike prefix suffix null notnull。
//   - neq 对空值安全：col IS NULL OR col <> ?（否则「套餐不等于 3」会漏掉没有套餐的用户）；
//   - like / notlike / prefix / suffix 会转义 % 和 _，按字面匹配。
// 参数既可来自 JSON/表单 body 里的数组，也可来自查询串里的 JSON 字符串，因此 GET 与 POST 都能用。

type filterItem struct {
	ID    string       `json:"id"`
	Value any          `json:"value"`
	Logic string       `json:"logic"`
	Any   []filterItem `json:"any"`
	All   []filterItem `json:"all"`
}

const (
	maxFilterDepth  = 4  // 条件组最多嵌套几层
	maxFilterLeaves = 60 // 叶子条件总数上限，避免构造出过大的 SQL
)

type sortItem struct {
	ID   string `json:"id"`
	Desc bool   `json:"desc"`
}

// decodeList 把 params[key] 解码成结构体切片，兼容「已解析数组」与「JSON 字符串」两种来源。
func decodeList(v any, out any) {
	switch x := v.(type) {
	case nil:
		return
	case string:
		if x != "" {
			_ = json.Unmarshal([]byte(x), out)
		}
	default:
		if b, err := json.Marshal(x); err == nil {
			_ = json.Unmarshal(b, out)
		}
	}
}

func requestFilters(c *gin.Context) []filterItem {
	var fs []filterItem
	decodeList(requestParams(c)["filter"], &fs)
	return fs
}

func requestSorts(c *gin.Context) []sortItem {
	var ss []sortItem
	decodeList(requestParams(c)["sort"], &ss)
	return ss
}

var filterOps = map[string]string{
	"eq": "=", "gt": ">", "gte": ">=", "lt": "<", "lte": "<=",
	"like": "LIKE", "notlike": "NOT LIKE", "null": "null", "notnull": "notnull",
	"neq": "neq", "prefix": "prefix", "suffix": "suffix",
}

// escapeLike 转义 LIKE 的通配符，让用户输入的 % 和 _ 按字面匹配。
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// computedFilterCols 对应 PHP 里 'total_used' => DB::raw('(u + d)')。
var computedFilterCols = map[string]string{"total_used": "(u + d)"}

// filterRelations 为 relation.col 过滤使用的子查询：关联名 → (本表外键列, 关联表)。
var filterRelations = map[string][2]string{
	"plan":        {"plan_id", "v2_plan"},
	"invite_user": {"invite_user_id", "v2_user"},
	"group":       {"group_id", "v2_server_group"},
}

func numericValue(s string) any {
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return s
	}
	if strings.Contains(s, ".") {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return s
	}
	return n
}

// condition 生成单个字段的 SQL 条件；返回空串表示忽略该过滤项。
func filterCondition(col string, value any) (string, []any) {
	if arr, ok := value.([]any); ok {
		return col + " IN ?", []any{arr}
	}
	s, isStr := value.(string)
	if !isStr || !strings.Contains(s, ":") {
		return col + " LIKE ?", []any{"%" + escapeLike(fmt.Sprint(value)) + "%"}
	}
	op, raw, _ := strings.Cut(s, ":")
	sqlOp, known := filterOps[strings.ToLower(op)]
	if !known {
		sqlOp = "LIKE"
	}
	switch sqlOp {
	case "null":
		return col + " IS NULL", nil
	case "notnull":
		return col + " IS NOT NULL", nil
	case "LIKE", "NOT LIKE":
		return col + " " + sqlOp + " ?", []any{"%" + escapeLike(raw) + "%"}
	case "prefix":
		return col + " LIKE ?", []any{escapeLike(raw) + "%"}
	case "suffix":
		return col + " LIKE ?", []any{"%" + escapeLike(raw)}
	case "neq":
		return "(" + col + " IS NULL OR " + col + " <> ?)", []any{numericValue(raw)}
	}
	return col + " " + sqlOp + " ?", []any{numericValue(raw)}
}

// filterLeaf 生成叶子条件的 SQL；字段不被支持时返回空串。
func filterLeaf(f filterItem, allowed map[string]bool) (string, []any) {
	if rel, col, ok := strings.Cut(f.ID, "."); ok {
		meta, known := filterRelations[rel]
		if !known || !safeIdent(col) {
			return "", nil
		}
		sub, subArgs := filterCondition(col, f.Value)
		return meta[0] + " IN (SELECT id FROM " + meta[1] + " WHERE " + sub + ")", subArgs
	}
	if !allowed[f.ID] {
		return "", nil
	}
	col := f.ID
	if expr, ok := computedFilterCols[f.ID]; ok {
		col = expr
	} else if f.ID == "group_ids" {
		col = "group_id"
	}
	return filterCondition(col, f.Value)
}

// filterGroup 把一组条件用 sep（" AND " / " OR "）连成带括号的一个条件。
func filterGroup(items []filterItem, sep string, allowed map[string]bool) (string, []any) {
	var parts []string
	var args []any
	for _, it := range items {
		s, a := buildFilterExpr(it, allowed)
		if s == "" {
			continue
		}
		parts = append(parts, s)
		args = append(args, a...)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "(" + strings.Join(parts, sep) + ")", args
}

// buildFilterExpr 把一个筛选项（叶子或条件组）转成 SQL 条件；返回空串表示整项被忽略。
func buildFilterExpr(f filterItem, allowed map[string]bool) (string, []any) {
	if len(f.Any) == 0 && len(f.All) == 0 {
		return filterLeaf(f, allowed)
	}
	var parts []string
	var args []any
	if len(f.Any) > 0 {
		if s, a := filterGroup(f.Any, " OR ", allowed); s != "" {
			parts, args = append(parts, s), append(args, a...)
		}
	}
	if len(f.All) > 0 {
		if s, a := filterGroup(f.All, " AND ", allowed); s != "" {
			parts, args = append(parts, s), append(args, a...)
		}
	}
	switch len(parts) {
	case 0:
		return "", nil
	case 1:
		return parts[0], args // 组本身已带括号
	}
	return "(" + strings.Join(parts, " AND ") + ")", args
}

// adminFilters 按 allowed 白名单应用请求里的 filter。顶层各项默认 AND；顶层单项带 logic=or 时用 OR 连接。
func adminFilters(c *gin.Context, allowed map[string]bool, q *gorm.DB) *gorm.DB {
	for _, f := range requestFilters(c) {
		cond, args := buildFilterExpr(f, allowed)
		if cond == "" {
			continue
		}
		if strings.EqualFold(f.Logic, "or") {
			q = q.Or(cond, args...)
		} else {
			q = q.Where(cond, args...)
		}
	}
	return q
}

// walkFilters 递归遍历筛选项，统计叶子数和最大嵌套深度，并收集不被支持的字段名。
func walkFilters(items []filterItem, allowed map[string]bool, depth int, leaves *int, maxDepth *int, bad *[]string) {
	if depth > *maxDepth {
		*maxDepth = depth
	}
	for _, f := range items {
		if len(f.Any) > 0 || len(f.All) > 0 {
			walkFilters(f.Any, allowed, depth+1, leaves, maxDepth, bad)
			walkFilters(f.All, allowed, depth+1, leaves, maxDepth, bad)
			continue
		}
		*leaves++
		if rel, col, ok := strings.Cut(f.ID, "."); ok {
			if _, known := filterRelations[rel]; !known || !safeIdent(col) {
				*bad = append(*bad, f.ID)
			}
			continue
		}
		if !allowed[f.ID] {
			*bad = append(*bad, f.ID)
		}
	}
}

// unknownFilterFields 返回请求里不被支持的筛选字段名（含条件组里的）：既不在白名单里，也不是受支持的 relation.col。
// adminFilters 对这些字段是直接跳过的；对「按筛选结果批量操作」的接口必须先拒绝，否则被跳过的条件会让操作范围变大。
func unknownFilterFields(c *gin.Context, allowed map[string]bool) []string {
	var bad []string
	leaves, depth := 0, 0
	walkFilters(requestFilters(c), allowed, 0, &leaves, &depth, &bad)
	return bad
}

// checkFilters 校验请求里的筛选：字段必须受支持，条件组嵌套不超过 maxFilterDepth 层，叶子条件不超过 maxFilterLeaves 个。
func checkFilters(c *gin.Context, allowed map[string]bool) error {
	var bad []string
	leaves, depth := 0, 0
	walkFilters(requestFilters(c), allowed, 0, &leaves, &depth, &bad)
	switch {
	case len(bad) > 0:
		return fmt.Errorf("不支持的筛选字段：%s", strings.Join(bad, "、"))
	case depth > maxFilterDepth:
		return fmt.Errorf("筛选条件嵌套过深（最多 %d 层）", maxFilterDepth)
	case leaves > maxFilterLeaves:
		return fmt.Errorf("筛选条件过多（最多 %d 个）", maxFilterLeaves)
	}
	return nil
}

func safeIdent(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// adminSorts 按白名单应用 sort（[{id, desc}]）。
func adminSorts(c *gin.Context, allowed map[string]bool, q *gorm.DB) *gorm.DB {
	for _, s := range requestSorts(c) {
		if !allowed[s.ID] {
			continue
		}
		dir := "ASC"
		if s.Desc {
			dir = "DESC"
		}
		q = q.Order(s.ID + " " + dir)
	}
	return q
}
