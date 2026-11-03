package api

import (
	"reflect"
	"strings"
	"testing"

	"xboard/internal/model"
)

func TestFilterConditionNewOperators(t *testing.T) {
	cases := []struct {
		name  string
		value any
		sql   string
		args  []any
	}{
		{"neq 对空值安全", "neq:3", "(plan_id IS NULL OR plan_id <> ?)", []any{int64(3)}},
		{"neq 文本", "neq:abc", "(plan_id IS NULL OR plan_id <> ?)", []any{"abc"}},
		{"neq 空串", "neq:", "(plan_id IS NULL OR plan_id <> ?)", []any{""}},
		{"prefix", "prefix:ab", "plan_id LIKE ?", []any{"ab%"}},
		{"suffix", "suffix:ab", "plan_id LIKE ?", []any{"%ab"}},
		{"like 转义下划线", "like:a_b", "plan_id LIKE ?", []any{`%a\_b%`}},
		{"like 转义百分号", "like:50%", "plan_id LIKE ?", []any{`%50\%%`}},
		{"like 转义反斜杠", "like:a" + `\` + "b", "plan_id LIKE ?", []any{"%a" + `\\` + "b%"}},
		{"notlike 转义", "notlike:a_b", "plan_id NOT LIKE ?", []any{`%a\_b%`}},
		{"prefix 转义", "prefix:a%", "plan_id LIKE ?", []any{`a\%%`}},
		{"普通值也转义", "a_b", "plan_id LIKE ?", []any{`%a\_b%`}},
	}
	for _, c := range cases {
		sql, args := filterCondition("plan_id", c.value)
		if sql != c.sql || !reflect.DeepEqual(args, c.args) {
			t.Errorf("%s: got (%q, %#v) want (%q, %#v)", c.name, sql, args, c.sql, c.args)
		}
	}
}

func TestBuildFilterExprGroups(t *testing.T) {
	leaf := func(id, v string) filterItem { return filterItem{ID: id, Value: v} }
	// any(online_count>0, all(plan_id 非空, expired_at<100))
	f := filterItem{Any: []filterItem{
		leaf("online_count", "gt:0"),
		{All: []filterItem{leaf("plan_id", "notnull:1"), leaf("expired_at", "lt:100")}},
	}}
	sql, args := buildFilterExpr(f, adminUserFilter)
	want := "(online_count > ? OR (plan_id IS NOT NULL AND expired_at < ?))"
	if sql != want || !reflect.DeepEqual(args, []any{int64(0), int64(100)}) {
		t.Fatalf("got (%q, %#v)\nwant %q", sql, args, want)
	}
	// 不被支持的字段在组里被跳过，其余条件保留
	f2 := filterItem{Any: []filterItem{leaf("nope", "eq:1"), leaf("banned", "eq:1")}}
	sql, args = buildFilterExpr(f2, adminUserFilter)
	if sql != "(banned = ?)" || len(args) != 1 {
		t.Fatalf("unknown child must be skipped: %q %#v", sql, args)
	}
	// 整组都不被支持 → 整项忽略
	if sql, _ := buildFilterExpr(filterItem{Any: []filterItem{leaf("nope", "eq:1")}}, adminUserFilter); sql != "" {
		t.Fatalf("empty group must produce no condition, got %q", sql)
	}
	// 同时有 any 和 all：两者 AND
	f3 := filterItem{Any: []filterItem{leaf("banned", "eq:1")}, All: []filterItem{leaf("id", "gt:5")}}
	sql, _ = buildFilterExpr(f3, adminUserFilter)
	if sql != "((banned = ?) AND (id > ?))" {
		t.Fatalf("any+all: %q", sql)
	}
}

func TestAdminFiltersGroupSQLAndOrder(t *testing.T) {
	_, db := dryRunApp(t)
	c := filterCtx(t, `[
		{"id":"email","value":"like:@163.com"},
		{"any":[{"id":"online_count","value":"gt:0"},{"id":"plan_id","value":"null:1"}]},
		{"id":"banned","value":"eq:0"}
	]`)
	var rows []model.User
	stmt := adminFilters(c, adminUserFilter, db.Model(&model.User{})).Find(&rows).Statement
	sql := stmt.SQL.String()
	// GORM 会在含 OR 的条件外再包一层括号，这里只断言「OR 被括号包住」和各条件的先后顺序。
	i1, i2, i3 := strings.Index(sql, "email LIKE ?"), strings.Index(sql, "(online_count > ? OR plan_id IS NULL)"), strings.Index(sql, "banned = ?")
	if i1 < 0 || i2 < i1 || i3 < i2 {
		t.Fatalf("sql 结构不对: %q", sql)
	}
	if got := stmt.Vars; len(got) != 3 || got[0] != "%@163.com%" || got[1] != int64(0) || got[2] != int64(0) {
		t.Fatalf("参数顺序不对: %#v", got)
	}
}

func TestCheckFilters(t *testing.T) {
	ok := filterCtx(t, `[{"any":[{"id":"online_count","value":"gt:0"},{"all":[{"id":"banned","value":"eq:1"},{"id":"plan.name","value":"like:x"}]}]}]`)
	if err := checkFilters(ok, adminUserFilter); err != nil {
		t.Fatalf("合法的嵌套条件不应被拒绝: %v", err)
	}
	// 条件组里的未知字段也必须被发现，不能因为藏在组里而绕过检查
	hidden := filterCtx(t, `[{"any":[{"id":"online_count","value":"gt:0"},{"all":[{"id":"password","value":"x"}]}]}]`)
	if err := checkFilters(hidden, adminUserFilter); err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("嵌套里的未知字段必须被拒绝, got %v", err)
	}
	if bad := unknownFilterFields(hidden, adminUserFilter); strings.Join(bad, ",") != "password" {
		t.Fatalf("unknownFilterFields 也应递归: %v", bad)
	}
	// 嵌套过深
	deep := `{"id":"banned","value":"eq:1"}`
	for i := 0; i < maxFilterDepth+1; i++ {
		deep = `{"any":[` + deep + `]}`
	}
	if err := checkFilters(filterCtx(t, "["+deep+"]"), adminUserFilter); err == nil || !strings.Contains(err.Error(), "嵌套") {
		t.Fatalf("过深的嵌套必须被拒绝, got %v", err)
	}
	// 条件过多
	var many []string
	for i := 0; i < maxFilterLeaves+1; i++ {
		many = append(many, `{"id":"banned","value":"eq:1"}`)
	}
	if err := checkFilters(filterCtx(t, "["+strings.Join(many, ",")+"]"), adminUserFilter); err == nil || !strings.Contains(err.Error(), "过多") {
		t.Fatalf("过多的条件必须被拒绝, got %v", err)
	}
	// 刚好在上限内的深度可以通过
	edge := `{"id":"banned","value":"eq:1"}`
	for i := 0; i < maxFilterDepth; i++ {
		edge = `{"any":[` + edge + `]}`
	}
	if err := checkFilters(filterCtx(t, "["+edge+"]"), adminUserFilter); err != nil {
		t.Fatalf("上限内的嵌套应通过: %v", err)
	}
}
