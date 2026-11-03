package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"xboard/internal/model"
	"xboard/internal/store"
)

// dryRunApp 构造一个不连接数据库、只生成 SQL 的 App。
func dryRunApp(t *testing.T) (*App, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/x", SkipInitializeWithVersion: true}),
		&gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return &App{S: &store.Store{DB: db}}, db
}

func filterCtx(t *testing.T, filter string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/x?filter="+url.QueryEscape(filter), nil)
	return c
}

func TestCountDistinctDevices(t *testing.T) {
	now := int64(10_000)
	data := map[string]string{
		"1:1.1.1.1":     "9990", // 节点 1 上的 1.1.1.1
		"2:1.1.1.1":     "9990", // 同一个 IP 连了节点 2：仍是一台设备
		"1:2.2.2.2":     "9900",
		"3:2001:db8::1": "9999", // IPv6 含冒号，只在第一个冒号处切
		"1:3.3.3.3":     "9000", // 超过 300 秒，已过期
		"bad-field":     "9999", // 没有冒号
		"1:4.4.4.4":     "not-a-number",
		"1:":            "9999", // IP 为空
	}
	if got := countDistinctDevices(data, now); got != 3 {
		t.Fatalf("want 3 distinct live IPs, got %d", got)
	}
	if got := countDistinctDevices(nil, now); got != 0 {
		t.Fatalf("empty must be 0, got %d", got)
	}
}

func TestUnknownFilterFields(t *testing.T) {
	c := filterCtx(t, `[{"id":"email","value":"a"},{"id":"online_count","value":"gt:0"},{"id":"nope","value":"x"},{"id":"plan.name","value":"p"},{"id":"evil.col","value":"x"},{"id":"password","value":"x"}]`)
	bad := unknownFilterFields(c, adminUserFilter)
	if strings.Join(bad, ",") != "nope,evil.col,password" {
		t.Fatalf("got %v", bad)
	}
}

// 前端筛选界面里提供的每一个字段，后端都必须支持；否则会出现「选了筛选却被静默忽略」。
func TestEveryFrontendFilterFieldIsSupported(t *testing.T) {
	src, err := os.ReadFile("../../../admin-src/source/views/config/userFilter.js")
	if err != nil {
		t.Skip("前端源码不在（发布环境）：", err)
	}
	text := string(src)
	start := strings.Index(text, "export const FIELDS")
	end := strings.Index(text, "export const OP_KIND")
	if start < 0 || end < start {
		t.Fatal("没找到 FIELDS 定义，解析规则需要更新")
	}
	fields := regexp.MustCompile(`field:\s*"([a-z_.]+)"`).FindAllStringSubmatch(text[start:end], -1)
	if len(fields) < 10 {
		t.Fatalf("只解析到 %d 个筛选字段，解析规则可能失效", len(fields))
	}
	for _, m := range fields {
		f := m[1]
		if rel, _, ok := strings.Cut(f, "."); ok {
			if _, known := filterRelations[rel]; !known {
				t.Errorf("筛选字段 %s 的关联 %s 后端不支持", f, rel)
			}
			continue
		}
		if !adminUserFilter[f] {
			t.Errorf("筛选字段 %s 前端提供了，但后端白名单没有（会被忽略）", f)
		}
	}
}

func TestOnlineCountFilterBuildsSQL(t *testing.T) {
	a, db := dryRunApp(t)
	cases := []struct{ filter, want string }{
		{`[{"id":"online_count","value":"gt:0"}]`, "online_count > ?"},
		{`[{"id":"plan_id","value":"null:1"}]`, "plan_id IS NULL"},
		{`[{"id":"plan_id","value":"notnull:1"}]`, "plan_id IS NOT NULL"},
		{`[{"id":"remarks","value":"like:a:b"}]`, "remarks LIKE ?"}, // 内容里带冒号也按「包含」处理
	}
	for _, tc := range cases {
		c := filterCtx(t, tc.filter)
		var rows []model.User
		stmt := adminFilters(c, adminUserFilter, db.Model(&model.User{})).Find(&rows).Statement
		if sql := stmt.SQL.String(); !strings.Contains(sql, tc.want) {
			t.Errorf("%s => %s, want contains %q", tc.filter, sql, tc.want)
		}
		_ = a
	}
}

func TestAdminUserScopeRejectsUnsafeFilteredScope(t *testing.T) {
	a, _ := dryRunApp(t)
	for name, filter := range map[string]string{
		"空筛选":    `[]`,
		"未知字段":   `[{"id":"nope","value":"x"}]`,
		"部分未知字段": `[{"id":"banned","value":"eq:0"},{"id":"nope","value":"x"}]`,
	} {
		c := filterCtx(t, filter)
		if _, err := a.adminUserScope(c, "filtered", nil); err == nil {
			t.Errorf("%s：filtered 范围必须被拒绝，否则会作用到全部用户", name)
		}
	}
	c := filterCtx(t, `[{"id":"online_count","value":"gt:0"}]`)
	if _, err := a.adminUserScope(c, "filtered", nil); err != nil {
		t.Errorf("合法筛选不应被拒绝: %v", err)
	}
	if _, err := a.adminUserScope(c, "selected", nil); err == nil {
		t.Error("selected 没有 user_ids 必须被拒绝")
	}
}

func TestBatchBanValidatesInput(t *testing.T) {
	a, _ := dryRunApp(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/ban", a.adminUserBan)
	for name, body := range map[string]string{
		"非法 scope":      `{"scope":"everyone"}`,
		"filtered 无筛选":  `{"scope":"filtered","filter":[]}`,
		"filtered 未知字段": `{"scope":"filtered","filter":[{"id":"nope","value":"x"}]}`,
		"selected 无用户":  `{"scope":"selected","user_ids":[]}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/ban", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 422 {
			t.Errorf("%s: 期望 422，得到 %d，响应 %s", name, w.Code, w.Body.String())
		}
	}
}
