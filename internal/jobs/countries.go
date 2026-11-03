package jobs

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"xboard/internal/model"
	"xboard/internal/sub"
)

// 「可用国家」快照：给没有会员的用户展示当前能提供服务的国家（不含主机地址与凭据）。
//
// 一天更新一次（refresh:countries，每天 04:00），管理员可随时手动更新。
// 快照存放在 Redis，缺失（首次运行或 Redis 被清空）时由请求方按需重建。

const (
	countrySnapshotKey   = "country_snapshot"
	countryRefreshingKey = "country_refreshing"

	// DefaultCountryRefreshTime 是每日自动更新的缺省时刻；后台设置 country_refresh_time 可改（HH:MM，服务器本地时区）。
	DefaultCountryRefreshTime = "04:00"
)

// NormalizeHHMM 把 "4:00" / "04:00" 规整为 "04:00"；格式不对返回 false。
func NormalizeHHMM(s string) (string, bool) {
	t, err := time.Parse("15:04", strings.TrimSpace(s))
	if err != nil {
		return "", false
	}
	return t.Format("15:04"), true
}

// CountryRefreshTime 返回每日自动更新的时刻；设置缺失或格式不对时用缺省值，保证调度器永远有合法时刻。
func (e *Env) CountryRefreshTime() string {
	if v, ok := NormalizeHHMM(e.Set.Get("country_refresh_time", DefaultCountryRefreshTime)); ok {
		return v
	}
	return DefaultCountryRefreshTime
}

// defaultCountryTag 是 country_tags 里只写国家码、没写标签文字时用的文字。
const defaultCountryTag = "推荐"

// ParseCountryTags 解析后台设置 country_tags，返回 大写国家码 → 标签文字。
// 格式为 "HK:推荐,JP:高速"：逗号或换行分隔，中英文标点都可以；只写 "HK" 时标签为「推荐」。
// 国家码不是两位字母的项直接跳过，写错一项不影响其它项。
func ParseCountryTags(raw string) map[string]string {
	tags := map[string]string{}
	items := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == '\r'
	})
	for _, it := range items {
		it = strings.ReplaceAll(it, "：", ":")
		code, text, _ := strings.Cut(it, ":")
		code = strings.ToUpper(strings.TrimSpace(code))
		if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			text = defaultCountryTag
		}
		tags[code] = text
	}
	return tags
}

// CountryTags 返回当前设置的国家标签。标签不进快照，读接口时现取，改设置后不用等每日更新。
func (e *Env) CountryTags() map[string]string {
	return ParseCountryTags(e.Set.Get("country_tags", ""))
}

// CountryEntry 是一个国家的汇总。
type CountryEntry struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Flag          string `json:"flag"`
	Nodes         int    `json:"nodes"`       // 当前可用的节点数
	TotalNodes    int    `json:"total_nodes"` // 该国家已显示的节点总数
	BestLatencyMs *int64 `json:"best_latency_ms"`
}

// CountrySnapshot 是某一时刻的可用国家列表。
type CountrySnapshot struct {
	UpdatedAt int64          `json:"updated_at"`
	Source    string         `json:"source"` // daily | manual | lazy
	Countries []CountryEntry `json:"countries"`
	// UnknownCountry 为已显示且可用、但无法判定国家的节点数（不会出现在列表里）。
	UnknownCountry int `json:"unknown_country"`
	// Kept 为 true 表示本次计算没有任何可用节点，疑似探测异常，已沿用上一份快照。
	Kept bool `json:"kept"`
}

// nodeAvailable 对应 ServerService::isNodeAvailable：在线上报，或最近一次测速成功且未过期。
// stale 为负数（测速已关闭）时与订阅一致（sub.NodeAvailable）：不判断过期，以最后一次结论为准，
// 从未测过的节点视为可用——否则测速一关，过了有效期所有国家都会变成不可用。
func (e *Env) nodeAvailable(s model.Server, stale int64) bool {
	if e.nodeIsOnline(s) {
		return true
	}
	if stale < 0 {
		return s.LastLatencyAt == nil || s.LatencyMs != nil
	}
	return s.LatencyMs != nil && s.LastLatencyAt != nil && e.nowTs()-*s.LastLatencyAt <= stale
}

// availabilityStale 返回判断可用性的有效期；测速关闭时返回 -1（不判断过期），与订阅下发一致。
func (e *Env) availabilityStale() int64 {
	if !e.Set.Bool("server_latency_probe_enable", true) {
		return -1
	}
	return e.effectiveStaleSeconds()
}

// BuildCountrySnapshot 按当前数据计算可用国家：只统计已显示的主节点。
func (e *Env) BuildCountrySnapshot(source string) CountrySnapshot {
	var nodes []model.Server
	e.DB.Where("parent_id IS NULL AND `show` = 1").Find(&nodes)
	stale := e.availabilityStale()
	return summarizeCountries(nodes, func(n model.Server) bool { return e.nodeAvailable(n, stale) }, e.nowTs(), source)
}

// summarizeCountries 把节点按国家汇总；avail 判断节点当前是否可用。纯函数，便于测试。
func summarizeCountries(nodes []model.Server, avail func(model.Server) bool, now int64, source string) CountrySnapshot {
	byCC := map[string]*CountryEntry{}
	unknown := 0
	for _, n := range nodes {
		cc := ""
		if n.CountryCode != nil {
			cc = strings.ToUpper(strings.TrimSpace(*n.CountryCode))
		}
		if cc == "" {
			cc = ParseCountryFromName(n.Name)
		}
		ok := avail(n)
		if cc == "" {
			if ok {
				unknown++
			}
			continue
		}
		ent := byCC[cc]
		if ent == nil {
			ent = &CountryEntry{Code: cc, Name: sub.CountryNameZH(cc), Flag: sub.CountryFlag(cc)}
			byCC[cc] = ent
		}
		ent.TotalNodes++
		if ok {
			ent.Nodes++
			if n.LatencyMs != nil && (ent.BestLatencyMs == nil || *n.LatencyMs < *ent.BestLatencyMs) {
				v := *n.LatencyMs
				ent.BestLatencyMs = &v
			}
		}
	}
	snap := CountrySnapshot{UpdatedAt: now, Source: source, Countries: []CountryEntry{}, UnknownCountry: unknown}
	for _, ent := range byCC {
		if ent.Nodes > 0 { // 只列出有可用节点的国家
			snap.Countries = append(snap.Countries, *ent)
		}
	}
	sort.Slice(snap.Countries, func(i, j int) bool {
		a, b := snap.Countries[i], snap.Countries[j]
		if a.Nodes != b.Nodes {
			return a.Nodes > b.Nodes
		}
		return a.Code < b.Code
	})
	return snap
}

func (e *Env) storeCountrySnapshot(s CountrySnapshot) {
	if e.Redis == nil {
		return
	}
	b, _ := json.Marshal(s)
	_ = e.Redis.Set(context.Background(), countrySnapshotKey, string(b), 0)
}

// LoadCountrySnapshot 读取已保存的快照。
func (e *Env) LoadCountrySnapshot() (CountrySnapshot, bool) {
	var s CountrySnapshot
	if e.Redis == nil {
		return s, false
	}
	raw, ok, err := e.Redis.Get(context.Background(), countrySnapshotKey)
	if err != nil || !ok {
		return s, false
	}
	if json.Unmarshal([]byte(raw), &s) != nil {
		return s, false
	}
	if s.Countries == nil {
		s.Countries = []CountryEntry{}
	}
	return s, true
}

// CountrySnapshotOrBuild 供用户接口使用：有快照直接返回，没有（首次/被清空）就按需计算并保存。
func (e *Env) CountrySnapshotOrBuild() CountrySnapshot {
	if s, ok := e.LoadCountrySnapshot(); ok {
		return s
	}
	s := e.BuildCountrySnapshot("lazy")
	e.storeCountrySnapshot(s)
	return s
}

// RefreshCountries 重新计算并保存快照。
// 若本次没有任何国家可用而上一份快照有内容，多半是探测异常（网络、sing-box 等），保留上一份，避免一次故障清空列表。
func (e *Env) RefreshCountries(source string) CountrySnapshot {
	return e.keepOrStore(e.BuildCountrySnapshot(source))
}

// keepOrStore 保存新快照；新快照为空而旧快照有内容时沿用旧快照。
func (e *Env) keepOrStore(snap CountrySnapshot) CountrySnapshot {
	if len(snap.Countries) == 0 {
		if prev, ok := e.LoadCountrySnapshot(); ok && len(prev.Countries) > 0 {
			prev.Kept = true
			e.storeCountrySnapshot(prev)
			return prev
		}
	}
	e.storeCountrySnapshot(snap)
	return snap
}

// CountriesRefreshing 是否有手动更新正在进行。
func (e *Env) CountriesRefreshing() bool {
	if e.Redis == nil {
		return false
	}
	_, ok, _ := e.Redis.Get(context.Background(), countryRefreshingKey)
	return ok
}

// RefreshCountriesManual 手动更新：先对全部节点测速（忽略自动测速开关），再生成快照。
// 同一时间只允许一个；会阻塞到完成，调用方应放在 goroutine 里。返回 false 表示已有更新在进行。
func (e *Env) RefreshCountriesManual(probe bool) (CountrySnapshot, bool) {
	if e.Redis != nil {
		ok, _ := e.Redis.SetNX(context.Background(), countryRefreshingKey, 1, 30*time.Minute)
		if !ok {
			return CountrySnapshot{}, false
		}
		defer func() { _ = e.Redis.Del(context.Background(), countryRefreshingKey) }()
	}
	if probe {
		_ = e.CheckServerLatency(nil)
	}
	return e.RefreshCountries("manual"), true
}
