package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/sub"
)

// 管理端节点管理（核心 CRUD）。对应 V2/Admin/Server/ManageController 的 getNodes、save、update、drop、sort。
// 未包含：探测（probe*）、批量导入（batch*/parseNodes/import*）、插件相关、copy、generateEchKey、syncStats。

var nodeTypes = map[string]bool{
	"shadowsocks": true, "vmess": true, "vless": true, "trojan": true, "hysteria": true,
	"tuic": true, "anytls": true, "socks": true, "http": true, "naive": true, "mieru": true,
}

// nodeJSONCols 为需要以原始 JSON 存储的列。
var nodeJSONCols = map[string]bool{
	"group_ids": true, "route_ids": true, "tags": true, "protocol_settings": true, "rate_time_ranges": true,
	"custom_outbounds": true, "custom_routes": true, "cert_config": true,
}

// nodeWritable 为允许通过 save 写入的字段。
var nodeWritable = []string{"name", "type", "code", "host", "port", "server_port", "rate", "show", "sort",
	"parent_id", "machine_id", "enabled", "country_code", "rate_time_enable", "rate_time_ranges", "transfer_enable",
	"group_ids", "route_ids", "tags", "protocol_settings", "custom_outbounds", "custom_routes", "cert_config"}

// nodeView 生成节点列表项：分组、父节点、测速结果、运行状态（Redis）与是否下发。
// 字段对应 PHP getAllServers 追加的访问器：last_check_at / last_push_at / online / is_online /
// available_status / load_status / metrics / online_conn，再加测速字段 latency_ms / last_latency_at。
func (a *App) nodeView(s model.Server, rt nodeRuntime, delivered bool) gin.H {
	var groups []gin.H
	var gids []int64
	_ = json.Unmarshal(s.GroupIDs, &gids)
	if len(gids) > 0 {
		var gs []model.ServerGroup
		a.S.DB.Select("id", "name").Where("id IN ?", gids).Find(&gs)
		for _, g := range gs {
			groups = append(groups, gin.H{"id": g.ID, "name": g.Name})
		}
	}
	var parent any
	if s.ParentID != nil {
		var p model.Server
		if a.S.DB.Select("id", "name").First(&p, *s.ParentID).Error == nil {
			parent = gin.H{"id": p.ID, "name": p.Name}
		}
	}
	return gin.H{
		"id": s.ID, "type": s.Type, "code": s.Code, "name": s.Name, "rate": fmt.Sprintf("%.2f", s.Rate), "host": s.Host, "port": s.Port,
		"server_port": s.ServerPort, "show": s.Show, "sort": s.Sort, "parent_id": s.ParentID, "machine_id": s.MachineID,
		"enabled": s.Enabled, "group_ids": s.GroupIDs, "route_ids": s.RouteIDs, "tags": s.Tags,
		"protocol_settings": normalizedProtocolSettings(s), "transfer_enable": s.TransferEnable, "u": s.U, "d": s.D,
		"country_code": s.CountryCode, "exit_ip": s.ExitIP, "groups": groups, "parent": parent, "is_delivered": delivered,
		"latency_ms": s.LatencyMs, "last_latency_at": s.LastLatencyAt,
		"custom_outbounds": jsonOrNil(s.CustomOutbounds), "custom_routes": jsonOrNil(s.CustomRoutes),
		"cert_config": jsonOrNil(s.CertConfig), "rate_time_enable": s.RateTimeEnable,
		"rate_time_ranges": jsonOrNil(s.RateTimeRanges),
		"created_at":       s.CreatedAt.Unix(), "updated_at": s.UpdatedAt.Unix(),
		// 对应 Server::cacheKey："{type}-{id}-{updated_at}-{is_online}"
		"cache_key":     fmt.Sprintf("%s-%d-%d-%d", s.Type, s.ID, s.UpdatedAt.Unix(), rt.IsOnline),
		"last_check_at": rt.LastCheckAt, "last_push_at": rt.LastPushAt, "online": rt.Online,
		"is_online": rt.IsOnline, "available_status": rt.AvailableStatus,
		"load_status": rt.LoadStatus, "metrics": rt.Metrics, "online_conn": rt.OnlineConn,
	}
}

// normalizedProtocolSettings 对应 Server::getProtocolSettingsAttribute：按协议补默认值并转换类型。
func normalizedProtocolSettings(s model.Server) map[string]any {
	raw := map[string]any{}
	if len(s.ProtocolSettings) > 0 {
		_ = json.Unmarshal(s.ProtocolSettings, &raw)
	}
	return sub.NormalizeProtocolSettings(s.Type, raw)
}

// jsonOrNil 把 JSON 列转为响应值：空或 null 返回 nil，避免输出非法的空 RawMessage。
func jsonOrNil(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return raw
}

// nodeRuntime 是节点在 Redis 里的运行状态。
type nodeRuntime struct {
	LastCheckAt     *int64
	LastPushAt      *int64
	Online          int64
	IsOnline        int
	AvailableStatus int
	LoadStatus      any
	Metrics         any
	OnlineConn      int64
}

// 节点状态（available_status）取值，对应 Server::STATUS_*。
const (
	nodeStatusOffline      = 0
	nodeStatusOnlineNoPush = 1
	nodeStatusOnline       = 2
	// nodeStatusProbed 仅用于管理端列表：没有 agent 心跳（PHP 口径为离线），但面板实测连通，
	// 多见于从第三方订阅导入的节点。避免把明明可用的节点显示成「未运行」。
	nodeStatusProbed  = 3
	nodeCheckInterval = 300
)

// loadNodeRuntime 读取节点的运行状态。子节点的状态取自父节点（与 PHP 的 parent_id ?: id 一致）。
func (a *App) loadNodeRuntime(s model.Server) nodeRuntime {
	id := s.ID
	if s.ParentID != nil {
		id = *s.ParentID
	}
	t := strings.ToUpper(s.Type)
	key := func(name string) string { return fmt.Sprintf("SERVER_%s_%s_%d", t, name, id) }
	ctx := bgCtx()
	vals, err := a.S.Redis.MGet(ctx, key("LAST_CHECK_AT"), key("LAST_PUSH_AT"), key("ONLINE_USER"),
		key("LOAD_STATUS"), key("METRICS")).Result()
	rt := nodeRuntime{}
	if err != nil || len(vals) < 5 {
		return rt
	}
	asStr := func(v any) string {
		if x, ok := v.(string); ok {
			return x
		}
		return ""
	}
	num := func(v any) *int64 {
		if n, err := strconv.ParseInt(asStr(v), 10, 64); err == nil {
			return &n
		}
		return nil
	}
	rt.LastCheckAt, rt.LastPushAt = num(vals[0]), num(vals[1])
	if n := num(vals[2]); n != nil {
		rt.Online = *n
	}
	if raw := asStr(vals[3]); raw != "" {
		var v any
		if json.Unmarshal([]byte(raw), &v) == nil {
			rt.LoadStatus = v
		}
	}
	if raw := asStr(vals[4]); raw != "" {
		var m map[string]any
		if json.Unmarshal([]byte(raw), &m) == nil {
			rt.Metrics = m
			switch v := m["active_connections"].(type) {
			case float64:
				rt.OnlineConn = int64(v)
			}
		}
	}
	nowTs := now()
	fresh := func(p *int64) bool { return p != nil && nowTs-nodeCheckInterval < *p }
	// 经 WebSocket 连接的节点没有 HTTP 心跳，用 node_ws_alive 作为在线依据。
	wsAlive := a.S.Redis.Exists(ctx, fmt.Sprintf("node_ws_alive:%d", id)).Val() > 0
	if fresh(rt.LastCheckAt) || wsAlive {
		rt.IsOnline = 1
	}
	switch {
	case !fresh(rt.LastCheckAt) && !wsAlive:
		rt.AvailableStatus = nodeStatusOffline
	case !fresh(rt.LastPushAt):
		rt.AvailableStatus = nodeStatusOnlineNoPush
	default:
		rt.AvailableStatus = nodeStatusOnline
	}
	return rt
}

// effectiveStaleSeconds 对应 ServerService::effectiveStaleSeconds：
// 测速结果的有效期不能短于探测间隔的 1.5 倍，否则两次探测之间节点会被误判为过期。
func (a *App) effectiveStaleSeconds() int {
	stale := a.Set.Int("server_latency_stale_seconds", 600)
	if a.Set.Bool("server_latency_probe_enable", true) {
		interval := a.Set.Int("server_latency_probe_interval", 3)
		if interval < 1 {
			interval = 1
		}
		if v := int(float64(interval*60) * 1.5); v > stale {
			stale = v
		}
	}
	return stale
}

// probeFresh 判断节点是否有「仍然有效且连通」的测速结果（latency_ms 非空表示连通，空表示测过但不通）。
// stale 为负数（测速已关闭，见 sub.NoExpiry）时不判断过期，以最后一次结论为准。
func probeFresh(s model.Server, nowTs, stale int64) bool {
	return s.LatencyMs != nil && s.LastLatencyAt != nil && (stale < 0 || nowTs-*s.LastLatencyAt <= stale)
}

// deliveryStale 返回判断节点可用性时的有效期；测速关闭时返回 sub.NoExpiry——
// 测速结果不会再刷新，若仍按时间判断，过了有效期所有节点都会被判为不可用。
func (a *App) deliveryStale() int {
	if !a.Set.Bool("server_latency_probe_enable", true) {
		return sub.NoExpiry
	}
	return a.effectiveStaleSeconds()
}

// computeDeliveredIDs 算出「订阅实际会下发」的节点：与 doSubscribe 走同一套筛选（sub.SelectForDelivery）。
// 候选为已显示、分配了权限组、未超流量的节点（含子节点，订阅同样下发子节点）；
// 开启按国家去重时每国保留一个最优可用节点，没有可用节点时与订阅一样退回全部候选。
//
// 与 PHP ManageController::computeDeliveredIds 的差异：PHP 不论是否去重都先剔除「不可用」节点和子节点，
// 而订阅本身并不剔除，导致这一列和实际下发对不上；这里以实际下发为准。权限组是否有用户因人而异，不在此判断。
func (a *App) computeDeliveredIDs(rows []model.Server, rts []nodeRuntime) map[int64]bool {
	var cands []sub.Map
	for i, s := range rows {
		if !s.Show {
			continue
		}
		var gids []any
		if json.Unmarshal(s.GroupIDs, &gids) != nil || len(gids) == 0 {
			continue
		}
		if s.TransferEnable != nil && *s.TransferEnable > 0 && s.U+s.D >= *s.TransferEnable {
			continue
		}
		m := sub.Map{"id": s.ID, "is_online": rts[i].IsOnline, "online": rts[i].Online}
		if s.LatencyMs != nil {
			m["latency_ms"] = *s.LatencyMs
		}
		if s.LastLatencyAt != nil {
			m["last_latency_at"] = *s.LastLatencyAt
		}
		if s.CountryCode != nil {
			m["country_code"] = *s.CountryCode
		}
		if s.Sort != nil {
			m["sort"] = *s.Sort
		}
		cands = append(cands, m)
	}
	picked := sub.SelectForDelivery(cands, a.Set.Bool("server_dedup_by_country", false), a.deliveryStale(), false, nil)
	out := map[int64]bool{}
	for _, p := range picked {
		if id, ok := p["id"].(int64); ok {
			out[id] = true
		}
	}
	return out
}

func (a *App) nodeFetch(c *gin.Context) {
	var rows []model.Server
	a.S.DB.Order("sort ASC").Find(&rows)
	rts := make([]nodeRuntime, len(rows))
	for i, s := range rows {
		rts[i] = a.loadNodeRuntime(s)
	}
	delivered := a.computeDeliveredIDs(rows, rts)
	stale := int64(a.deliveryStale())
	for i, s := range rows {
		if rts[i].AvailableStatus == nodeStatusOffline && probeFresh(s, now(), stale) {
			rts[i].AvailableStatus = nodeStatusProbed
		}
	}
	out := make([]gin.H, 0, len(rows))
	for i, s := range rows {
		out = append(out, a.nodeView(s, rts[i], delivered[s.ID]))
	}
	resp.OK(c, out)
}

func (a *App) nodeSave(c *gin.Context) {
	m := bodyMap(c)
	upd := map[string]any{}
	for _, k := range nodeWritable {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch {
		case nodeJSONCols[k]:
			b, _ := json.Marshal(normalizeIDList(k, v))
			upd[k] = string(b)
		case k == "show" || k == "enabled" || k == "rate_time_enable":
			b, _ := mBool(m, k)
			upd[k] = b
		default:
			upd[k] = v
		}
	}
	if t, ok := upd["type"].(string); ok && !nodeTypes[t] {
		resp.Fail(c, 422, "节点类型参数有误")
		return
	}
	// 协议设置写入前按协议规范化（补默认值、转类型、去掉未声明字段），与读取时一致。
	if raw, ok := m["protocol_settings"]; ok {
		typ, _ := upd["type"].(string)
		if typ == "" {
			if id, ok := mInt(m, "id"); ok && id > 0 {
				var cur model.Server
				if a.S.DB.Select("id", "type").First(&cur, id).Error == nil {
					typ = cur.Type
				}
			}
		}
		if obj, ok := raw.(map[string]any); ok && typ != "" {
			b, _ := json.Marshal(sub.NormalizeProtocolSettings(typ, obj))
			upd["protocol_settings"] = string(b)
		}
	}
	if id, ok := mInt(m, "id"); ok && id > 0 {
		var s model.Server
		if err := a.S.DB.First(&s, id).Error; err != nil {
			resp.Fail(c, 400, "服务器不存在")
			return
		}
		if err := a.S.DB.Model(&s).Updates(upd).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		resp.OK(c, true)
		return
	}
	// v2_server 的时间戳列是 TIMESTAMP；map 创建不会自动填充，缺失会让 cache_key 带零值时间。
	upd["created_at"], upd["updated_at"] = time.Now(), time.Now()
	if err := a.S.DB.Model(&model.Server{}).Create(upd).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}
	resp.OK(c, true)
}

// nodeUpdate 对应 ManageController::update：只允许切换显示、宿主机、启用状态。
func (a *App) nodeUpdate(c *gin.Context) {
	m := bodyMap(c)
	id, ok := mInt(m, "id")
	if !ok {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	var s model.Server
	if err := a.S.DB.First(&s, id).Error; err != nil {
		resp.Fail(c, 400, "服务器不存在")
		return
	}
	upd := map[string]any{}
	if v, ok := mInt(m, "show"); ok {
		upd["show"] = v != 0
	}
	if v, ok := mInt(m, "machine_id"); ok {
		if v == 0 {
			upd["machine_id"] = nil
		} else {
			upd["machine_id"] = v
		}
	}
	if v, ok := mBool(m, "enabled"); ok {
		upd["enabled"] = v
	}
	if len(upd) > 0 {
		if err := a.S.DB.Model(&s).Updates(upd).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
	}
	resp.OK(c, true)
}

func (a *App) nodeDrop(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if id == 0 {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	res := a.S.DB.Delete(&model.Server{}, id)
	if res.RowsAffected == 0 {
		resp.Fail(c, 400, "服务器不存在")
		return
	}
	resp.OK(c, true)
}

// nodeSort 接收 [{id, order}]，按 order 写入 sort。
func (a *App) nodeSort(c *gin.Context) {
	var items []struct {
		ID    int64 `json:"id"`
		Order int64 `json:"order"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&items); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range items {
			if err := tx.Model(&model.Server{}).Where("id = ?", it.ID).Update("sort", it.Order).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

// normalizeIDList 把 group_ids / route_ids 里的数字字符串转成整数。
// 前端编辑节点时会把 ID 作为字符串回传，而订阅与节点端按 JSON_CONTAINS(group_ids, 数字) 匹配，
// 字符串 "39" 与数字 39 在 JSON 里类型不同，会导致该节点对用户组不可见。
func normalizeIDList(key string, v any) any {
	if key != "group_ids" && key != "route_ids" {
		return v
	}
	arr, ok := v.([]any)
	if !ok {
		return v
	}
	out := make([]any, 0, len(arr))
	for _, x := range arr {
		if n, ok := x.(string); ok {
			if i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64); err == nil {
				out = append(out, i)
				continue
			}
		}
		out = append(out, x)
	}
	return out
}
