package api

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"
)

// 在线设备数的数据库回写。
//
// 设备明细只存在 Redis 哈希 user_devices:<uid>（字段 "<节点ID>:<IP>"），而管理端用户列表、
// 「在线设备」筛选、仪表盘统计读的都是 v2_user.online_count / last_online_at。
// PHP 的 DeviceStateService::notifyUpdate 在每次设备变化后把 Redis 里的设备数写回这两列；
// Go 移植时漏掉了这一步，导致这两列从不更新，列表永远显示「0 / N」、「在线设备」筛选永远无结果。

// countDistinctDevices 统计未过期的不同 IP 数：同一个 IP 连多个节点只算一台设备，
// 对应 DeviceStateService::getDeviceCount。nowTs 注入便于测试。
func countDistinctDevices(data map[string]string, nowTs int64) int {
	ips := map[string]struct{}{}
	for field, v := range data {
		ts, err := strconv.ParseInt(v, 10, 64)
		if err != nil || nowTs-ts > deviceTTL {
			continue
		}
		// 字段是 "<节点ID>:<IP>"，IPv6 本身含冒号，所以只在第一个冒号处切。
		_, ip, ok := strings.Cut(field, ":")
		if !ok || ip == "" {
			continue
		}
		ips[ip] = struct{}{}
	}
	return len(ips)
}

// syncOnlineCount 把用户当前的在线设备数写回 v2_user.online_count，并刷新 last_online_at，
// 对应 DeviceStateService::notifyUpdate。设备数为 0 时同样写入（例如节点下线后清零）。
func (a *App) syncOnlineCount(ctx context.Context, uids ...int64) {
	for _, uid := range uids {
		data, err := a.S.Redis.HGetAll(ctx, devicePrefix+strconv.FormatInt(uid, 10)).Result()
		if err != nil {
			log.Printf("syncOnlineCount: 读取用户 %d 的设备状态失败: %v", uid, err)
			continue
		}
		if err := a.S.DB.Exec("UPDATE v2_user SET online_count = ?, last_online_at = ? WHERE id = ?",
			countDistinctDevices(data, now()), time.Now(), uid).Error; err != nil {
			log.Printf("syncOnlineCount: 更新用户 %d 失败: %v", uid, err)
		}
	}
}
