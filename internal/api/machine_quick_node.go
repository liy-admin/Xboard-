package api

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/sub"
)

// 宿主机「一键创建节点」与宿主机来源 IP 记录。PHP 版没有此功能，属于新增。
//
// 来源 IP 存在 Redis（machine_last_ip:<id>），不加数据库列：`xboard migrate` 只建缺失的表、
// 不给已有表加列，加列会让已部署的库与代码不一致。

const (
	machineLastIPTTL          = 30 * 24 * time.Hour
	quickNodeDefaultPort      = 443
	quickNodeDefaultSNI       = "www.apple.com"
	quickNodeRealityHandshake = "443"
)

func machineLastIPKey(id int64) string { return fmt.Sprintf("machine_last_ip:%d", id) }

// recordMachineContact 记录宿主机的一次请求：更新最后心跳时间，并记下来源 IP（供一键创建节点做默认地址）。
func (a *App) recordMachineContact(m *model.ServerMachine, ip string) {
	a.S.DB.Model(m).Update("last_seen_at", now())
	if net.ParseIP(ip) != nil && a.S.Redis != nil {
		a.S.Redis.Set(context.Background(), machineLastIPKey(m.ID), ip, machineLastIPTTL)
	}
}

// machineLastIP 返回宿主机最近一次请求的来源 IP，没有记录时返回空串。
func (a *App) machineLastIP(id int64) string {
	if a.S.Redis == nil {
		return ""
	}
	ip, err := a.S.Redis.Get(context.Background(), machineLastIPKey(id)).Result()
	if err != nil {
		return ""
	}
	return ip
}

// machineQuickNode 在宿主机下一键创建 VLESS + Reality 节点（xtls-rprx-vision、uTLS chrome），
// Reality 密钥对由面板生成。管理员只需选权限组；地址缺省为宿主机最近的来源 IP，端口 443，
// 名称为宿主机名，目标站点 www.apple.com（www.microsoft.com 目前的握手与内核的 Reality 不兼容，连接会被判为无效）。同一宿主机上已有节点占用该端口时拒绝。
func (a *App) machineQuickNode(c *gin.Context) {
	m := bodyMap(c)
	id, _ := mInt(m, "machine_id")
	machine, ok := a.machineByID(c, id)
	if !ok {
		return
	}

	groupIDs := quickNodeGroupIDs(m["group_ids"])
	if len(groupIDs) == 0 {
		resp.Fail(c, 422, "请至少选择一个权限组")
		return
	}
	var found int64
	a.S.DB.Model(&model.ServerGroup{}).Where("id IN ?", groupIDs).Count(&found)
	if int(found) != len(groupIDs) {
		resp.Fail(c, 422, "权限组不存在")
		return
	}

	host, _ := mStr(m, "host")
	host = strings.TrimSpace(host)
	if host == "" {
		host = a.machineLastIP(machine.ID)
	}
	if host == "" {
		resp.Fail(c, 422, "这台服务器还没有连接过面板，无法自动获取地址，请填写节点地址")
		return
	}
	if !validNodeHost(host) {
		resp.Fail(c, 422, "节点地址必须是 IP 或域名")
		return
	}

	port := int64(quickNodeDefaultPort)
	if v, ok := mInt(m, "port"); ok {
		port = v
	}
	if port < 1 || port > 65535 {
		resp.Fail(c, 422, "端口必须在 1-65535 之间")
		return
	}
	var conflict model.Server
	if a.S.DB.Select("id", "name").Where("machine_id = ? AND server_port = ?", machine.ID, port).First(&conflict).Error == nil {
		resp.Fail(c, 422, fmt.Sprintf("端口 %d 已被这台服务器上的节点 #%d（%s）使用", port, conflict.ID, conflict.Name))
		return
	}

	sni, _ := mStr(m, "server_name")
	sni = strings.TrimSpace(sni)
	if sni == "" {
		sni = quickNodeDefaultSNI
	}
	if net.ParseIP(sni) != nil || !validNodeHost(sni) {
		resp.Fail(c, 422, "Reality 目标站点必须是域名")
		return
	}

	name, _ := mStr(m, "name")
	name = strings.TrimSpace(name)
	if name == "" {
		name = machine.Name
	}
	if len([]rune(name)) > 255 {
		name = string([]rune(name)[:255])
	}
	show := true
	if v, ok := mBool(m, "show"); ok {
		show = v
	}

	priv, pub, err := realityKeyPair()
	if err != nil {
		resp.Fail(c, 500, "生成 Reality 密钥失败")
		return
	}
	shortID := make([]byte, 8)
	if _, err := rand.Read(shortID); err != nil {
		resp.Fail(c, 500, "生成 Reality 密钥失败")
		return
	}
	settings := sub.NormalizeProtocolSettings("vless", map[string]any{
		"tls":     2,
		"network": "tcp",
		"flow":    "xtls-rprx-vision",
		"reality_settings": map[string]any{
			"server_name":    sni,
			"server_port":    quickNodeRealityHandshake,
			"public_key":     pub,
			"private_key":    priv,
			"short_id":       hex.EncodeToString(shortID),
			"allow_insecure": false,
		},
		"utls": map[string]any{"enabled": true, "fingerprint": "chrome"},
	})
	settingsJSON, _ := json.Marshal(settings)
	// 与 nodeSave 一致：group_ids 存整数，订阅与节点端按 JSON_CONTAINS(group_ids, 数字) 匹配。
	groupsJSON, _ := json.Marshal(groupIDs)

	enabled := true
	machineID := machine.ID
	ts := time.Now()
	srv := model.Server{
		Type:             "vless",
		Name:             name,
		GroupIDs:         groupsJSON,
		RouteIDs:         json.RawMessage("[]"),
		Rate:             1,
		Host:             host,
		Port:             fmt.Sprint(port),
		ServerPort:       int(port),
		ProtocolSettings: settingsJSON,
		Show:             show,
		Enabled:          &enabled,
		MachineID:        &machineID,
		CreatedAt:        ts,
		UpdatedAt:        ts,
	}
	if err := a.S.DB.Create(&srv).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}

	a.pushMachineNodes(machine.ID)
	resp.OK(c, gin.H{"id": srv.ID, "name": srv.Name, "host": srv.Host, "port": srv.ServerPort})
}

// pushMachineNodes 经 Redis node:push 发布 sync.nodes，在线的宿主机立即重新发现节点；
// 不在线或 Redis 不可用时，宿主机按轮询间隔自行拉取。
func (a *App) pushMachineNodes(machineID int64) {
	if a.S.Redis == nil {
		return
	}
	var servers []model.Server
	a.S.DB.Select("id", "type", "name").
		Where("machine_id = ? AND (enabled = 1 OR enabled IS NULL)", machineID).
		Order("sort ASC").Find(&servers)
	nodes := make([]gin.H, 0, len(servers))
	for _, s := range servers {
		nodes = append(nodes, gin.H{"id": s.ID, "type": s.Type, "name": s.Name})
	}
	body, err := json.Marshal(gin.H{"machine_id": machineID, "event": "sync.nodes", "data": gin.H{"nodes": nodes}})
	if err != nil {
		return
	}
	a.S.Redis.Publish(context.Background(), wsNodePushChan, body)
}

// quickNodeGroupIDs 把权限组参数（数字或数字字符串的数组）转成去重后的正整数列表。
func quickNodeGroupIDs(v any) []int64 {
	items, _ := v.([]any)
	seen := map[int64]bool{}
	out := make([]int64, 0, len(items))
	for _, it := range items {
		id, ok := anyInt(it)
		if !ok || id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// validNodeHost 判断是否为 IP，或至少含一个点的合法主机名。
func validNodeHost(h string) bool {
	if net.ParseIP(h) != nil {
		return true
	}
	if len(h) > 253 || !strings.Contains(h, ".") {
		return false
	}
	for _, label := range strings.Split(h, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

// realityKeyPair 生成 Reality 使用的 X25519 密钥对：32 字节，base64 URL 安全编码、无填充，
// 与 `xray x25519` 输出格式一致。私钥按 X25519 规则做 clamp，保存规范形式。
func realityKeyPair() (priv, pub string, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	raw[0] &= 248
	raw[31] &= 127
	raw[31] |= 64
	key, err := ecdh.X25519().NewPrivateKey(raw)
	if err != nil {
		return "", "", err
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString(raw), enc.EncodeToString(key.PublicKey().Bytes()), nil
}
