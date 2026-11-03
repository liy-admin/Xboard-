package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"xboard/internal/model"
	"xboard/internal/nodeparse"
	"xboard/internal/resp"
)

// 节点批量导入：解析分享链接或订阅地址（ProxyParserService）、批量保存、导入订阅源配置。
// 对应 V2/Admin/Server/ManageController 的 parseNodes、batchSave、batchImport、get/saveImportSubscriptions。

// parseInput 按 type 解析 links 或 subscription 输入。
func parseInput(typ, content, subURL string) (nodeparse.Result, error) {
	if typ == "links" {
		return nodeparse.ParseText(content), nil
	}
	res, err := nodeparse.ParseSubscription(subURL)
	if err != nil {
		return nodeparse.Result{Errors: []nodeparse.ParseError{{Error: err.Error()}}}, nil
	}
	return res, nil
}

func (a *App) nodeParse(c *gin.Context) {
	m := bodyMap(c)
	typ, _ := mStr(m, "type")
	if typ != "links" && typ != "subscription" {
		resp.Fail(c, 422, "type 参数有误")
		return
	}
	content, _ := mStr(m, "content")
	subURL, _ := mStr(m, "url")
	parsed, err := parseInput(typ, content, subURL)
	if err != nil {
		resp.Fail(c, 422, err.Error())
		return
	}
	out := make([]gin.H, 0, len(parsed.Servers))
	for i, s := range parsed.Servers {
		out = append(out, gin.H{"index": i, "success": true, "type": s.Type, "code": s.Code, "name": s.Name,
			"host": s.Host, "port": s.Port})
	}
	resp.OK(c, gin.H{"servers": out, "parse_errors": parsed.Errors})
}

// serverRow 把一条节点数据转为可写入 v2_server 的列映射。
func serverRow(data map[string]any) (map[string]any, string) {
	typ, _ := data["type"].(string)
	if !nodeTypes[typ] {
		return nil, fmt.Sprintf("type 不合法: %s", typ)
	}
	row := map[string]any{}
	for _, k := range nodeWritable {
		v, ok := data[k]
		if !ok || v == nil {
			continue
		}
		if nodeJSONCols[k] {
			b, _ := json.Marshal(normalizeIDList(k, v))
			row[k] = string(b)
		} else {
			row[k] = v
		}
	}
	for _, k := range []string{"name", "host", "port", "server_port", "rate"} {
		if v, ok := row[k]; !ok || v == "" {
			return nil, k + " 不能为空"
		}
	}
	row["created_at"], row["updated_at"] = time.Now(), time.Now()
	return row, ""
}

func (a *App) nodeBatchSave(c *gin.Context) {
	var in struct {
		Servers []map[string]any `json:"servers"`
	}
	if err := requestBindJSON(c, &in); err != nil || len(in.Servers) == 0 || len(in.Servers) > 100 {
		resp.Fail(c, 422, "servers 参数有误")
		return
	}
	results := []gin.H{}
	created, failed := 0, 0
	for i, s := range in.Servers {
		row, msg := serverRow(s)
		if row == nil {
			results = append(results, gin.H{"index": i, "success": false, "error": msg})
			failed++
			continue
		}
		if err := a.S.DB.Model(&model.Server{}).Create(row).Error; err != nil {
			results = append(results, gin.H{"index": i, "success": false, "error": "创建失败: " + err.Error()})
			failed++
			continue
		}
		name, _ := row["name"].(string)
		results = append(results, gin.H{"index": i, "success": true, "name": name})
		created++
	}
	resp.OK(c, gin.H{"created": created, "failed": failed, "results": results})
}

// requestBindJSON 读取 JSON 请求体到目标结构。
func requestBindJSON(c *gin.Context, v any) error {
	dec := json.NewDecoder(c.Request.Body)
	dec.UseNumber()
	return dec.Decode(v)
}

// nodeBatchImport 对应 batchImport：解析后按默认值批量入库（show 默认为 0，即不展示）。
func (a *App) nodeBatchImport(c *gin.Context) {
	m := bodyMap(c)
	typ, _ := mStr(m, "type")
	if typ != "links" && typ != "subscription" {
		resp.Fail(c, 422, "type 参数有误")
		return
	}
	content, _ := mStr(m, "content")
	subURL, _ := mStr(m, "url")
	parsed, err := parseInput(typ, content, subURL)
	if err != nil {
		resp.Fail(c, 422, err.Error())
		return
	}
	defaults, _ := m["defaults"].(map[string]any)
	results := []gin.H{}
	created, failed := 0, 0
	for i, s := range parsed.Servers {
		data := map[string]any{
			"type": s.Type, "code": s.Code, "name": s.Name, "host": s.Host, "port": s.Port,
			"server_port": s.ServerPort, "rate": s.Rate, "protocol_settings": s.ProtocolSettings,
		}
		if defaults != nil {
			for _, k := range []string{"group_ids", "route_ids", "tags"} {
				if arr, ok := defaults[k].([]any); ok && len(arr) > 0 {
					data[k] = arr
				}
			}
			if r, ok := defaults["rate"]; ok && r != nil {
				data["rate"] = r
			}
		}
		show := 0
		if defaults != nil {
			if v, ok := mInt(defaults, "show"); ok {
				show = int(v)
			}
		}
		data["show"] = show != 0
		row, msg := serverRow(data)
		if row == nil {
			results = append(results, gin.H{"index": i, "success": false, "name": s.Name, "error": msg})
			failed++
			continue
		}
		if err := a.S.DB.Model(&model.Server{}).Create(row).Error; err != nil {
			results = append(results, gin.H{"index": i, "success": false, "name": s.Name, "error": "创建失败: " + err.Error()})
			failed++
			continue
		}
		results = append(results, gin.H{"index": i, "success": true, "name": s.Name, "type": s.Type})
		created++
	}
	resp.OK(c, gin.H{"created": created, "failed": failed, "parse_errors": parsed.Errors, "results": results})
}

// importSubscriptionsGet 对应 getImportSubscriptions。
func (a *App) importSubscriptionsGet(c *gin.Context) {
	raw := a.Set.Get("node_import_subscriptions", "")
	var list []map[string]any
	if strings.TrimSpace(raw) != "" && json.Unmarshal([]byte(raw), &list) == nil {
		resp.OK(c, list)
		return
	}
	// 旧版本保存时把列表写成了 Go 的 fmt 文本（"[map[k:v ...]]"），读不出来；这里还原并写回正确的 JSON。
	if rec, ok := recoverFmtSubscriptions(raw); ok {
		if err := a.settingUpsert("node_import_subscriptions", rec); err == nil {
			a.Set.Invalidate()
		}
		resp.OK(c, rec)
		return
	}
	resp.OK(c, []any{})
}

// importSubscriptionsSave 对应 saveImportSubscriptions：规范化后写入设置 node_import_subscriptions。
func (a *App) importSubscriptionsSave(c *gin.Context) {
	var in struct {
		Subscriptions []map[string]any `json:"subscriptions"`
	}
	if err := requestBindJSON(c, &in); err != nil {
		resp.Fail(c, 422, "参数有误")
		return
	}
	if len(in.Subscriptions) > 50 {
		resp.Fail(c, 422, "订阅源最多 50 个")
		return
	}
	list := make([]map[string]any, 0, len(in.Subscriptions))
	for _, s := range in.Subscriptions {
		name, _ := s["name"].(string)
		u, _ := s["url"].(string)
		if name == "" || u == "" {
			resp.Fail(c, 422, "name 与 url 不能为空")
			return
		}
		if pu, err := url.Parse(u); err != nil || (pu.Scheme != "http" && pu.Scheme != "https") {
			resp.Fail(c, 422, "url 格式有误")
			return
		}
		id, _ := s["id"].(string)
		if id == "" {
			id = uuid.NewString()
		}
		rate := 1.0
		if r, ok := toNum(s["rate"]); ok {
			rate = r
		}
		show, _ := mInt(s, "show")
		interval, ok := mInt(s, "interval_hours")
		if !ok || interval == 0 {
			interval = 24
		}
		schedule, _ := mBool(s, "schedule_enabled")
		refetch, _ := mBool(s, "refetch_balala")
		tag, _ := s["tag"].(string)
		lastRun, _ := s["last_run_at"].(string)
		var lr any
		if lastRun != "" {
			lr = lastRun
		}
		list = append(list, map[string]any{
			"id": id, "name": name, "url": u, "tag": tag, "group_ids": idList(s["group_ids"]), "rate": rate,
			"show": show, "schedule_enabled": schedule, "interval_hours": interval, "refetch_balala": refetch,
			"last_run_at": lr,
		})
	}
	if err := a.settingUpsert("node_import_subscriptions", list); err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	a.Set.Invalidate()
	resp.OK(c, list)
}

// subKeys 是订阅预设的全部字段名（fmt 打印 map 时按键名排序）。
var subKeys = []string{"group_ids", "id", "interval_hours", "last_run_at", "name", "rate", "refetch_balala", "schedule_enabled", "show", "tag", "url"}

var subKeyRe = regexp.MustCompile(`(?:^| )(` + strings.Join(subKeys, "|") + `):`)

// recoverFmtSubscriptions 把 fmt.Sprint 打印出的 "[map[k:v ...] map[...]]" 还原为预设列表。
// 字段名是固定的，所以按「已知键名」切分值，名称里带空格也能还原。无法识别时返回 false。
func recoverFmtSubscriptions(raw string) ([]map[string]any, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "[map[") || !strings.HasSuffix(raw, "]]") {
		return nil, false
	}
	inner := raw[len("[map[") : len(raw)-len("]]")]
	var out []map[string]any
	for _, body := range strings.Split(inner, "] map[") {
		item, ok := recoverFmtEntry(body)
		if !ok {
			return nil, false
		}
		out = append(out, item)
	}
	return out, len(out) > 0
}

func recoverFmtEntry(body string) (map[string]any, bool) {
	locs := subKeyRe.FindAllStringSubmatchIndex(body, -1)
	if len(locs) == 0 {
		return nil, false
	}
	vals := map[string]string{}
	for i, loc := range locs {
		key := body[loc[2]:loc[3]]
		end := len(body)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		vals[key] = strings.TrimSpace(body[loc[1]:end])
	}
	if vals["url"] == "" || vals["name"] == "" {
		return nil, false
	}
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return n }
	rate := 1.0
	if f, err := strconv.ParseFloat(vals["rate"], 64); err == nil {
		rate = f
	}
	var lastRun any
	if v := vals["last_run_at"]; v != "" && v != "<nil>" {
		lastRun = v
	}
	groups := []int{}
	for _, f := range strings.Fields(strings.Trim(vals["group_ids"], "[]")) {
		if n, err := strconv.Atoi(f); err == nil {
			groups = append(groups, n)
		}
	}
	interval := atoi(vals["interval_hours"])
	if interval == 0 {
		interval = 24
	}
	return map[string]any{
		"id": vals["id"], "name": vals["name"], "url": vals["url"], "tag": vals["tag"],
		"group_ids": groups, "rate": rate, "show": atoi(vals["show"]),
		"schedule_enabled": vals["schedule_enabled"] == "true", "interval_hours": interval,
		"refetch_balala": vals["refetch_balala"] == "true", "last_run_at": lastRun,
	}, true
}

// tagCondition 返回「tags 数组包含该标识」的条件参数，与定时导入按标识删旧节点用的是同一判断。
func tagCondition(tag string) string {
	b, _ := json.Marshal(tag)
	return string(b)
}

func tagFrom(c *gin.Context) (string, bool) {
	tag, _ := mStr(requestParams(c), "tag")
	tag = strings.TrimSpace(tag)
	if tag == "" {
		resp.Fail(c, 422, "该订阅没有填写订阅标识，无法确定它导入了哪些节点")
		return "", false
	}
	return tag, true
}

// nodeCountByTag 返回带该订阅标识的节点数（删除前确认用，不触发客户端推送）。
func (a *App) nodeCountByTag(c *gin.Context) {
	tag, ok := tagFrom(c)
	if !ok {
		return
	}
	var n int64
	a.S.DB.Model(&model.Server{}).Where("JSON_CONTAINS(tags, ?)", tagCondition(tag)).Count(&n)
	resp.OK(c, gin.H{"tag": tag, "count": n})
}

// nodeDeleteByTag 删除带该订阅标识的全部节点（已保存订阅的「删除所属节点」），预设本身保留。
func (a *App) nodeDeleteByTag(c *gin.Context) {
	tag, ok := tagFrom(c)
	if !ok {
		return
	}
	res := a.S.DB.Where("JSON_CONTAINS(tags, ?)", tagCondition(tag)).Delete(&model.Server{})
	if res.Error != nil {
		resp.Fail(c, 500, "删除失败")
		return
	}
	resp.OK(c, gin.H{"tag": tag, "deleted": res.RowsAffected})
}
