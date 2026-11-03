package api

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/resp"
)

// 设备在线状态存于 Redis 哈希 user_devices:<uid>，字段为 "<nodeId>:<ip>"，值为最后上报时间戳。
const (
	devicePrefix = "user_devices:"
	deviceTTL    = 300 // 秒，与 DeviceStateService::TTL 一致
)

var (
	ipv6PortRe = regexp.MustCompile(`^\[(.+)\]:\d+$`)
	ipv4PortRe = regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+):\d+$`)
)

// normalizeDeviceIP 去掉端口后缀（[v6]:port 与 v4:port），对应 DeviceStateService::normalizeIP。
func normalizeDeviceIP(ip string) string {
	if m := ipv6PortRe.FindStringSubmatch(ip); m != nil {
		return m[1]
	}
	if m := ipv4PortRe.FindStringSubmatch(ip); m != nil {
		return m[1]
	}
	return ip
}

type userDevice struct {
	NodeID   int64
	IP       string
	LastSeen int64
}

// parseUserDevices 解析 Redis 哈希内容，过滤超过 TTL 的记录。nowTs 注入便于测试。
func parseUserDevices(data map[string]string, nowTs int64) []userDevice {
	out := make([]userDevice, 0, len(data))
	for field, v := range data {
		ts, err := strconv.ParseInt(v, 10, 64)
		if err != nil || nowTs-ts > deviceTTL {
			continue
		}
		node, ip, ok := strings.Cut(field, ":")
		if !ok {
			continue
		}
		nid, _ := strconv.ParseInt(node, 10, 64)
		out = append(out, userDevice{NodeID: nid, IP: ip, LastSeen: ts})
	}
	return out
}

func (a *App) deviceFetch(c *gin.Context) {
	u := cur(c)
	data, err := a.S.Redis.HGetAll(bgCtx(), devicePrefix+strconv.FormatInt(u.ID, 10)).Result()
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	devices := parseUserDevices(data, time.Now().Unix())

	ids := make([]int64, 0, len(devices))
	seen := map[int64]bool{}
	for _, d := range devices {
		if !seen[d.NodeID] {
			seen[d.NodeID] = true
			ids = append(ids, d.NodeID)
		}
	}
	names := map[int64]string{}
	if len(ids) > 0 {
		var servers []model.Server
		a.S.DB.Select("id", "name").Where("id IN ?", ids).Find(&servers)
		for _, s := range servers {
			names[s.ID] = s.Name
		}
	}
	out := make([]gin.H, 0, len(devices))
	for _, d := range devices {
		var name any
		if n, ok := names[d.NodeID]; ok {
			name = n
		}
		out = append(out, gin.H{"node_id": d.NodeID, "node_name": name, "ip": d.IP, "last_seen_at": d.LastSeen})
	}
	resp.OK(c, out)
}

func (a *App) deviceKick(c *gin.Context) {
	u := cur(c)
	var in struct {
		NodeID int64  `json:"node_id" form:"node_id"`
		IP     string `json:"ip" form:"ip"`
	}
	_ = c.ShouldBind(&in)
	if in.NodeID <= 0 || in.IP == "" {
		resp.Fail(c, 400, "参数错误")
		return
	}
	// 只清理追踪记录腾出名额，不保证断开该设备当前连接（与 PHP 语义一致）。
	a.S.Redis.HDel(bgCtx(), devicePrefix+strconv.FormatInt(u.ID, 10),
		strconv.FormatInt(in.NodeID, 10)+":"+in.IP)
	a.syncOnlineCount(bgCtx(), u.ID)
	resp.OK(c, true)
}

// bindDevice 对应 DeviceBindingService::bind：登录/注册时把设备绑定到真实用户，不改 claimed_trial。
func (a *App) bindDevice(deviceID string, u *model.User) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return
	}
	var b model.DeviceBinding
	if err := a.S.DB.Where("device_id = ?", deviceID).First(&b).Error; err != nil {
		ts := now()
		a.S.DB.Create(&model.DeviceBinding{DeviceID: deviceID, UserID: u.ID, Type: "trial", CreatedAt: ts, UpdatedAt: ts})
		return
	}
	if b.UserID != u.ID {
		a.S.DB.Model(&b).Updates(map[string]any{"user_id": u.ID, "updated_at": now()})
	}
}
