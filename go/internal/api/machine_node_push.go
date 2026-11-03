package api

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
)

// pushMachineAfterNode 在节点增删改成功后通知相关宿主机：涉及的每台宿主机收到 sync.nodes
// 立即重新发现节点（新增、删除、启停、换绑都即时生效），仍托管在宿主机上的单个节点再收到
// sync.config 立即应用新配置。否则宿主机要等下一次轮询（默认 60 秒）。
// 请求里的 id / ids 在改动前查出原宿主机，machine_id 参数是新宿主机。
func (a *App) pushMachineAfterNode() gin.HandlerFunc {
	return func(c *gin.Context) {
		m := bodyMap(c)
		var ids []int64
		if id, ok := mInt(m, "id"); ok && id > 0 {
			ids = append(ids, id)
		}
		if list, ok := m["ids"].([]any); ok {
			for _, v := range list {
				if id, ok := anyInt(v); ok && id > 0 {
					ids = append(ids, id)
				}
			}
		}
		machines := map[int64]bool{}
		if v, ok := mInt(m, "machine_id"); ok && v > 0 {
			machines[v] = true
		}
		for _, mid := range a.nodeMachineIDs(ids) {
			machines[mid] = true
		}

		c.Next()

		if s := c.Writer.Status(); s < 200 || s >= 300 {
			return
		}
		for _, mid := range a.nodeMachineIDs(ids) {
			machines[mid] = true
		}
		for mid := range machines {
			a.pushMachineNodes(mid)
		}
		if len(ids) == 1 {
			a.pushNodeConfig(ids[0])
		}
	}
}

// nodeMachineIDs 返回这些节点当前绑定的宿主机（去掉未绑定的）。
func (a *App) nodeMachineIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	var out []int64
	a.S.DB.Model(&model.Server{}).Where("id IN ? AND machine_id IS NOT NULL AND machine_id > 0", ids).
		Distinct().Pluck("machine_id", &out)
	return out
}

// pushNodeConfig 向托管在宿主机上且启用的节点下发最新配置。
func (a *App) pushNodeConfig(id int64) {
	if a.S.Redis == nil {
		return
	}
	var s model.Server
	if err := a.S.DB.First(&s, id).Error; err != nil || s.MachineID == nil || *s.MachineID == 0 {
		return
	}
	if s.Enabled != nil && !*s.Enabled {
		return
	}
	body, err := json.Marshal(gin.H{"node_id": s.ID, "event": "sync.config", "data": gin.H{"config": a.nodeConfigMap(&s)}})
	if err != nil {
		return
	}
	a.S.Redis.Publish(context.Background(), wsNodePushChan, body)
}
