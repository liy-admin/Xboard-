package sub

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"xboard/internal/model"
)

// Ctx 是一次订阅生成的上下文（对应 AbstractProtocol 的构造参数）。
type Ctx struct {
	User          *model.User
	Servers       []Map
	ClientName    string
	ClientVersion string
	UserAgent     string
	AppName       string
	AppURL        string
	Host          string // 订阅请求的 Host，用于生成直连规则
	// SubscribeURL 生成订阅地址（对应 Helper::getSubscribeUrl），domain 为空时使用后台配置的地址。
	SubscribeURL func(token, domain string) string
	// Template 返回订阅模板内容（v2_subscribe_templates），空则用内置默认。
	Template func(name string) string
}

// requirements: client(或 "*") -> 协议类型 -> 字段路径 -> 取值 -> 最低版本。
// 特殊键 "__strict__"="1" 表示严格白名单（取值缺失或不在表内即不兼容）；base_version 不参与过滤。
type requirements = map[string]map[string]map[string]map[string]string

func filterServers(ctx *Ctx, allowed map[string]bool, req requirements) []Map {
	_, hasGlobal := req["*"]
	_, hasClient := req[ctx.ClientName]
	out := make([]Map, 0, len(ctx.Servers))
	for _, s := range ctx.Servers {
		if len(allowed) > 0 && !allowed[Str(s["type"])] {
			continue
		}
		skip := ((ctx.ClientName == "" || ctx.ClientVersion == "") && !hasGlobal) || (!hasGlobal && !hasClient)
		if !skip {
			t := Str(s["type"])
			if r, ok := req["*"][t]; ok && !compatible(ctx.ClientVersion, r, s) {
				continue
			}
			if r, ok := req[ctx.ClientName][t]; ok && !compatible(ctx.ClientVersion, r, s) {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

func compatible(clientVer string, rules map[string]map[string]string, s Map) bool {
	for field, valMap := range rules {
		if field == "base_version" || field == "incompatible" {
			continue
		}
		strict := valMap["__strict__"] == "1"
		actual := Get(s, field)
		switch actual.(type) {
		case string, float64, int, int64:
		default:
			if strict {
				return false
			}
			continue
		}
		minVer, ok := valMap[Str(actual)]
		if !ok {
			if strict {
				return false
			}
			continue
		}
		if minVer != "0.0.0" && versionCompare(clientVer, minVer) < 0 {
			return false
		}
	}
	return true
}

// AvailableServers 对应 ServerService::getAvailableServers，并附加 online/is_online 等运行态字段。
func AvailableServers(ctx context.Context, db *gorm.DB, rdb *redis.Client, u *model.User) ([]Map, error) {
	var rows []model.Server
	gid := int64(0)
	if u.GroupID != nil {
		gid = *u.GroupID
	}
	err := db.Where(`JSON_CONTAINS(group_ids, ?) AND `+"`show` = 1"+
		` AND (transfer_enable IS NULL OR transfer_enable = 0 OR u + d < transfer_enable)`, strconv.FormatInt(gid, 10)).
		Order("sort ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	// 子节点的 key 材料与运行态取自父节点
	byID := map[int64]model.Server{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	out := make([]Map, 0, len(rows))
	now := time.Now().Unix()
	for _, r := range rows {
		parent := r
		if r.ParentID != nil {
			if p, ok := byID[*r.ParentID]; ok {
				parent = p
			} else {
				var p model.Server
				if db.First(&p, *r.ParentID).Error == nil {
					parent = p
				}
			}
		}
		stateID := r.ID
		if r.ParentID != nil {
			stateID = *r.ParentID
		}
		typ := strings.ToUpper(r.Type)
		lastCheck, _ := rdb.Get(ctx, fmt.Sprintf("SERVER_%s_LAST_CHECK_AT_%d", typ, stateID)).Int64()
		online, _ := rdb.Get(ctx, fmt.Sprintf("SERVER_%s_ONLINE_USER_%d", typ, stateID)).Int()
		isOnline := 0
		if lastCheck != 0 && now-300 <= lastCheck {
			isOnline = 1
		}

		ps := Map{}
		if len(r.ProtocolSettings) > 0 {
			_ = json.Unmarshal(r.ProtocolSettings, &ps)
		}
		ps = NormalizeProtocolSettings(r.Type, ps)
		var tags []any
		_ = json.Unmarshal(r.Tags, &tags)

		port := any(r.Port)
		var ports any
		if strings.Contains(r.Port, "-") {
			ports = r.Port
			port = randomPort(r.Port)
		} else {
			n, _ := strconv.Atoi(r.Port)
			port = n
		}
		code := ""
		if r.Code != nil {
			code = *r.Code
		}
		m := Map{
			"id": r.ID, "type": r.Type, "code": code, "name": r.Name, "host": r.Host,
			"port": port, "server_port": r.ServerPort, "protocol_settings": ps, "tags": tags,
			"rate": currentRate(r), "online": online, "is_online": isOnline,
			"password": serverPassword(r, parent, u, ps),
		}
		m["cache_key"] = nodeCacheKey(m, r.UpdatedAt)
		if ports != nil {
			m["ports"] = ports
		}
		if r.Sort != nil {
			m["sort"] = *r.Sort
		}
		if r.CountryCode != nil {
			m["country_code"] = *r.CountryCode
		}
		if r.LatencyMs != nil {
			m["latency_ms"] = *r.LatencyMs
		}
		if r.LastLatencyAt != nil {
			m["last_latency_at"] = *r.LastLatencyAt
		}
		out = append(out, m)
	}
	return out, nil
}

func randomPort(rng string) int {
	parts := strings.SplitN(rng, "-", 2)
	lo, _ := strconv.Atoi(parts[0])
	hi := lo
	if len(parts) == 2 {
		hi, _ = strconv.Atoi(parts[1])
	}
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo + rand.Intn(hi-lo+1)
}

func currentRate(s model.Server) float64 {
	if !s.RateTimeEnable {
		return s.Rate
	}
	var ranges []struct {
		Start string  `json:"start"`
		End   string  `json:"end"`
		Rate  float64 `json:"rate"`
	}
	_ = json.Unmarshal(s.RateTimeRanges, &ranges)
	hm := time.Now().Format("15:04")
	for _, r := range ranges {
		if hm >= r.Start && hm <= r.End {
			return r.Rate
		}
	}
	return s.Rate
}

var ssCiphers = map[string]int{
	"2022-blake3-aes-128-gcm": 16, "2022-blake3-aes-256-gcm": 32, "2022-blake3-chacha20-poly1305": 32,
}

// serverPassword 对应 Server::generateServerPassword。
func serverPassword(s, parent model.Server, u *model.User, ps Map) string {
	auth := u.UUID
	if s.Code != nil && *s.Code != "" {
		auth = *s.Code
	}
	if s.Type != "shadowsocks" {
		return auth
	}
	size, ok := ssCiphers[Str(ps["cipher"])]
	if !ok {
		return auth
	}
	key := md5hex(strconv.FormatInt(parent.CreatedAt.Unix(), 10))
	if len(key) > size {
		key = key[:size]
	}
	ukey := auth
	if len(ukey) > size {
		ukey = ukey[:size]
	}
	return b64(key) + ":" + b64(ukey)
}

// PickBestPerCountry 对应 ServerService::pickBestPerCountry（国家名映射需另行补全）。
// NoExpiry 作为 staleSec 传入，表示测速已关闭：测速结果不会再刷新，不能再按时间判断过期，
// 否则过了有效期所有节点都会被判为不可用。此时以最后一次测速结论为准，从未测过的节点视为可用。
const NoExpiry = -1

// probedReachable 判断节点是否有「仍然有效的测通结果」（用于在同一国家里按延迟挑最优）。
func probedReachable(s Map, now int64, staleSec int) bool {
	if s["latency_ms"] == nil || Int(s["last_latency_at"]) == 0 {
		return false
	}
	return staleSec < 0 || now-int64(Int(s["last_latency_at"])) <= int64(staleSec)
}

// NodeAvailable 判断节点是否可下发：agent 在线，或测速结论为可达。
// staleSec 为 NoExpiry 时不判断过期，且从未测过（没有任何结论）的节点视为可用；
// 只有「测过且不通」（last_latency_at 有值、latency_ms 为空）才判为不可用。
func NodeAvailable(s Map, now int64, staleSec int) bool {
	if Int(s["is_online"]) != 0 {
		return true
	}
	if staleSec < 0 && Int(s["last_latency_at"]) == 0 {
		return true
	}
	return probedReachable(s, now, staleSec)
}

// SelectForDelivery 是订阅下发前的最终筛选，订阅与后台「是否下发」共用，保证两边结论一致：
// 未开启按国家去重时原样下发；开启时每个国家保留一个最优可用节点，
// 若一个可用节点都没有（如探测整体异常），退回原始列表，避免客户端拿到空订阅。
func SelectForDelivery(servers []Map, dedup bool, staleSec int, naming bool, countryName func(string) string) []Map {
	if !dedup {
		return servers
	}
	if d := PickBestPerCountry(servers, staleSec, naming, countryName); len(d) > 0 || len(servers) == 0 {
		return d
	}
	return servers
}

func PickBestPerCountry(servers []Map, staleSec int, naming bool, countryName func(string) string) []Map {
	now := time.Now().Unix()
	available := func(s Map) bool { return NodeAvailable(s, now, staleSec) }
	fresh := func(s Map) bool { return probedReachable(s, now, staleSec) }
	var noCountry []Map
	order := []string{}
	by := map[string][]Map{}
	for _, s := range servers {
		if !available(s) {
			continue
		}
		cc := strings.ToUpper(Str(s["country_code"]))
		if cc == "" {
			noCountry = append(noCountry, s)
			continue
		}
		if _, ok := by[cc]; !ok {
			order = append(order, cc)
		}
		by[cc] = append(by[cc], s)
	}
	var picked []Map
	for _, cc := range order {
		g := by[cc]
		var fr []Map
		for _, s := range g {
			if fresh(s) {
				fr = append(fr, s)
			}
		}
		var best Map
		if len(fr) > 0 {
			sort.SliceStable(fr, func(i, j int) bool { return Int(fr[i]["latency_ms"]) < Int(fr[j]["latency_ms"]) })
			best = fr[0]
		} else {
			sort.SliceStable(g, func(i, j int) bool {
				if a, b := Int(g[i]["online"]), Int(g[j]["online"]); a != b {
					return a < b
				}
				return Int(g[i]["sort"]) < Int(g[j]["sort"])
			})
			best = g[0]
		}
		if naming && countryName != nil {
			best = cloneMap(best)
			best["name"] = countryName(cc)
		}
		picked = append(picked, best)
	}
	res := append(picked, noCountry...)
	sort.SliceStable(res, func(i, j int) bool { return Int(res[i]["sort"]) < Int(res[j]["sort"]) })
	return res
}

func cloneMap(m Map) Map {
	n := make(Map, len(m))
	for k, v := range m {
		n[k] = v
	}
	return n
}

// nodeCacheKey 对应 Server::cacheKey 访问器："{type}-{id}-{updated_at}-{is_online}"，
// is_online 为 false 时 PHP 插值为空串。
func nodeCacheKey(m Map, updated time.Time) string {
	online := ""
	if Truthy(m["is_online"]) {
		online = "1"
	}
	return fmt.Sprintf("%s-%s-%s-%s", Str(m["type"]), Str(m["id"]), updated.Format("2006-01-02 15:04:05"), online)
}
