package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
)

// 节点 V2 接口（对应 app/Http/Controllers/V2/Server 与 Middleware/ServerV2）。
//
// 鉴权两种方式：
//   - 带 machine_id：宿主机令牌，node_id 可选（握手时）或必须属于该宿主机；
//   - 否则：server_token + node_id（与 V1 UniProxy 相同）。
//
// 完成的接口：handshake、report、machine/nodes、machine/status，以及复用 V1 处理器的
// config/user/push/alive/alivelist/status。

const machineKey = "machine"

// requestParams 合并查询串、表单与 JSON 请求体，结果缓存在上下文中，供中间件与处理器共用。
func requestParams(c *gin.Context) map[string]any {
	if v, ok := c.Get("_params"); ok {
		return v.(map[string]any)
	}
	m := map[string]any{}
	for k, vs := range c.Request.URL.Query() {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	if strings.Contains(c.ContentType(), "application/json") && c.Request.Body != nil {
		body, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.UseNumber()
		var jm map[string]any
		if dec.Decode(&jm) == nil {
			for k, v := range jm {
				m[k] = v
			}
		}
	} else {
		_ = c.Request.ParseForm()
		for k, vs := range c.Request.PostForm {
			if len(vs) > 0 {
				m[k] = vs[0]
			}
		}
	}
	c.Set("_params", m)
	return m
}

// anyInt 把请求参数解析为整数（支持字符串、json.Number 与 float64）。
func anyInt(v any) (int64, bool) {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i, true
		}
		if f, err := x.Float64(); err == nil {
			return int64(f), true
		}
	case int:
		return int64(x), true
	case int64:
		return x, true
	case string:
		if i, err := strconv.ParseInt(x, 10, 64); err == nil {
			return i, true
		}
	case float64:
		return int64(x), true
	}
	return 0, false
}

func (a *App) serverV2Auth(c *gin.Context) {
	p := requestParams(c)
	if _, ok := p["machine_id"]; ok && p["machine_id"] != "" {
		a.authByMachine(c, p)
		return
	}
	a.authByServerToken(c, p)
}

func (a *App) authByServerToken(c *gin.Context, p map[string]any) {
	tok, _ := p["token"].(string)
	want := a.Set.Get("server_token", "")
	if want == "" || !auth.ConstEq(want, tok) {
		resp.Fail(c, 422, "Invalid token")
		return
	}
	handshake := strings.HasSuffix(c.Request.URL.Path, "/handshake")
	nodeRaw, hasNode := p["node_id"]
	if !hasNode || nodeRaw == "" || nodeRaw == nil {
		if !handshake {
			resp.Fail(c, 422, "node_id required")
			return
		}
		c.Next()
		return
	}
	id, ok := anyInt(nodeRaw)
	if !ok {
		resp.Fail(c, 422, "node_id must be integer")
		return
	}
	var n model.Server
	if err := a.S.DB.First(&n, id).Error; err != nil {
		resp.Fail(c, 500, "Server does not exist")
		return
	}
	c.Set(nodeKey, &n)
	c.Next()
}

func (a *App) authByMachine(c *gin.Context, p map[string]any) {
	mid, ok := anyInt(p["machine_id"])
	tok, _ := p["token"].(string)
	if !ok || tok == "" {
		resp.Fail(c, 422, "machine_id and token required")
		return
	}
	var m model.ServerMachine
	if err := a.S.DB.Where("id = ? AND token = ?", mid, tok).First(&m).Error; err != nil {
		resp.Fail(c, 401, "Machine not found or invalid token")
		return
	}
	if !m.IsActive {
		resp.Fail(c, 403, "Machine is disabled")
		return
	}
	if nid, ok := anyInt(p["node_id"]); ok && nid > 0 {
		var n model.Server
		err := a.S.DB.Where("id = ? AND machine_id = ? AND (enabled = 1 OR enabled IS NULL)", nid, m.ID).First(&n).Error
		if err != nil {
			resp.Fail(c, 500, "Node not found on this machine")
			return
		}
		c.Set(nodeKey, &n)
	}
	a.recordMachineContact(&m, c.ClientIP())
	c.Set(machineKey, &m)
	c.Next()
}

// registerServerV2 挂载 /api/v2/server 下的节点接口。
func (a *App) registerServerV2(r *gin.Engine) {
	g := r.Group("/api/v2/server", a.serverV2Auth)
	g.GET("/handshake", a.handshake)
	g.POST("/handshake", a.handshake)
	g.POST("/report", a.serverReport)
	g.GET("/config", a.nodeConfig)
	g.GET("/user", a.nodeUsers)
	g.POST("/push", a.nodePush)
	g.POST("/alive", a.nodeAlive)
	g.GET("/alivelist", a.nodeAliveList)
	g.POST("/status", a.nodeStatus)

	m := r.Group("/api/v2/server/machine")
	m.POST("/nodes", a.machineNodes)
	m.POST("/status", a.machineStatus)
}

// pushInterval 对应 admin_setting('server_push_interval', 60)。
func (a *App) pushInterval() int64 { return int64(a.Set.Int("server_push_interval", 60)) }

// cacheTTL 对应 max(300, push_interval * 3)。
func (a *App) cacheTTL() time.Duration {
	ttl := a.pushInterval() * 3
	if ttl < 300 {
		ttl = 300
	}
	return time.Duration(ttl) * time.Second
}

func (a *App) handshake(c *gin.Context) {
	ws := gin.H{"enabled": false}
	if a.Set.Bool("server_ws_enable", true) && a.S.Redis.Exists(bgCtx(), "ws_server:heartbeat").Val() > 0 {
		wsURL := strings.TrimSpace(a.Set.Get("server_ws_url", ""))
		if wsURL == "" {
			scheme := "ws"
			if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
				scheme = "wss"
			}
			wsURL = scheme + "://" + c.Request.Host + "/ws"
		}
		ws = gin.H{"enabled": true, "ws_url": strings.TrimRight(wsURL, "/")}
	}
	c.JSON(200, gin.H{"websocket": ws})
}

// serverReport 对应 ServerController::report：更新心跳、流量、在线、设备、状态与指标。
func (a *App) serverReport(c *gin.Context) {
	n, ok := c.Get(nodeKey)
	if !ok {
		resp.Fail(c, 500, "Server does not exist")
		return
	}
	node := n.(*model.Server)
	a.touch(c, node, "LAST_CHECK_AT")

	p := requestParams(c)
	if traffic, ok := p["traffic"].(map[string]any); ok && len(traffic) > 0 {
		a.applyTrafficMap(c, node, traffic)
	}
	if alive, ok := p["alive"].(map[string]any); ok && len(alive) > 0 {
		a.setUserDevices(c, node.ID, alive)
	}
	if online, ok := p["online"].(map[string]any); ok && len(online) > 0 {
		for uid, conn := range online {
			v, _ := anyInt(conn)
			key := fmt.Sprintf("USER_ONLINE_CONN_%s_%d_%s", strings.ToUpper(node.Type), node.ID, uid)
			a.S.Redis.Set(c, key, v, a.cacheTTL())
		}
	}
	if st, ok := p["status"].(map[string]any); ok && len(st) > 0 {
		a.storeNodeStatus(c, node, st)
	}
	if mt, ok := p["metrics"].(map[string]any); ok && len(mt) > 0 {
		b, _ := json.Marshal(mt)
		a.S.Redis.Set(c, fmt.Sprintf("SERVER_%s_METRICS_%d", strings.ToUpper(node.Type), node.ID), string(b), a.cacheTTL())
	}
	resp.OK(c, true)
}

// applyTrafficMap 把 {uid: [up, down]} 记入用户流量，与 V1 push 行为一致。
func (a *App) applyTrafficMap(c *gin.Context, n *model.Server, traffic map[string]any) {
	data := map[string][]float64{}
	for uid, v := range traffic {
		arr, ok := v.([]any)
		if !ok || len(arr) != 2 {
			continue
		}
		up, ok1 := toNum(arr[0])
		down, ok2 := toNum(arr[1])
		if ok1 && ok2 {
			data[uid] = []float64{up, down}
		}
	}
	if len(data) == 0 {
		return
	}
	a.S.Redis.Set(c, fmt.Sprintf("SERVER_%s_ONLINE_USER_%d", strings.ToUpper(n.Type), n.ID), len(data), time.Hour)
	a.S.Redis.Set(c, fmt.Sprintf("SERVER_%s_LAST_PUSH_AT_%d", strings.ToUpper(n.Type), n.ID), now(), time.Hour)
	a.applyTraffic(n, data)
}

// applyTraffic 累加流量（按节点倍率）。V1 push 与 report 共用。
func (a *App) applyTraffic(n *model.Server, data map[string][]float64) {
	rate := n.Rate
	if rate <= 0 {
		rate = 1
	}
	for uid, v := range data {
		id, err := strconv.ParseInt(uid, 10, 64)
		if err != nil || len(v) != 2 {
			continue
		}
		a.S.DB.Exec("UPDATE v2_user SET u = u + ?, d = d + ?, t = ?, updated_at = ? WHERE id = ?",
			int64(v[0]*rate), int64(v[1]*rate), now(), now(), id)
	}
}

// setUserDevices 对应 DeviceStateService::setDevices：先清掉本节点旧记录，再写入新 IP 并续期 300 秒。
func (a *App) setUserDevices(c context.Context, nodeID int64, alive map[string]any) {
	for uid, raw := range alive {
		id, err := strconv.ParseInt(uid, 10, 64)
		if err != nil {
			continue
		}
		ips, _ := raw.([]any)
		key := devicePrefix + strconv.FormatInt(id, 10)
		prefix := strconv.FormatInt(nodeID, 10) + ":"
		old, _ := a.S.Redis.HKeys(c, key).Result()
		for _, f := range old {
			if strings.HasPrefix(f, prefix) {
				a.S.Redis.HDel(c, key, f)
			}
		}
		seen := map[string]bool{}
		fields := map[string]any{}
		ts := now()
		for _, ipRaw := range ips {
			s, _ := ipRaw.(string)
			ip := normalizeDeviceIP(s)
			if ip == "" || seen[ip] {
				continue
			}
			seen[ip] = true
			fields[prefix+ip] = ts
		}
		if len(fields) > 0 {
			a.S.Redis.HSet(c, key, fields)
			a.S.Redis.Expire(c, key, deviceTTL*time.Second)
		}
		a.syncOnlineCount(c, id) // 对应 setDevices 末尾的 notifyUpdate
	}
}

// storeNodeStatus 对应 processStatus：缓存节点负载。
func (a *App) storeNodeStatus(c *gin.Context, n *model.Server, st map[string]any) {
	data := gin.H{
		"cpu":           floatOf(st["cpu"]),
		"mem":           pick2(st, "mem"),
		"swap":          pick2(st, "swap"),
		"disk":          pick2(st, "disk"),
		"updated_at":    now(),
		"kernel_status": st["kernel_status"],
	}
	b, _ := json.Marshal(data)
	t := strings.ToUpper(n.Type)
	ttl := a.cacheTTL()
	a.S.Redis.Set(c, fmt.Sprintf("SERVER_%s_LOAD_STATUS_%d", t, n.ID), string(b), ttl)
	a.S.Redis.Set(c, fmt.Sprintf("SERVER_%s_LAST_LOAD_AT_%d", t, n.ID), now(), ttl)
}

func floatOf(v any) float64 {
	f, _ := toNum(v)
	return f
}

// pick2 取 {total, used} 两个整数字段，缺失按 0 处理。
func pick2(st map[string]any, k string) gin.H {
	m, _ := st[k].(map[string]any)
	t, _ := anyInt(m["total"])
	u, _ := anyInt(m["used"])
	return gin.H{"total": t, "used": u}
}

// machineAuth 只用于宿主机接口：machine_id + token。ip 为请求来源，用于记录宿主机地址。
func (a *App) machineAuth(p map[string]any, ip string) (*model.ServerMachine, error) {
	mid, ok := anyInt(p["machine_id"])
	tok, _ := p["token"].(string)
	if !ok || tok == "" {
		return nil, fmt.Errorf("machine_id and token required")
	}
	var m model.ServerMachine
	if err := a.S.DB.Where("id = ? AND token = ?", mid, tok).First(&m).Error; err != nil {
		return nil, fmt.Errorf("Machine not found or disabled")
	}
	if !m.IsActive {
		return nil, fmt.Errorf("Machine not found or disabled")
	}
	a.recordMachineContact(&m, ip)
	return &m, nil
}

func (a *App) machineNodes(c *gin.Context) {
	m, err := a.machineAuth(requestParams(c), c.ClientIP())
	if err != nil {
		resp.Fail(c, 403, err.Error())
		return
	}
	var servers []model.Server
	a.S.DB.Where("machine_id = ? AND (enabled = 1 OR enabled IS NULL)", m.ID).Order("sort ASC").Find(&servers)
	nodes := make([]gin.H, 0, len(servers))
	for _, s := range servers {
		nodes = append(nodes, gin.H{"id": s.ID, "type": s.Type, "name": s.Name})
	}
	c.JSON(200, gin.H{
		"nodes": nodes,
		"base_config": gin.H{
			"push_interval": a.Set.Int("server_push_interval", 60),
			"pull_interval": a.Set.Int("server_pull_interval", 60),
		},
	})
}

// machineStatus 对应 MachineController::status：更新最新负载并写入历史（约 5% 概率顺带清理一天前的记录）。
func (a *App) machineStatus(c *gin.Context) {
	p := requestParams(c)
	m, err := a.machineAuth(p, c.ClientIP())
	if err != nil {
		resp.Fail(c, 403, err.Error())
		return
	}
	cpu, okCPU := toNum(p["cpu"])
	mem, _ := p["mem"].(map[string]any)
	memTotal, okMT := anyInt(mem["total"])
	memUsed, okMU := anyInt(mem["used"])
	if !okCPU || cpu < 0 || cpu > 100 || !okMT || !okMU {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	disk, _ := p["disk"].(map[string]any)
	diskTotal, _ := anyInt(disk["total"])
	diskUsed, _ := anyInt(disk["used"])
	ts := now()
	load := gin.H{
		"cpu":        cpu,
		"mem":        gin.H{"total": memTotal, "used": memUsed},
		"swap":       pick2(p, "swap"),
		"disk":       gin.H{"total": diskTotal, "used": diskUsed},
		"updated_at": ts,
	}
	var inSpeed, outSpeed *float64
	netRaw, _ := p["net"].(map[string]any)
	if netRaw != nil {
		if in, ok := toNum(netRaw["in_speed"]); ok {
			if out, ok := toNum(netRaw["out_speed"]); ok {
				load["net"] = gin.H{"in_speed": in, "out_speed": out}
				inSpeed, outSpeed = &in, &out
			}
		}
	}
	b, _ := json.Marshal(load)
	a.S.DB.Model(m).Updates(map[string]any{"load_status": json.RawMessage(b), "last_seen_at": ts})
	hist := model.ServerMachineLoadHistory{
		MachineID: m.ID, CPU: cpu, MemTotal: memTotal, MemUsed: memUsed,
		DiskTotal: diskTotal, DiskUsed: diskUsed, RecordedAt: ts,
		NetInSpeed: inSpeed, NetOutSpeed: outSpeed,
	}
	a.S.DB.Create(&hist)
	if rand.Intn(20) == 0 {
		a.S.DB.Where("machine_id = ? AND recorded_at < ?", m.ID, ts-86400).Delete(&model.ServerMachineLoadHistory{})
	}
	c.JSON(200, gin.H{"data": true})
}

// nodeAlive 对应 UniProxyController::alive：请求体 {"<uid>": ["ip", ...]}，写入设备状态（与 ServerService::processAlive 一致）。
func (a *App) nodeAlive(c *gin.Context) {
	n := node(c)
	var body map[string]any
	dec := json.NewDecoder(c.Request.Body)
	dec.UseNumber()
	if err := dec.Decode(&body); err != nil || body == nil {
		c.JSON(400, gin.H{"error": "Invalid online data"})
		return
	}
	a.setUserDevices(c, n.ID, body)
	c.JSON(200, gin.H{"data": true})
}

// nodeAliveList 对应 UniProxyController::alivelist：仅统计设置了设备数限制的可用用户，
// 去重后的在线 IP 数（跨所有节点，超过 TTL 的记录不计）。
func (a *App) nodeAliveList(c *gin.Context) {
	ctx := bgCtx()
	alive := map[string]int{}
	nowTs := now()
	for _, u := range a.availableUsers(node(c)) {
		if u.DeviceLimit == nil || *u.DeviceLimit <= 0 {
			continue
		}
		data, err := a.S.Redis.HGetAll(ctx, devicePrefix+strconv.FormatInt(u.ID, 10)).Result()
		if err != nil {
			continue
		}
		ips := map[string]struct{}{}
		for field, ts := range data {
			t, err := strconv.ParseInt(ts, 10, 64)
			if err != nil || nowTs-t > deviceTTL {
				continue
			}
			if _, ip, ok := strings.Cut(field, ":"); ok {
				ips[ip] = struct{}{}
			}
		}
		if n := a.legacyAliveIPs(ctx, u.ID, nowTs, ips); n > 0 {
			alive[strconv.FormatInt(u.ID, 10)] = n
		}
	}
	c.JSON(200, gin.H{"alive": alive})
}

// legacyAliveIPs 兼容切换前由旧版 Go 实现写入的 ALIVE_IP_USER_<uid> 记录（字段为节点 ID，值为 {aliveips, lastupdateAt}）。
// 这些记录原本 10 分钟过期，切换后用于衔接，把有效 IP 并入 ips 集合并返回合计数。
func (a *App) legacyAliveIPs(ctx context.Context, uid, nowTs int64, ips map[string]struct{}) int {
	m, err := a.S.Redis.HGetAll(ctx, "ALIVE_IP_USER_"+strconv.FormatInt(uid, 10)).Result()
	if err != nil {
		return len(ips)
	}
	for _, raw := range m {
		var e struct {
			IPs []string `json:"aliveips"`
			At  int64    `json:"lastupdateAt"`
		}
		if json.Unmarshal([]byte(raw), &e) != nil || nowTs-e.At > deviceTTL {
			continue
		}
		for _, ip := range e.IPs {
			ips[strings.SplitN(ip, "_", 2)[0]] = struct{}{}
		}
	}
	return len(ips)
}
