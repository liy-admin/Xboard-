//go:build integration

package api

// 集成测试：需要真实的 MySQL / MariaDB 与 Redis，用与服务相同的环境变量连接（DB_* / REDIS_*）。
// 只写本测试创建的临时用户与 Redis 键，结束后清理。运行方式：
//
//	DB_HOST=... DB_DATABASE=... DB_USERNAME=... DB_PASSWORD=... REDIS_DB=15 \
//	  go test -tags integration ./internal/api -run Integration -v

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/config"
	"xboard/internal/model"
	"xboard/internal/setting"
	"xboard/internal/store"
)

func integrationApp(t *testing.T) *App {
	t.Helper()
	cfg := config.Load()
	st, err := store.Open(cfg)
	if err != nil {
		t.Skip("没有可用的数据库：", err)
	}
	if err := st.Redis.Ping(context.Background()).Err(); err != nil {
		t.Skip("没有可用的 Redis：", err)
	}
	return &App{S: st, Set: setting.New(st.DB)}
}

func newITUser(t *testing.T, a *App, tag string, admin bool) *model.User {
	t.Helper()
	ts := now()
	u := &model.User{
		Email: fmt.Sprintf("it-online-%s-%d@example.test", tag, time.Now().UnixNano()), Password: "x",
		UUID: auth.RandHex(16), Token: auth.RandHex(16), IsAdmin: admin, CreatedAt: ts, UpdatedAt: ts,
	}
	if err := a.S.DB.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		a.S.DB.Exec("DELETE FROM v2_user WHERE id = ?", u.ID)
		a.S.Redis.Del(context.Background(), devicePrefix+strconv.FormatInt(u.ID, 10))
	})
	return u
}

type onlineRow struct {
	OnlineCount  int
	LastOnlineAt *time.Time
}

func onlineOf(t *testing.T, a *App, id int64) onlineRow {
	t.Helper()
	var r onlineRow
	if err := a.S.DB.Table("v2_user").Select("online_count, last_online_at").Where("id = ?", id).Scan(&r).Error; err != nil {
		t.Fatal(err)
	}
	return r
}

func listIDs(t *testing.T, a *App, filter string) map[int64]bool {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/list", a.adminUserList)
	req := httptest.NewRequest(http.MethodGet, "/list?pageSize=500&current=1&filter="+url.QueryEscape(filter), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("list %s => %d %s", filter, w.Code, w.Body.String())
	}
	var body struct {
		Data []struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	out := map[int64]bool{}
	for _, d := range body.Data {
		out[d.ID] = true
	}
	return out
}

func TestIntegrationOnlineCountFollowsDevices(t *testing.T) {
	a := integrationApp(t)
	ctx := context.Background()
	u := newITUser(t, a, "a", false)
	other := newITUser(t, a, "b", false)
	uid := strconv.FormatInt(u.ID, 10)

	// 节点 9001 上报两个设备（带端口的会被规范化）
	a.setUserDevices(ctx, 9001, map[string]any{uid: []any{"1.1.1.1:5000", "2.2.2.2"}})
	if r := onlineOf(t, a, u.ID); r.OnlineCount != 2 || r.LastOnlineAt == nil {
		t.Fatalf("两个设备在线，期望 online_count=2 且 last_online_at 已写入，得到 %+v", r)
	}
	// 节点 9002 上同一个 IP：仍是两台设备
	a.setUserDevices(ctx, 9002, map[string]any{uid: []any{"1.1.1.1"}})
	if r := onlineOf(t, a, u.ID); r.OnlineCount != 2 {
		t.Fatalf("同一 IP 连多个节点只算一台，期望 2，得到 %d", r.OnlineCount)
	}

	// 「在线设备 > 0」筛选能命中在线用户，不会命中没有设备的用户
	ids := listIDs(t, a, `[{"id":"online_count","value":"gt:0"}]`)
	if !ids[u.ID] || ids[other.ID] {
		t.Fatalf("筛选 online_count>0 应只命中在线用户 %d，不含 %d；得到 %v", u.ID, other.ID, ids)
	}
	ids = listIDs(t, a, `[{"id":"online_count","value":"eq:2"}]`)
	if !ids[u.ID] {
		t.Fatalf("筛选 online_count=2 应命中 %d", u.ID)
	}

	// 节点 9001 下线：只剩节点 9002 上的 1.1.1.1
	a.removeNodeDevicesOf(9001, u.ID)
	if r := onlineOf(t, a, u.ID); r.OnlineCount != 1 {
		t.Fatalf("节点 9001 下线后期望 1，得到 %d", r.OnlineCount)
	}
	// 用户自己踢掉最后一个设备
	a.S.Redis.HDel(ctx, devicePrefix+uid, "9002:1.1.1.1")
	a.syncOnlineCount(ctx, u.ID)
	if r := onlineOf(t, a, u.ID); r.OnlineCount != 0 {
		t.Fatalf("设备清空后期望 0，得到 %d", r.OnlineCount)
	}
	if ids := listIDs(t, a, `[{"id":"online_count","value":"gt:0"}]`); ids[u.ID] {
		t.Fatal("设备清空后不应再命中在线筛选")
	}
}

func TestIntegrationCleanupResetsStaleOnlineCount(t *testing.T) {
	a := integrationApp(t)
	stale := newITUser(t, a, "stale", false)
	fresh := newITUser(t, a, "fresh", false)
	noStamp := newITUser(t, a, "nostamp", false)
	old := time.Now().Add(-20 * time.Minute)
	a.S.DB.Exec("UPDATE v2_user SET online_count = 3, last_online_at = ? WHERE id = ?", old, stale.ID)
	a.S.DB.Exec("UPDATE v2_user SET online_count = 2, last_online_at = ? WHERE id = ?", time.Now(), fresh.ID)
	a.S.DB.Exec("UPDATE v2_user SET online_count = 4, last_online_at = NULL WHERE id = ?", noStamp.ID)

	if err := a.jobsEnv().CleanupOnlineStatus(); err != nil {
		t.Fatal(err)
	}
	if r := onlineOf(t, a, stale.ID); r.OnlineCount != 0 {
		t.Errorf("20 分钟没有心跳的用户应清零，得到 %d（旧实现拿整数比较 timestamp 列，永远不会清零）", r.OnlineCount)
	}
	if r := onlineOf(t, a, fresh.ID); r.OnlineCount != 2 {
		t.Errorf("刚上报过的用户不应被清零，得到 %d", r.OnlineCount)
	}
	if r := onlineOf(t, a, noStamp.ID); r.OnlineCount != 0 {
		t.Errorf("没有心跳时间的用户应清零，得到 %d", r.OnlineCount)
	}
}

func TestIntegrationBatchBanKeepsAdmins(t *testing.T) {
	a := integrationApp(t)
	normal := newITUser(t, a, "n1", false)
	normal2 := newITUser(t, a, "n2", false)
	admin := newITUser(t, a, "adm", true)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/ban", a.adminUserBan)
	post := func(body string) int {
		req := httptest.NewRequest(http.MethodPost, "/ban", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	banned := func(id int64) bool {
		var b bool
		a.S.DB.Table("v2_user").Select("banned").Where("id = ?", id).Scan(&b)
		return b
	}

	// 已选范围：包含管理员时，管理员不会被封
	if code := post(fmt.Sprintf(`{"scope":"selected","user_ids":[%d,%d]}`, normal.ID, admin.ID)); code != 200 {
		t.Fatalf("selected => %d", code)
	}
	if !banned(normal.ID) || banned(admin.ID) || banned(normal2.ID) {
		t.Fatalf("只应封禁 normal：normal=%v admin=%v normal2=%v", banned(normal.ID), banned(admin.ID), banned(normal2.ID))
	}

	// 筛选范围：按 id 精确筛选，只作用于命中的用户
	f := fmt.Sprintf(`[{"id":"id","value":"eq:%d"}]`, normal2.ID)
	if code := post(`{"scope":"filtered","filter":` + f + `}`); code != 200 {
		t.Fatalf("filtered => %d", code)
	}
	if !banned(normal2.ID) || banned(admin.ID) {
		t.Fatalf("筛选封禁结果不对：normal2=%v admin=%v", banned(normal2.ID), banned(admin.ID))
	}

	// 含 or 逻辑的筛选也不能越过管理员保护
	or := fmt.Sprintf(`[{"id":"id","value":"eq:%d"},{"id":"id","value":"eq:%d","logic":"or"}]`, normal.ID, admin.ID)
	if code := post(`{"scope":"filtered","filter":` + or + `}`); code != 200 {
		t.Fatalf("filtered(or) => %d", code)
	}
	if banned(admin.ID) {
		t.Fatal("or 逻辑的筛选不应绕过管理员保护")
	}
}

// 前端快捷筛选 / 高级筛选实际发给后端的条件，在真实库上的语义。
func TestIntegrationFrontendFilterSemantics(t *testing.T) {
	a := integrationApp(t)
	noPlan := newITUser(t, a, "noplan", false)   // 无套餐、备注里带冒号、永久（到期为空）
	expired := newITUser(t, a, "expired", false) // 有套餐、已过期
	soon := newITUser(t, a, "soon", false)       // 有套餐、3 天后到期
	nowTs := now()
	a.S.DB.Exec("UPDATE v2_user SET remarks = ? WHERE id = ?", "note:1", noPlan.ID)
	a.S.DB.Exec("UPDATE v2_user SET plan_id = 999999, expired_at = ? WHERE id = ?", nowTs-86400, expired.ID)
	a.S.DB.Exec("UPDATE v2_user SET plan_id = 999999, expired_at = ? WHERE id = ?", nowTs+3*86400, soon.ID)

	check := func(name, filter string, want, notWant []int64) {
		ids := listIDs(t, a, filter)
		for _, id := range want {
			if !ids[id] {
				t.Errorf("%s：应命中用户 %d", name, id)
			}
		}
		for _, id := range notWant {
			if ids[id] {
				t.Errorf("%s：不应命中用户 %d", name, id)
			}
		}
	}
	check("无套餐", `[{"id":"plan_id","value":"null:1"}]`, []int64{noPlan.ID}, []int64{expired.ID, soon.ID})
	check("有套餐", `[{"id":"plan_id","value":"notnull:1"}]`, []int64{expired.ID, soon.ID}, []int64{noPlan.ID})
	check("备注包含(带冒号)", `[{"id":"remarks","value":"like:note:1"}]`, []int64{noPlan.ID}, []int64{expired.ID, soon.ID})
	check("已过期", fmt.Sprintf(`[{"id":"expired_at","value":"lt:%d"},{"id":"expired_at","value":"gt:0"}]`, nowTs),
		[]int64{expired.ID}, []int64{noPlan.ID, soon.ID}) // 永久有效（到期为空）的用户不能算已过期
	check("7 天内到期", fmt.Sprintf(`[{"id":"expired_at","value":"gt:%d"},{"id":"expired_at","value":"lt:%d"}]`, nowTs, nowTs+7*86400),
		[]int64{soon.ID}, []int64{noPlan.ID, expired.ID})
	// 日期「等于」：当天 [起点, 起点+1天)，用的是 gte + lt 两条条件
	day := (nowTs + 3*86400) / 86400 * 86400
	check("到期日期等于", fmt.Sprintf(`[{"id":"expired_at","value":"gte:%d"},{"id":"expired_at","value":"lt:%d"}]`, day-86400, day+2*86400),
		[]int64{soon.ID}, []int64{expired.ID, noPlan.ID})
}
