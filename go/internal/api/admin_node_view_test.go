package api

import (
	"encoding/json"
	"testing"

	"xboard/internal/model"
	"xboard/internal/setting"
)

func ip(v int64) *int64   { return &v }
func sp(v string) *string { return &v }

func deliveredApp(cfg map[string]string) *App {
	return &App{Set: setting.NewFromMap(cfg)}
}

func TestEffectiveStaleSecondsFollowsProbeInterval(t *testing.T) {
	// 探测间隔 10 分钟：有效期至少 15 分钟，否则两次探测之间节点会被误判为过期。
	a := deliveredApp(map[string]string{"server_latency_probe_enable": "1", "server_latency_probe_interval": "10", "server_latency_stale_seconds": "600"})
	if got := a.effectiveStaleSeconds(); got != 900 {
		t.Fatalf("got %d want 900", got)
	}
	// 关闭自动探测后只看配置值
	a = deliveredApp(map[string]string{"server_latency_probe_enable": "0", "server_latency_probe_interval": "10", "server_latency_stale_seconds": "600"})
	if got := a.effectiveStaleSeconds(); got != 600 {
		t.Fatalf("got %d want 600", got)
	}
}

// g1 是「分配了权限组 1」的 group_ids；没有权限组的节点不会出现在任何人的订阅里。
var g1 = json.RawMessage(`[1]`)

func TestComputeDeliveredIDsMatchesSubscriptionWithoutDedup(t *testing.T) {
	// 未开启按国家去重时，订阅不按测速结果筛选：显示、有权限组、未超流量的都下发（含子节点）。
	now0 := now()
	limit := ip(1000)
	rows := []model.Server{
		{ID: 1, Show: true, GroupIDs: g1, LatencyMs: ip(100), LastLatencyAt: ip(now0 - 30)}, // 测通 → 下发
		{ID: 2, Show: true, GroupIDs: g1}, // 从未测速 → 订阅照样下发
		{ID: 3, Show: false, GroupIDs: g1, LatencyMs: ip(100), LastLatencyAt: ip(now0 - 30)},                                       // 隐藏 → 不下发
		{ID: 4, Show: true, GroupIDs: g1, ParentID: ip(1), LatencyMs: ip(100), LastLatencyAt: ip(now0 - 30)},                       // 子节点 → 订阅同样下发
		{ID: 5, Show: true, GroupIDs: g1, LatencyMs: ip(100), LastLatencyAt: ip(now0 - 30), TransferEnable: limit, U: 600, D: 500}, // 超流量 → 不下发
		{ID: 6, Show: true, LatencyMs: ip(100), LastLatencyAt: ip(now0 - 30)},                                                      // 没有权限组 → 不下发
		{ID: 7, Show: true, GroupIDs: json.RawMessage(`[]`)},                                                                       // 权限组为空 → 不下发
		{ID: 8, Show: true, GroupIDs: g1, LastLatencyAt: ip(now0 - 30)},                                                            // 测过不通 → 未去重时仍下发
	}
	got := deliveredApp(nil).computeDeliveredIDs(rows, make([]nodeRuntime, len(rows)))
	want := map[int64]bool{1: true, 2: true, 4: true, 8: true}
	for _, r := range rows {
		if got[r.ID] != want[r.ID] {
			t.Errorf("node %d delivered=%v want %v", r.ID, got[r.ID], want[r.ID])
		}
	}
}

func TestComputeDeliveredIDsDedupByCountry(t *testing.T) {
	now0 := now()
	mk := func(id int64, cc string, lat int64) model.Server {
		return model.Server{ID: id, Show: true, GroupIDs: g1, CountryCode: sp(cc), LatencyMs: ip(lat), LastLatencyAt: ip(now0 - 10)}
	}
	rows := []model.Server{mk(1, "HK", 200), mk(2, "HK", 90), mk(3, "JP", 150), {ID: 4, Show: true, GroupIDs: g1, LatencyMs: ip(80), LastLatencyAt: ip(now0 - 10)}}
	got := deliveredApp(map[string]string{"server_dedup_by_country": "1"}).computeDeliveredIDs(rows, make([]nodeRuntime, len(rows)))
	// 每国只留延迟最低的一个；没有国家码的保留
	for id, want := range map[int64]bool{1: false, 2: true, 3: true, 4: true} {
		if got[id] != want {
			t.Errorf("node %d delivered=%v want %v", id, got[id], want)
		}
	}
}

func TestComputeDeliveredIDsProbeDisabledKeepsLastResult(t *testing.T) {
	// 截图里的场景：去重开启、测速已关闭、测速结果是 16 分钟前的（超过 10 分钟有效期）。
	// 测速关闭后结果不会再刷新，不能按时间判过期，否则所有节点都显示「不下发」。
	now0 := now()
	old := now0 - 16*60
	mk := func(id int64, cc string, lat *int64) model.Server {
		return model.Server{ID: id, Show: true, GroupIDs: g1, CountryCode: sp(cc), LatencyMs: lat, LastLatencyAt: ip(old)}
	}
	rows := []model.Server{mk(1, "HK", ip(128)), mk(2, "HK", ip(61)), mk(3, "JP", ip(102)), mk(4, "TW", nil),
		{ID: 5, Show: true, GroupIDs: g1, CountryCode: sp("SG")}} // 5: 从未测过
	cfg := map[string]string{"server_dedup_by_country": "1", "server_latency_probe_enable": "", "server_latency_stale_seconds": "600"}
	got := deliveredApp(cfg).computeDeliveredIDs(rows, make([]nodeRuntime, len(rows)))
	// HK 取上次延迟最低的 2；JP 下发；TW 上次测过不通 → 不下发；SG 从未测过 → 视为可用
	for id, want := range map[int64]bool{1: false, 2: true, 3: true, 4: false, 5: true} {
		if got[id] != want {
			t.Errorf("node %d delivered=%v want %v", id, got[id], want)
		}
	}
	// 同样的数据但测速开启：结果已过期，按订阅的兜底规则退回全部候选（不会出现空订阅）
	cfg["server_latency_probe_enable"] = "1"
	got = deliveredApp(cfg).computeDeliveredIDs(rows, make([]nodeRuntime, len(rows)))
	for _, r := range rows {
		if !got[r.ID] {
			t.Errorf("probe enabled + all stale: node %d should fall back to delivered", r.ID)
		}
	}
}

func TestNodeViewExposesProbeAndRuntimeFields(t *testing.T) {
	// 回归：节点列表曾漏掉 latency_ms / last_latency_at，导致测速列永远显示「未测速」。
	a := deliveredApp(nil)
	s := model.Server{ID: 9, Type: "vless", GroupIDs: json.RawMessage("[]"), LatencyMs: ip(120), LastLatencyAt: ip(1700000000), ExitIP: sp("1.2.3.4")}
	rt := nodeRuntime{LastCheckAt: ip(1700000001), Online: 3, IsOnline: 1, AvailableStatus: nodeStatusOnline,
		LoadStatus: map[string]any{"cpu": 12.0}, OnlineConn: 7}
	v := a.nodeView(s, rt, true)
	for _, k := range []string{"latency_ms", "last_latency_at", "exit_ip", "online", "is_online", "available_status",
		"load_status", "metrics", "online_conn", "last_check_at", "last_push_at", "is_delivered"} {
		if _, ok := v[k]; !ok {
			t.Errorf("node view must expose %q", k)
		}
	}
	if v["latency_ms"].(*int64) == nil || *v["latency_ms"].(*int64) != 120 || v["online"] != int64(3) || v["is_delivered"] != true {
		t.Errorf("unexpected values: %+v", v)
	}
}

func TestProbeFresh(t *testing.T) {
	n := now()
	cases := []struct {
		name string
		s    model.Server
		want bool
	}{
		{"连通且新鲜", model.Server{LatencyMs: ip(80), LastLatencyAt: ip(n - 10)}, true},
		{"连通但已过期", model.Server{LatencyMs: ip(80), LastLatencyAt: ip(n - 5000)}, false},
		{"测过但不通", model.Server{LastLatencyAt: ip(n - 10)}, false},
		{"从未测速", model.Server{}, false},
	}
	for _, c := range cases {
		if got := probeFresh(c.s, n, 600); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
