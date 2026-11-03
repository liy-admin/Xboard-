package api

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
)

func md5hex(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }

const nodeKey = "node"

// serverAuth 对应 Server 中间件：校验 server_token，按 node_id(+node_type) 取节点。
func (a *App) serverAuth(c *gin.Context) {
	tok := c.Query("token")
	want := a.Set.Get("server_token", "")
	if want == "" || !auth.ConstEq(want, tok) {
		resp.Fail(c, 422, "Invalid token")
		return
	}
	nodeID := c.Query("node_id")
	if nodeID == "" {
		resp.Fail(c, 422, "node_id required")
		return
	}
	q := a.S.DB.Where("code = ? OR id = ?", nodeID, nodeID)
	if t := c.Query("node_type"); t != "" && t != "v2node" {
		q = q.Where("type = ?", strings.ToLower(t))
	}
	var n model.Server
	if err := q.Order(fmt.Sprintf("CASE WHEN code = '%s' THEN 0 ELSE 1 END", strings.ReplaceAll(nodeID, "'", ""))).First(&n).Error; err != nil {
		resp.Fail(c, 500, "Server does not exist")
		return
	}
	c.Set(nodeKey, &n)
	c.Next()
}

func node(c *gin.Context) *model.Server { return c.MustGet(nodeKey).(*model.Server) }

func (a *App) etagJSON(c *gin.Context, v any) {
	b, _ := json.Marshal(v)
	sum := sha1.Sum(b)
	etag := hex.EncodeToString(sum[:])
	if strings.Contains(c.GetHeader("If-None-Match"), etag) {
		c.Status(304)
		return
	}
	c.Header("ETag", `"`+etag+`"`)
	c.Data(200, "application/json", b)
}

func (a *App) nodeConfig(c *gin.Context) {
	a.etagJSON(c, a.nodeConfigMap(node(c)))
}

// nodeConfigMap 对应 ServerService::buildNodeConfig，HTTP 拉取与 WebSocket sync.config 共用。
func (a *App) nodeConfigMap(n *model.Server) map[string]any {
	ps := map[string]any{}
	_ = json.Unmarshal(n.ProtocolSettings, &ps)

	cfg := map[string]any{
		"protocol":        n.Type,
		"listen_ip":       "0.0.0.0",
		"server_port":     n.ServerPort,
		"network":         ps["network"],
		"networkSettings": ps["network_settings"],
	}
	// 按协议补充节点端所需字段（逐项对应 ServerService::buildNodeConfig）。
	switch n.Type {
	case "shadowsocks":
		cfg["cipher"], cfg["plugin"], cfg["plugin_opts"] = ps["cipher"], ps["plugin"], ps["plugin_opts"]
	case "vmess":
		cfg["tls"], cfg["tls_settings"], cfg["multiplex"] = ps["tls"], ps["tls_settings"], ps["multiplex"]
	case "trojan", "vless":
		cfg["tls"], cfg["multiplex"] = ps["tls"], ps["multiplex"]
		if f, _ := ps["tls"].(float64); f == 2 {
			cfg["tls_settings"] = ps["reality_settings"]
		} else {
			cfg["tls_settings"] = ps["tls_settings"]
		}
		if n.Type == "vless" {
			cfg["flow"] = ps["flow"]
		} else {
			cfg["host"] = n.Host
		}
	default: // hysteria / tuic / anytls / socks / naive / http：透传协议设置
		cfg["host"] = n.Host
		cfg["tls_settings"] = ps["tls"]
		for k, v := range ps {
			// 这些协议的 tls 是对象，已作为 tls_settings 下发；节点端的 tls 字段是整数，透传会解码失败。
			if _, isObj := v.(map[string]any); isObj && k == "tls" {
				continue
			}
			if _, ok := cfg[k]; !ok {
				cfg[k] = v
			}
		}
	}
	// 节点端按 cert_config 申请或生成证书（hysteria / tuic / anytls 等必须有证书才能启动）。
	if len(n.CertConfig) > 0 {
		var cert map[string]any
		if json.Unmarshal(n.CertConfig, &cert) == nil && len(cert) > 0 {
			cfg["cert_config"] = cert
		}
	}
	cfg["base_config"] = gin.H{
		"push_interval": a.Set.Int("server_push_interval", 60),
		"pull_interval": a.Set.Int("server_pull_interval", 60),
	}
	return cfg
}

type nodeUser struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	SpeedLimit  *int64 `json:"speed_limit"`
	DeviceLimit *int64 `json:"device_limit"`
}

func (a *App) availableUsers(n *model.Server) []nodeUser {
	var gids []int64
	_ = json.Unmarshal(n.GroupIDs, &gids)
	out := []nodeUser{}
	if len(gids) == 0 {
		return out
	}
	a.S.DB.Table("v2_user").
		Select("id, uuid, speed_limit, device_limit").
		Where("group_id IN ? AND (transfer_enable = 0 OR u + d < transfer_enable) AND (expired_at >= ? OR expired_at IS NULL) AND banned = 0", gids, now()).
		Scan(&out)
	return out
}

func (a *App) touch(c *gin.Context, n *model.Server, field string) {
	a.S.Redis.Set(c, fmt.Sprintf("SERVER_%s_%s_%d", strings.ToUpper(n.Type), field, n.ID), now(), time.Hour)
}

func (a *App) nodeUsers(c *gin.Context) {
	n := node(c)
	a.touch(c, n, "LAST_CHECK_AT")
	a.etagJSON(c, gin.H{"users": a.availableUsers(n)})
}

// nodePush: body 为 {"<uid>": [upload, download]}，按节点倍率累加到用户流量。
func (a *App) nodePush(c *gin.Context) {
	n := node(c)
	var data map[string][]float64
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.Fail(c, 422, "Invalid data format")
		return
	}
	rate := n.Rate
	if rate <= 0 {
		rate = 1
	}
	cnt := 0
	for uid, v := range data {
		if len(v) != 2 {
			continue
		}
		id, err := strconv.ParseInt(uid, 10, 64)
		if err != nil {
			continue
		}
		u, d := int64(v[0]*rate), int64(v[1]*rate)
		a.S.DB.Exec("UPDATE v2_user SET u = u + ?, d = d + ?, t = ?, updated_at = ? WHERE id = ?", u, d, now(), now(), id)
		cnt++
	}
	if cnt > 0 {
		key := fmt.Sprintf("SERVER_%s_ONLINE_USER_%d", strings.ToUpper(n.Type), n.ID)
		a.S.Redis.Set(c, key, cnt, time.Hour)
		a.touch(c, n, "LAST_PUSH_AT")
	}
	resp.OK(c, true)
}

func (a *App) nodeStatus(c *gin.Context) {
	n := node(c)
	var in map[string]any
	if err := c.ShouldBindJSON(&in); err != nil {
		resp.Fail(c, 422, "Invalid data")
		return
	}
	in["updated_at"] = now()
	b, _ := json.Marshal(in)
	a.S.Redis.Set(c, fmt.Sprintf("SERVER_%s_LOAD_STATUS_%d", strings.ToUpper(n.Type), n.ID), string(b), time.Hour)
	c.JSON(200, gin.H{"data": true, "code": 0, "message": "success"})
}
