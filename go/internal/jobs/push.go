package jobs

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// redisPusher 与 NodeSyncService 的 Redis 协议保持一致：
// 在线状态键 node_ws_alive:<id>，推送频道 node:push（消息体 {node_id, event, data}）。
type redisPusher struct{ c *redis.Client }

// NewRedisPusher 返回基于 Redis 的节点推送器。
func NewRedisPusher(c *redis.Client) NodePusher { return redisPusher{c} }

func (p redisPusher) IsOnline(serverID int64) bool {
	v, err := p.c.Get(context.Background(), "node_ws_alive:"+strconv.FormatInt(serverID, 10)).Result()
	return err == nil && v != "" && v != "0"
}

func (p redisPusher) PushRemoveUsers(serverID int64, userIDs []int64) error {
	users := make([]map[string]int64, 0, len(userIDs))
	for _, id := range userIDs {
		users = append(users, map[string]int64{"id": id})
	}
	body, err := json.Marshal(map[string]any{
		"node_id": serverID,
		"event":   "sync.user.delta",
		"data":    map[string]any{"action": "remove", "users": users},
	})
	if err != nil {
		return err
	}
	return p.c.Publish(context.Background(), "node:push", body).Err()
}

// BroadcastSubscriptionUpdated 向 client:push 发布「订阅已更新」，由 WebSocket 进程转发给在线客户端。
func (p redisPusher) BroadcastSubscriptionUpdated(reason string) error {
	body, err := json.Marshal(map[string]any{
		"scope": "all", "event": "subscription.updated", "data": map[string]any{"reason": reason},
	})
	if err != nil {
		return err
	}
	return p.c.Publish(context.Background(), "client:push", body).Err()
}
