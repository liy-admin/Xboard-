package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"xboard/internal/model"
)

// 节点与客户端 WebSocket 通道，对应 app/WebSocket/NodeWorker.php（Workerman）。
//
// 与 PHP 的差异：PHP 监听独立端口，Go 与 HTTP 共用监听地址，路径为 /ws。
// 握手接口返回的 ws_url 为 {scheme}://{host}/ws，与之一致。
//
// 连接参数（query）：
//   - client_token：客户端（终端 App）用户令牌；
//   - machine_id + token：宿主机，管理其下所有启用的节点；
//   - node_id + token：单个节点，token 为 server_token。

const (
	wsAuthTimeout   = 10 * time.Second
	wsPingInterval  = 55 * time.Second
	wsHeartbeatKey  = "ws_server:heartbeat"
	wsPendingKey    = "device:push_pending_nodes"
	wsNodePushChan  = "node:push"
	wsClientChan    = "client:push"
	wsSendQueueSize = 256
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// wsConn 表示一个 WebSocket 连接及其身份。
type wsConn struct {
	ws           *websocket.Conn
	send         chan []byte
	once         sync.Once
	mu           sync.RWMutex
	closed       bool
	nodeID       int64
	machineID    int64
	machineNodes []int64
	userID       int64
}

// enqueue 向连接投递一帧。连接已关闭或队列已满时丢弃（返回 false）：
// 关闭与投递共用一把锁，避免鉴权超时等协程在 close 之后写入已关闭的通道而 panic。
func (c *wsConn) enqueue(b []byte) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- b:
		return true
	default:
		return false // 队列已满：丢弃，避免慢连接阻塞整体推送
	}
}

func (c *wsConn) close() {
	c.once.Do(func() {
		c.mu.Lock()
		c.closed = true
		close(c.send)
		c.mu.Unlock()
	})
}

// wsHub 维护连接注册表（对应 NodeRegistry 与 ClientRegistry）。
type wsHub struct {
	mu       sync.RWMutex
	nodes    map[int64]*wsConn
	machines map[int64]*wsConn
	clients  map[int64]map[*wsConn]struct{}
}

func newWSHub() *wsHub {
	return &wsHub{nodes: map[int64]*wsConn{}, machines: map[int64]*wsConn{}, clients: map[int64]map[*wsConn]struct{}{}}
}

func (h *wsHub) addNode(id int64, c *wsConn) {
	h.mu.Lock()
	h.nodes[id] = c
	h.mu.Unlock()
}

// removeNode 仅在注册的仍是同一连接时删除，避免重连时误删新连接。
func (h *wsHub) removeNode(id int64, c *wsConn) {
	h.mu.Lock()
	if h.nodes[id] == c {
		delete(h.nodes, id)
	}
	h.mu.Unlock()
}

func (h *wsHub) node(id int64) *wsConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.nodes[id]
}

func (h *wsHub) machine(id int64) *wsConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.machines[id]
}

func (h *wsHub) addClient(uid int64, c *wsConn) {
	h.mu.Lock()
	if h.clients[uid] == nil {
		h.clients[uid] = map[*wsConn]struct{}{}
	}
	h.clients[uid][c] = struct{}{}
	h.mu.Unlock()
}

func (h *wsHub) removeClient(uid int64, c *wsConn) {
	h.mu.Lock()
	if set := h.clients[uid]; set != nil {
		delete(set, c)
		if len(set) == 0 {
			delete(h.clients, uid)
		}
	}
	h.mu.Unlock()
}

// allConns 返回所有连接（节点、宿主机、客户端），用于心跳 ping 与全局广播。
func (h *wsHub) allConns() []*wsConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := map[*wsConn]bool{}
	var out []*wsConn
	add := func(c *wsConn) {
		if c != nil && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	for _, c := range h.nodes {
		add(c)
	}
	for _, c := range h.machines {
		add(c)
	}
	for _, set := range h.clients {
		for c := range set {
			add(c)
		}
	}
	return out
}

func (h *wsHub) clientConns(uid int64) []*wsConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []*wsConn
	for c := range h.clients[uid] {
		out = append(out, c)
	}
	return out
}

func (h *wsHub) allClients() []*wsConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []*wsConn
	for _, set := range h.clients {
		for c := range set {
			out = append(out, c)
		}
	}
	return out
}

// wsFrame 为发送给对端的 JSON 消息。
func wsFrame(event string, data any) []byte {
	b, _ := json.Marshal(map[string]any{"event": event, "data": data})
	return b
}

// sendNode 向节点发送事件；若节点由宿主机托管，则经宿主机连接发送并附带 node_id（对应 NodeRegistry::send）。
func (a *App) sendNode(nodeID int64, event string, data map[string]any) bool {
	if c := a.hub.node(nodeID); c != nil {
		return c.enqueue(wsFrame(event, data))
	}
	if m := a.hub.machineFor(nodeID); m != nil && event != "sync.nodes" {
		if data == nil {
			data = map[string]any{}
		}
		if _, ok := data["node_id"]; !ok {
			data["node_id"] = nodeID
		}
		return m.enqueue(wsFrame(event, data))
	}
	return false
}

// machineFor 返回托管某节点的宿主机连接。
func (h *wsHub) machineFor(nodeID int64) *wsConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, m := range h.machines {
		for _, id := range m.machineNodes {
			if id == nodeID {
				return m
			}
		}
	}
	return nil
}

func (a *App) wsHandler(c *gin.Context) {
	ws, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	conn := &wsConn{ws: ws, send: make(chan []byte, wsSendQueueSize)}
	go a.wsWriter(conn)
	q := c.Request.URL.Query()

	authed := make(chan struct{})
	go func() {
		select {
		case <-authed:
		case <-time.After(wsAuthTimeout):
			conn.enqueue(wsFrame("error", gin.H{"message": "auth timeout"}))
			conn.close()
		}
	}()

	var ok bool
	switch {
	case q.Get("client_token") != "":
		ok = a.wsAuthClient(conn, q.Get("client_token"))
	case q.Get("machine_id") != "":
		ok = a.wsAuthMachine(conn, q.Get("machine_id"), q.Get("token"), c.ClientIP())
	default:
		ok = a.wsAuthNode(conn, q.Get("node_id"), q.Get("token"))
	}
	if !ok {
		conn.close()
		return
	}
	close(authed)
	a.wsReader(conn)
	a.wsDisconnect(conn)
	conn.close()
}

// wsWriter 把队列里的帧写到连接。通道关闭后先写完已排队的帧（如鉴权失败的 error），再关闭底层连接。
func (a *App) wsWriter(c *wsConn) {
	defer func() { _ = c.ws.Close() }()
	for msg := range c.send {
		_ = c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
			c.close()
			return
		}
	}
}

func (a *App) wsReject(c *wsConn, msg string) bool {
	c.enqueue(wsFrame("error", gin.H{"message": msg}))
	return false
}

func (a *App) wsAuthClient(c *wsConn, token string) bool {
	var u model.User
	if err := a.S.DB.Select("id").Where("token = ?", token).First(&u).Error; err != nil {
		return a.wsReject(c, "client not found")
	}
	c.userID = u.ID
	a.hub.addClient(u.ID, c)
	c.enqueue(wsFrame("auth.success", gin.H{"user_id": u.ID}))
	return true
}

func (a *App) wsAuthNode(c *wsConn, nodeRaw, token string) bool {
	serverToken := a.Set.Get("server_token", "")
	if token == "" || serverToken == "" || token != serverToken {
		return a.wsReject(c, "invalid token")
	}
	id, _ := strconv.ParseInt(nodeRaw, 10, 64)
	var n model.Server
	if id == 0 || a.S.DB.First(&n, id).Error != nil {
		return a.wsReject(c, "node not found")
	}
	c.nodeID = id
	a.hub.addNode(id, c)
	a.S.Redis.Set(context.Background(), fmt.Sprintf("node_ws_alive:%d", id), 1, 24*time.Hour)
	a.clearNodeDevices(id)
	c.enqueue(wsFrame("auth.success", gin.H{"node_id": id}))
	a.wsFullSync(&n)
	a.broadcastSubscriptionUpdated("node.online")
	return true
}

func (a *App) wsAuthMachine(c *wsConn, midRaw, token, ip string) bool {
	mid, _ := strconv.ParseInt(midRaw, 10, 64)
	var m model.ServerMachine
	if mid == 0 || a.S.DB.Where("id = ? AND token = ?", mid, token).First(&m).Error != nil || !m.IsActive {
		return a.wsReject(c, "invalid machine credentials")
	}
	var nodes []model.Server
	a.S.DB.Where("machine_id = ? AND (enabled = 1 OR enabled IS NULL)", m.ID).Order("sort ASC").Find(&nodes)
	a.recordMachineContact(&m, ip)
	ids := make([]int64, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	c.machineID = m.ID
	c.machineNodes = ids
	a.hub.mu.Lock()
	a.hub.machines[m.ID] = c
	a.hub.mu.Unlock()
	for _, id := range ids {
		a.S.Redis.Set(context.Background(), fmt.Sprintf("node_ws_alive:%d", id), 1, 24*time.Hour)
		a.clearNodeDevices(id)
	}
	c.enqueue(wsFrame("auth.success", gin.H{"machine_id": m.ID, "node_ids": ids}))
	for i := range nodes {
		a.wsFullSync(&nodes[i])
	}
	a.broadcastSubscriptionUpdated("node.online")
	return true
}

// wsFullSync 对应 NodeEventHandlers::pushFullSync：下发节点配置与可用用户。
func (a *App) wsFullSync(n *model.Server) {
	a.sendNode(n.ID, "sync.config", map[string]any{"config": a.nodeConfigMap(n)})
	a.sendNode(n.ID, "sync.users", map[string]any{"users": a.availableUsers(n)})
}

// wsDisconnect 对应 onClose：摘除注册、清理设备状态并通知订阅变化。
func (a *App) wsDisconnect(c *wsConn) {
	if c.userID != 0 {
		a.hub.removeClient(c.userID, c)
		return
	}
	if c.machineID != 0 {
		a.hub.mu.Lock()
		if a.hub.machines[c.machineID] == c {
			delete(a.hub.machines, c.machineID)
		}
		a.hub.mu.Unlock()
		for _, id := range c.machineNodes {
			a.S.Redis.Del(context.Background(), fmt.Sprintf("node_ws_alive:%d", id))
			for _, uid := range a.clearNodeDevices(id) {
				a.notifyDevicesChanged(uid)
			}
		}
		a.broadcastSubscriptionUpdated("node.offline")
		return
	}
	if c.nodeID != 0 {
		a.hub.removeNode(c.nodeID, c)
		a.S.Redis.Del(context.Background(), fmt.Sprintf("node_ws_alive:%d", c.nodeID))
		for _, uid := range a.clearNodeDevices(c.nodeID) {
			a.notifyDevicesChanged(uid)
		}
		a.broadcastSubscriptionUpdated("node.offline")
	}
}

// notifyDevicesChanged 对应 DeviceStateService::notifyUpdate：通知该用户的在线客户端重新拉取。
func (a *App) notifyDevicesChanged(uid int64) {
	for _, c := range a.hub.clientConns(uid) {
		c.enqueue(wsFrame("subscription.updated", gin.H{"reason": "devices"}))
	}
}

// wsReader 处理节点、宿主机发来的消息。
func (a *App) wsReader(c *wsConn) {
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Event string         `json:"event"`
			Data  map[string]any `json:"data"`
		}
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if c.userID != 0 {
			continue
		}
		if c.machineID != 0 {
			if msg.Event == "pong" {
				for _, id := range c.machineNodes {
					a.S.Redis.Set(context.Background(), fmt.Sprintf("node_ws_alive:%d", id), 1, 24*time.Hour)
				}
				continue
			}
			nid, _ := anyInt(msg.Data["node_id"])
			if !containsID(c.machineNodes, nid) {
				continue
			}
			a.dispatchNodeEvent(nid, msg.Event, msg.Data)
			continue
		}
		if msg.Event == "pong" {
			a.S.Redis.Set(context.Background(), fmt.Sprintf("node_ws_alive:%d", c.nodeID), 1, 24*time.Hour)
			continue
		}
		a.dispatchNodeEvent(c.nodeID, msg.Event, msg.Data)
	}
}

func containsID(ids []int64, id int64) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

// dispatchNodeEvent 对应 NodeWorker 的 handlers 表：node.status、report.devices、request.devices。
func (a *App) dispatchNodeEvent(nodeID int64, event string, data map[string]any) {
	switch event {
	case "node.status":
		var n model.Server
		if a.S.DB.First(&n, nodeID).Error != nil {
			return
		}
		a.S.Redis.Set(context.Background(), fmt.Sprintf("SERVER_%s_LAST_CHECK_AT_%d", strings.ToUpper(n.Type), n.ID), now(), time.Hour)
		b, _ := json.Marshal(data)
		a.S.Redis.Set(context.Background(), fmt.Sprintf("SERVER_%s_METRICS_%d", strings.ToUpper(n.Type), n.ID), string(b), a.cacheTTL())
	case "report.devices":
		a.handleDeviceReport(nodeID, data)
	case "request.devices":
		a.sendDevicesTo(nodeID)
	}
}

// handleDeviceReport 对应 NodeEventHandlers::handleDeviceReport：以节点上报为准同步设备状态。
func (a *App) handleDeviceReport(nodeID int64, data map[string]any) {
	if inner, ok := data["devices"].(map[string]any); ok {
		data = inner
	}
	reported := map[string][]any{}
	for uid, v := range data {
		if _, err := strconv.ParseInt(uid, 10, 64); err != nil {
			continue
		}
		if ips, ok := v.([]any); ok {
			reported[uid] = ips
		}
	}
	old := a.nodeDeviceUsers(nodeID)
	for uid := range old {
		if _, ok := reported[strconv.FormatInt(uid, 10)]; !ok {
			a.removeNodeDevicesOf(nodeID, uid)
			a.notifyDevicesChanged(uid)
		}
	}
	for uid, ips := range reported {
		a.setUserDevices(bgCtx(), nodeID, map[string]any{uid: ips})
	}
	a.S.Redis.SAdd(context.Background(), wsPendingKey, nodeID)
}

// sendDevicesTo 对应 handleDeviceRequest / pushDeviceStateToNode：下发节点可用用户的在线 IP。
func (a *App) sendDevicesTo(nodeID int64) {
	var n model.Server
	if a.S.DB.First(&n, nodeID).Error != nil {
		return
	}
	users := a.availableUsers(&n)
	ids := make([]int64, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	a.sendNode(nodeID, "sync.devices", map[string]any{"users": a.usersDevices(ids)})
}

// usersDevices 返回 {uid: [ip...]}（仅 TTL 内的记录，去重）。
func (a *App) usersDevices(uids []int64) map[string][]string {
	out := map[string][]string{}
	nowTs := now()
	for _, uid := range uids {
		data, err := a.S.Redis.HGetAll(context.Background(), devicePrefix+strconv.FormatInt(uid, 10)).Result()
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		var ips []string
		for field, ts := range data {
			t, err := strconv.ParseInt(ts, 10, 64)
			if err != nil || nowTs-t > deviceTTL {
				continue
			}
			if _, ip, ok := strings.Cut(field, ":"); ok && !seen[ip] {
				seen[ip] = true
				ips = append(ips, ip)
			}
		}
		if len(ips) > 0 {
			out[strconv.FormatInt(uid, 10)] = ips
		}
	}
	return out
}

// clearNodeDevices 对应 DeviceStateService::clearAllNodeDevices：清除该节点的全部设备记录，返回受影响用户。
func (a *App) clearNodeDevices(nodeID int64) []int64 {
	var affected []int64
	ctx := context.Background()
	for _, uid := range a.scanDeviceUsers() {
		if a.removeNodeDevicesOf(nodeID, uid) {
			affected = append(affected, uid)
		}
	}
	_ = ctx
	return affected
}

// nodeDeviceUsers 返回在该节点上有设备记录的用户集合。
func (a *App) nodeDeviceUsers(nodeID int64) map[int64]bool {
	out := map[int64]bool{}
	prefix := strconv.FormatInt(nodeID, 10) + ":"
	for _, uid := range a.scanDeviceUsers() {
		fields, err := a.S.Redis.HKeys(context.Background(), devicePrefix+strconv.FormatInt(uid, 10)).Result()
		if err != nil {
			continue
		}
		for _, f := range fields {
			if strings.HasPrefix(f, prefix) {
				out[uid] = true
				break
			}
		}
	}
	return out
}

// removeNodeDevicesOf 删除某用户在某节点上的设备字段，返回是否删除了记录。
func (a *App) removeNodeDevicesOf(nodeID, uid int64) bool {
	key := devicePrefix + strconv.FormatInt(uid, 10)
	prefix := strconv.FormatInt(nodeID, 10) + ":"
	fields, err := a.S.Redis.HKeys(context.Background(), key).Result()
	if err != nil {
		return false
	}
	removed := false
	for _, f := range fields {
		if strings.HasPrefix(f, prefix) {
			a.S.Redis.HDel(context.Background(), key, f)
			removed = true
		}
	}
	if removed {
		a.syncOnlineCount(context.Background(), uid)
	}
	return removed
}

// scanDeviceUsers 扫描 Redis 中所有 user_devices:<uid> 键，返回用户 ID。
func (a *App) scanDeviceUsers() []int64 {
	var out []int64
	var cursor uint64
	ctx := context.Background()
	for {
		keys, next, err := a.S.Redis.Scan(ctx, cursor, devicePrefix+"*", 500).Result()
		if err != nil {
			break
		}
		for _, k := range keys {
			if id, err := strconv.ParseInt(strings.TrimPrefix(k, devicePrefix), 10, 64); err == nil {
				out = append(out, id)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return out
}

// broadcastSubscriptionUpdated 对应 ClientPushService::broadcastSubscriptionUpdated（client_ws_enable 开启时，3 秒节流）。
var wsLastBroadcast sync.Map

func (a *App) broadcastSubscriptionUpdated(reason string) {
	if !a.Set.Bool("client_ws_enable", false) {
		return
	}
	nowTs := time.Now().Unix()
	if last, ok := wsLastBroadcast.Load(reason); ok && nowTs-last.(int64) < 3 {
		return
	}
	wsLastBroadcast.Store(reason, nowTs)
	for _, c := range a.hub.allClients() {
		c.enqueue(wsFrame("subscription.updated", gin.H{"reason": reason}))
	}
}

// wsBackground 启动心跳、ping、设备推送与 Redis 订阅。进程内只需调用一次（见 StartWS）。
func (a *App) StartWS(ctx context.Context) {
	a.ensureHub()
	a.mailQ().Start(ctx)
	go a.wsHeartbeat(ctx)
	go a.wsPingLoop(ctx)
	go a.wsPendingDevices(ctx)
	go a.wsRedisLoop(ctx)
}

func (a *App) wsHeartbeat(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		a.S.Redis.Set(ctx, wsHeartbeatKey, now(), 30*time.Second)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (a *App) wsPingLoop(ctx context.Context) {
	t := time.NewTicker(wsPingInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ping := wsFrame("ping", nil)
			for _, c := range a.hub.allConns() {
				c.enqueue(ping)
			}
		}
	}
}

// wsPendingDevices 每 10 秒处理待下发设备状态的节点（对应 Timer::add(10)）。
func (a *App) wsPendingDevices(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ids, err := a.S.Redis.SPopN(ctx, wsPendingKey, 100).Result()
			if err != nil {
				continue
			}
			for _, s := range ids {
				id, err := strconv.ParseInt(s, 10, 64)
				if err == nil && a.hub.node(id) != nil {
					a.sendDevicesTo(id)
				}
			}
		}
	}
}

// wsRedisLoop 订阅 node:push 与 client:push，并转发给在线连接。断线后自动重连。
func (a *App) wsRedisLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		ps := a.S.Redis.Subscribe(ctx, wsNodePushChan, wsClientChan)
		ch := ps.Channel()
		for msg := range ch {
			a.handleRedisPush(msg)
		}
		_ = ps.Close()
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (a *App) handleRedisPush(msg *redis.Message) {
	var p map[string]any
	if json.Unmarshal([]byte(msg.Payload), &p) != nil {
		return
	}
	if msg.Channel == wsClientChan {
		event, _ := p["event"].(string)
		if event == "" {
			return
		}
		data, _ := p["data"].(map[string]any)
		if scope, _ := p["scope"].(string); scope == "user" {
			if uid, ok := anyInt(p["user_id"]); ok && uid != 0 {
				for _, c := range a.hub.clientConns(uid) {
					c.enqueue(wsFrame(event, data))
				}
				return
			}
		}
		for _, c := range a.hub.allClients() {
			c.enqueue(wsFrame(event, data))
		}
		return
	}
	event, _ := p["event"].(string)
	if event == "" {
		return
	}
	data, _ := p["data"].(map[string]any)
	if mid, ok := anyInt(p["machine_id"]); ok && mid != 0 {
		if event == "sync.nodes" {
			a.refreshMachineNodes(mid, data)
		}
		if m := a.hub.machine(mid); m != nil {
			m.enqueue(wsFrame(event, data))
		}
		return
	}
	if nid, ok := anyInt(p["node_id"]); ok && nid != 0 {
		if data == nil {
			data = map[string]any{}
		}
		a.sendNode(nid, event, data)
	}
}

// refreshMachineNodes 对应 NodeRegistry::refreshMachineNodes：宿主机节点列表变化后更新连接的节点集合。
func (a *App) refreshMachineNodes(mid int64, data map[string]any) {
	nodes, _ := data["nodes"].([]any)
	var ids []int64
	for _, n := range nodes {
		if m, ok := n.(map[string]any); ok {
			if id, ok := anyInt(m["id"]); ok {
				ids = append(ids, id)
			}
		}
	}
	if m := a.hub.machine(mid); m != nil {
		m.machineNodes = ids
	}
}

// wsRegister 挂载 /ws 路由（由 Router 调用）。
func (a *App) wsRegister(r *gin.Engine) {
	a.ensureHub()
	r.GET("/ws", a.wsHandler)
}

// ensureHub 惰性创建连接注册表（HTTP 路由与后台任务共用）。
func (a *App) ensureHub() {
	a.hubOnce.Do(func() {
		if a.hub == nil {
			a.hub = newWSHub()
		}
	})
}

// SyncTelegramCommands 启动时注册 Telegram 命令菜单（对外导出，供 cmd/xboard 调用）。
func (a *App) SyncTelegramCommands() { a.syncTelegramCommands() }
