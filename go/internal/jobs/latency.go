package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"xboard/internal/model"
)

// 节点测速（TCP 握手模式）。对应 check:server-latency 中 node_probe_mode 为 tcp 的分支。
// sing-box 实跑探测需要外部二进制，Go 版暂未实现（见 PROGRESS.md）。

const latencyStatusKey = "server_latency_probe_status"

// latencyStopKey 为「停止测速」标记。放在 Redis 里而不是进程内，是因为多实例部署时
// 测速可能跑在另一台机器的调度器上。
const latencyStopKey = "server_latency_probe_stop"

// LatencyStatus 对应 CheckServerLatency::STATUS_CACHE_KEY 中的状态结构。
type LatencyStatus struct {
	Running    bool   `json:"running"`
	Queued     bool   `json:"queued,omitempty"`
	Total      int    `json:"total"`
	Done       int    `json:"done"`
	OK         int    `json:"ok"`
	Current    string `json:"current"`
	StartedAt  int64  `json:"started_at"`
	Mode       string `json:"mode,omitempty"`
	FinishedAt *int64 `json:"finished_at"`
	UpdatedAt  int64  `json:"updated_at"`
	// Stopped 为 true 表示本轮被管理员手动停止（只测了 Done 个节点）。
	Stopped bool `json:"stopped,omitempty"`
}

// probePort 解析端口（支持 "10000-20000" 范围，随机取一个）。
func probePort(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if lo, hi, ok := strings.Cut(raw, "-"); ok {
		a, err1 := strconv.Atoi(lo)
		b, err2 := strconv.Atoi(hi)
		if err1 != nil || err2 != nil || a > b {
			return 0, false
		}
		return a + rand.Intn(b-a+1), true
	}
	p, err := strconv.Atoi(raw)
	return p, err == nil && p > 0
}

// tcpLatency 测量一次 TCP 握手耗时（毫秒）。
func tcpLatency(host string, port int, timeout time.Duration) (int64, bool) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	if err != nil {
		return 0, false
	}
	_ = conn.Close()
	return time.Since(start).Milliseconds(), true
}

func (e *Env) writeLatencyStatus(s LatencyStatus) {
	if e.Redis == nil {
		return
	}
	s.UpdatedAt = e.nowTs()
	b, _ := json.Marshal(s)
	_ = e.Redis.Set(context.Background(), latencyStatusKey, string(b), time.Hour)
}

// CheckServerLatency 手动触发：忽略 server_latency_probe_enable，其余同 check:server-latency --manual。
func (e *Env) CheckServerLatency(ids []int64) error { return e.runLatency(ids, true) }

// effectiveStaleSeconds 对应 ServerService::effectiveStaleSeconds：开启自动测速时窗口至少覆盖 1.5 个周期。
func (e *Env) effectiveStaleSeconds() int64 {
	stale := int64(e.Set.Int("server_latency_stale_seconds", 600))
	if e.Set.Bool("server_latency_probe_enable", true) {
		iv := e.Set.Int("server_latency_probe_interval", 3)
		if iv < 1 {
			iv = 1
		}
		if w := int64(float64(iv*60) * 1.5); w > stale {
			stale = w
		}
	}
	return stale
}

// nodeIsOnline 与订阅下发一致：Redis 中 5 分钟内有上报。
func (e *Env) nodeIsOnline(s model.Server) bool {
	if e.Redis == nil {
		return false
	}
	key := fmt.Sprintf("SERVER_%s_LAST_CHECK_AT_%d", strings.ToUpper(s.Type), s.ID)
	v, ok, _ := e.Redis.Get(context.Background(), key)
	if !ok {
		return false
	}
	ts, _ := strconv.ParseInt(v, 10, 64)
	return ts != 0 && e.nowTs()-300 <= ts
}

type latencyOutcome struct {
	latency *int64
	cc      string
	working bool
}

func (e *Env) finishLatencyStatus(st LatencyStatus) {
	st.Running, st.Queued, st.Current = false, false, ""
	fin := e.nowTs()
	st.FinishedAt = &fin
	e.writeLatencyStatus(st)
}

func (e *Env) runLatency(ids []int64, manual bool) error {
	st := LatencyStatus{Mode: e.Set.Get("node_probe_mode", "singbox"), StartedAt: e.nowTs()}
	if !manual && !e.Set.Bool("server_latency_probe_enable", true) {
		e.finishLatencyStatus(st)
		return nil
	}
	auto := e.Set.Bool("server_auto_manage_visibility", false)
	q := e.DB.Where("parent_id IS NULL")
	if !auto {
		q = q.Where("`show` = 1")
	}
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	}
	var servers []model.Server
	if err := q.Find(&servers).Error; err != nil {
		e.finishLatencyStatus(st)
		return err
	}
	if len(servers) == 0 {
		e.finishLatencyStatus(st)
		return nil
	}
	// 新一轮开始时清掉上一轮遗留的停止标记，避免它把这一轮也停掉。
	e.clearLatencyStop()
	st.Running, st.Total = true, len(servers)
	e.writeLatencyStatus(st)
	defer func() { e.finishLatencyStatus(st) }()

	stale := e.effectiveStaleSeconds()
	probe := map[int64]latencyOutcome{}
	for i := range servers {
		// 每个节点开始前检查停止标记；正在测的那个节点会测完（sing-box 模式最多十几秒）。
		if e.LatencyStopRequested() {
			st.Stopped = true
			break
		}
		s := &servers[i]
		st.Current = s.Name
		e.writeLatencyStatus(st)
		r := e.ProbeNode(*s, ProbeOptions{})
		now := e.nowTs()
		// 「没法测」不是节点不通：保留上次结果，不动延迟字段。
		if !r.Untestable {
			s.LatencyMs, s.LastLatencyAt = r.LatencyMs, i64Ptr(now)
			var exit, lat any
			if r.ExitIP != nil {
				exit = *r.ExitIP
			}
			if r.LatencyMs != nil {
				lat = *r.LatencyMs
			}
			// UpdateColumns 不触碰 updated_at：测速字段频繁变化，不能每次都触发订阅更新。
			e.DB.Model(&model.Server{}).Where("id = ?", s.ID).
				UpdateColumns(map[string]any{"latency_ms": lat, "exit_ip": exit, "last_latency_at": now})
		}
		if r.CountryCode != nil && *r.CountryCode != "" {
			s.CountryCode = r.CountryCode
			e.DB.Model(&model.Server{}).Where("id = ?", s.ID).UpdateColumn("country_code", *r.CountryCode)
		}
		// 可用性：agent 在线，或实测连通且结果仍新鲜。
		working := e.nodeIsOnline(*s) || (s.LatencyMs != nil && s.LastLatencyAt != nil && now-*s.LastLatencyAt <= stale)
		cc := ""
		if s.CountryCode != nil {
			cc = *s.CountryCode
		}
		probe[s.ID] = latencyOutcome{latency: s.LatencyMs, cc: cc, working: working}
		st.Done++
		if working {
			st.OK++
		}
		e.writeLatencyStatus(st)
	}

	// 手动停止：只测了一部分，不能据此做自动显隐或判定「0 个可用」。
	if st.Stopped {
		e.clearLatencyStop()
		log.Printf("check:server-latency: 已手动停止（%d/%d）", st.Done, st.Total)
		return nil
	}
	// 自动显隐依赖「探测了全部节点」；只测部分节点时不执行，否则会把没测到的节点误隐藏。
	if auto && len(ids) == 0 {
		e.applyVisibility(servers, probe)
	}
	// 全量探测且 0 个可用节点：BalalaSub 插件监听 server.latency.no_available 写入导入信号。
	if len(ids) == 0 && st.Total > 0 && st.OK == 0 {
		e.noAvailableHook()
	}
	return nil
}

// applyVisibility 每个国家只显示延迟最低的可达节点，无国家的可达节点保留显示，其余隐藏。
func (e *Env) applyVisibility(servers []model.Server, probe map[int64]latencyOutcome) {
	type best struct {
		id  int64
		lat int64
	}
	byCC := map[string]best{}
	for _, s := range servers {
		p := probe[s.ID]
		if !p.working || p.cc == "" {
			continue
		}
		cc := strings.ToUpper(p.cc)
		lat := int64(math.MaxInt64)
		if p.latency != nil {
			lat = *p.latency
		}
		if cur, ok := byCC[cc]; !ok || lat < cur.lat {
			byCC[cc] = best{s.ID, lat}
		}
	}
	show := map[int64]bool{}
	for _, b := range byCC {
		show[b.id] = true
	}
	targets := map[int64]bool{}
	anyShown := false
	for _, s := range servers {
		p := probe[s.ID]
		t := show[s.ID] || (p.working && p.cc == "")
		targets[s.ID] = t
		anyShown = anyShown || t
	}
	// 安全网：一个都不显示多半是本轮探测整体异常，保持现状以免订阅下发为空。
	if !anyShown {
		log.Printf("check:server-latency: 自动显隐已跳过：本轮没有任何可达节点（疑似探测异常）")
		return
	}
	changed := 0
	for _, s := range servers {
		if s.Show != targets[s.ID] {
			e.DB.Model(&model.Server{}).Where("id = ?", s.ID).Update("show", targets[s.ID])
			changed++
		}
	}
	if changed > 0 {
		e.broadcastSubscriptionUpdated("visibility")
	}
	log.Printf("check:server-latency: 自动显隐完成，变更 %d 个", changed)
}

// ClientPusher 由 Redis 推送器实现：向在线客户端广播「订阅已更新」。
type ClientPusher interface {
	BroadcastSubscriptionUpdated(reason string) error
}

// NotifyClients 通知在线客户端重新拉取订阅（受「客户端推送」开关控制）。
func (e *Env) NotifyClients(reason string) { e.broadcastSubscriptionUpdated(reason) }

func (e *Env) broadcastSubscriptionUpdated(reason string) {
	if !e.Set.Bool("client_ws_enable", false) {
		return
	}
	if cp, ok := e.Push.(ClientPusher); ok {
		_ = cp.BroadcastSubscriptionUpdated(reason)
	}
}

// noAvailableHook 对应插件对 server.latency.no_available 的监听：开启 auto_reimport_on_empty 时写入导入信号。
func (e *Env) noAvailableHook() {
	enabled, cfg := e.balalaPluginRow()
	if !enabled || !balalaTruthy(cfg["auto_reimport_on_empty"]) || e.Redis == nil {
		return
	}
	_ = e.Redis.Set(context.Background(), balalaRequestKey, time.Now().Format(time.RFC3339), time.Hour)
	log.Printf("balala_sub: 测速无可用节点，已写入导入信号，待定时任务执行")
}

// ScheduledLatency 由调度器每分钟调用：按 server_latency_probe_interval（分钟）节流，抢到时间片的实例执行。
func (e *Env) ScheduledLatency() error {
	if !e.Set.Bool("server_latency_probe_enable", true) {
		return nil
	}
	interval := e.Set.Int("server_latency_probe_interval", 3)
	if interval < 1 {
		interval = 1
	}
	if e.Redis != nil {
		ok, err := e.Redis.SetNX(context.Background(), "server_latency_next", 1, time.Duration(interval)*time.Minute)
		if err != nil || !ok {
			return err
		}
	}
	return e.runLatency(nil, false)
}

// LatencyStatusOf 读取探测状态（后台轮询使用）。
func (e *Env) LatencyStatusOf() (LatencyStatus, bool) {
	var st LatencyStatus
	if e.Redis == nil {
		return st, false
	}
	raw, ok, err := e.Redis.Get(context.Background(), latencyStatusKey)
	if err != nil || !ok {
		return st, false
	}
	_ = json.Unmarshal([]byte(raw), &st)
	return st, true
}

// RequestLatencyStop 请求停止正在进行的测速，返回当前是否有测速在运行。
// 标记带 10 分钟有效期，没被消费时自动失效。
func (e *Env) RequestLatencyStop() bool {
	st, ok := e.LatencyStatusOf()
	running := ok && st.Running && e.nowTs()-st.UpdatedAt < 120
	if running && e.Redis != nil {
		_ = e.Redis.Set(context.Background(), latencyStopKey, "1", 10*time.Minute)
	}
	return running
}

// LatencyStopRequested 是否已请求停止（测速循环与状态接口使用）。
func (e *Env) LatencyStopRequested() bool {
	if e.Redis == nil {
		return false
	}
	_, ok, err := e.Redis.Get(context.Background(), latencyStopKey)
	return err == nil && ok
}

func (e *Env) clearLatencyStop() {
	if e.Redis != nil {
		_ = e.Redis.Del(context.Background(), latencyStopKey)
	}
}
