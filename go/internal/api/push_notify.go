package api

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 后台改动节点、套餐、权限组、路由，或导入节点后，通知在线客户端重新拉取订阅（只发信号，不带数据）。
//
// 合并发送：最后一次改动之后再等 clientPushDelay 才发一次。连续保存多个节点只推一次，
// 而且推出去时数据库里已经是改完后的最终状态。是否真正发出由「客户端推送」开关（client_ws_enable）决定。

// clientPushDelay 为合并窗口；测试里会调小。
var clientPushDelay = 2 * time.Second

type pushDebouncer struct {
	mu      sync.Mutex
	timer   *time.Timer
	reasons map[string]bool
}

var debouncers sync.Map // *App -> *pushDebouncer

// pushNowHook 仅供测试替换实际发送。
var pushNowHook func(a *App, reason string)

// notifyClientsSoon 安排一次推送；窗口内的多次调用合并为一次，原因去重后用逗号连接。
func (a *App) notifyClientsSoon(reason string) {
	v, _ := debouncers.LoadOrStore(a, &pushDebouncer{})
	d := v.(*pushDebouncer)
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.reasons == nil {
		d.reasons = map[string]bool{}
	}
	d.reasons[reason] = true
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(clientPushDelay, func() {
		d.mu.Lock()
		rs := make([]string, 0, len(d.reasons))
		for r := range d.reasons {
			rs = append(rs, r)
		}
		d.reasons = map[string]bool{}
		d.timer = nil
		d.mu.Unlock()
		sort.Strings(rs)
		a.pushNow(strings.Join(rs, ","))
	})
}

// pushNow 经 Redis 发布「订阅已更新」，由各实例的 WebSocket 循环转发给在线客户端（多实例部署也能覆盖全部客户端）。
func (a *App) pushNow(reason string) {
	if pushNowHook != nil {
		pushNowHook(a, reason)
		return
	}
	a.jobsEnv().NotifyClients(reason)
}

// pushAfter 中间件：接口处理成功（2xx）后安排一次客户端推送；失败的请求不推送。
func (a *App) pushAfter(reason string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if s := c.Writer.Status(); s >= 200 && s < 300 {
			a.notifyClientsSoon(reason)
		}
	}
}
