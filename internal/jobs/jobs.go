package jobs

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"xboard/internal/model"
	"xboard/internal/setting"
	"xboard/internal/shop"
)

// Env 是所有任务共享的依赖。
type Env struct {
	DB    *gorm.DB
	Redis RedisClient
	Set   *setting.Service
	Shop  *shop.Service
	// Push 向节点推送用户增量（对应 NodeSyncService::push）；节点 WebSocket 未实现前为 nil，任务会跳过推送。
	Push NodePusher
	// Now 便于测试注入时间。
	Now func() time.Time
}

// NodePusher 抽象节点推送，返回是否在线并已推送。
type NodePusher interface {
	IsOnline(serverID int64) bool
	PushRemoveUsers(serverID int64, userIDs []int64) error
}

// RedisClient 是任务需要的最小 Redis 能力，便于测试替换。
type RedisClient interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	SCard(ctx context.Context, key string) (int64, error)
	SPopN(ctx context.Context, key string, n int64) ([]string, error)
	SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
	Del(ctx context.Context, keys ...string) error
}

const (
	orderStatusPending    = 0
	orderStatusProcessing = 1
	ticketOpen            = 0
	ticketClosed          = 1
	ticketReplied         = 1
)

func (e *Env) nowTs() int64 { return e.Now().Unix() }

// CheckOrder 对应 check:order：待支付超过 2 小时取消，处理中的订单尝试开通。
func (e *Env) CheckOrder() error {
	var orders []model.Order
	if err := e.DB.Where("status IN ?", []int{orderStatusPending, orderStatusProcessing}).
		Order("created_at ASC").Find(&orders).Error; err != nil {
		return err
	}
	for i := range orders {
		o := &orders[i]
		var err error
		switch o.Status {
		case orderStatusPending:
			if o.CreatedAt <= e.nowTs()-2*3600 {
				err = e.Shop.Cancel(o)
			}
		case orderStatusProcessing:
			err = e.Shop.Open(o)
		}
		if err != nil {
			log.Printf("check:order %s: %v", o.TradeNo, err)
		}
	}
	return nil
}

// CheckTicket 对应 check:ticket：管理员已回复且超过 24 小时未有用户跟进的工单自动关闭。
func (e *Env) CheckTicket() error {
	var tickets []model.Ticket
	if err := e.DB.Where("status = ? AND updated_at <= ? AND reply_status = ?",
		ticketOpen, e.nowTs()-24*3600, ticketReplied).Find(&tickets).Error; err != nil {
		return err
	}
	for _, t := range tickets {
		if t.UserID == t.LastReplyUserID {
			continue
		}
		if err := e.DB.Model(&model.Ticket{}).Where("id = ?", t.ID).
			Updates(map[string]any{"status": ticketClosed, "updated_at": e.nowTs()}).Error; err != nil {
			return err
		}
	}
	return nil
}

// CheckTrafficExceeded 对应 check:traffic-exceeded：取出待检查用户，超额的从其所在分组的在线节点上移除。
func (e *Env) CheckTrafficExceeded() error {
	ctx := context.Background()
	const key = "traffic:pending_check"
	n, err := e.Redis.SCard(ctx, key)
	if err != nil || n <= 0 {
		return err
	}
	raw, err := e.Redis.SPopN(ctx, key, n)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(raw))
	for _, s := range raw {
		var id int64
		if _, err := fmtSscan(s, &id); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var users []model.User
	if err := e.DB.Select("id", "group_id").
		Where("id IN ? AND u + d >= transfer_enable AND transfer_enable > 0 AND banned = 0", ids).
		Find(&users).Error; err != nil {
		return err
	}
	byGroup := map[int64][]int64{}
	for _, u := range users {
		if u.GroupID != nil && *u.GroupID != 0 {
			byGroup[*u.GroupID] = append(byGroup[*u.GroupID], u.ID)
		}
	}
	if e.Push == nil {
		return nil
	}
	for gid, uids := range byGroup {
		var servers []model.Server
		// group_ids 为 JSON 数组，等价于 JSON_CONTAINS(group_ids, gid)。
		if err := e.DB.Where("JSON_CONTAINS(group_ids, ?)", jsonInt(gid)).Find(&servers).Error; err != nil {
			return err
		}
		for _, s := range servers {
			if !e.Push.IsOnline(s.ID) {
				continue
			}
			if err := e.Push.PushRemoveUsers(s.ID, uids); err != nil {
				log.Printf("check:traffic-exceeded push server %d: %v", s.ID, err)
			}
		}
	}
	return nil
}

// ResetTraffic 对应 reset:traffic：按 next_reset_at 到期的有效会员执行流量重置。
func (e *Env) ResetTraffic() error {
	ts := e.nowTs()
	var users []model.User
	if err := e.DB.Where("next_reset_at IS NOT NULL AND next_reset_at <= ? AND (expired_at > ? OR expired_at IS NULL) AND banned = 0 AND plan_id IS NOT NULL", ts, ts).
		Find(&users).Error; err != nil {
		return err
	}
	for i := range users {
		u := &users[i]
		err := e.DB.Transaction(func(tx *gorm.DB) error {
			return e.Shop.PerformReset(tx, u, shop.SourceCron)
		})
		if err != nil {
			log.Printf("reset:traffic user %d: %v", u.ID, err)
		}
	}
	return nil
}

// ResetLog 对应 reset:log：清理两个月前的流量统计与三个月前的审计日志。
func (e *Env) ResetLog() error {
	ts := e.nowTs()
	if err := e.DB.Exec("DELETE FROM v2_stat_user WHERE record_at < ?", ts-2*30*86400).Error; err != nil {
		return err
	}
	if err := e.DB.Exec("DELETE FROM v2_stat_server WHERE record_at < ?", ts-2*30*86400).Error; err != nil {
		return err
	}
	return e.DB.Exec("DELETE FROM v2_admin_audit_log WHERE created_at < ?", ts-3*30*86400).Error
}

// onlineStaleSeconds 是在线数的有效期：设备明细在 Redis 里只保留 300 秒，
// 超过这个时间没有任何上报，v2_user.online_count 就不再有依据，应当清零。
// （PHP 用 10 分钟；这里与 Redis TTL 对齐，列表上的在线数过期得更及时。）
const onlineStaleSeconds = 300

// CleanupOnlineStatus 对应 cleanup:online-status：长时间没有心跳的用户清零在线数。
//
// last_online_at 是 MySQL timestamp 列，必须用时间值比较。此前传的是 Unix 整数，
// MySQL 会把它当成 YYYYMMDDHHMMSS 数字去比，条件永远不成立，过期的在线数从不清零。
func (e *Env) CleanupOnlineStatus() error {
	cutoff := time.Unix(e.nowTs()-onlineStaleSeconds, 0)
	return e.DB.Exec("UPDATE v2_user SET online_count = 0 WHERE online_count > 0 AND (last_online_at < ? OR last_online_at IS NULL)", cutoff).Error
}

// Run 按名字执行任务，供 CLI 与调度器共用。
func (e *Env) Run(name string) error {
	switch name {
	case "check:order":
		return e.CheckOrder()
	case "check:commission":
		return CheckCommission(e.DB, e.Set, e.nowTs())
	case "check:ticket":
		return e.CheckTicket()
	case "check:traffic-exceeded":
		return e.CheckTrafficExceeded()
	case "reset:traffic":
		return e.ResetTraffic()
	case "reset:log":
		return e.ResetLog()
	case "cleanup:online-status":
		return e.CleanupOnlineStatus()
	case "check:server-latency":
		return e.runLatency(nil, false)
	case "server:import-subscriptions":
		return e.ImportSubscriptions("", false)
	case "send:remindMail":
		return e.SendRemindMail()
	case "xboard:statistics":
		return e.StatisticsDaily()
	case "refresh:countries":
		e.RefreshCountries("daily")
		return nil
	case "cleanup:cancelled-orders":
		return e.ScheduledCleanupCancelled()
	case "balala-sub:import":
		return e.BalalaImport(false)
	case "balala-sub:import-force":
		return e.BalalaImport(true)
	case "balala-sub:fetch":
		_, err := e.BalalaFetch(context.Background(), e.LoadBalalaConfig())
		return err
	}
	return errUnknownJob(name)
}

type errUnknownJob string

func (e errUnknownJob) Error() string { return "unknown job: " + string(e) }

// runnable 列出可通过 CLI 直接执行的任务名。
var runnable = map[string]bool{
	"check:order": true, "check:commission": true, "check:ticket": true,
	"check:traffic-exceeded": true, "reset:traffic": true, "reset:log": true,
	"cleanup:online-status": true, "send:remindMail": true, "xboard:statistics": true, "check:server-latency": true, "refresh:countries": true, "cleanup:cancelled-orders": true, "server:import-subscriptions": true, "balala-sub:fetch": true, "balala-sub:import": true, "balala-sub:import-force": true,
}

// schedulerOnly 列出只由调度器触发、不提供命令行入口的任务（新功能一律走后台按钮 + 调度器，不加命令）。
var schedulerOnly = map[string]bool{"refresh:countries": true, "cleanup:cancelled-orders": true}

// IsRunnable 判断任务名是否已实现（调度器可用）。
func IsRunnable(name string) bool { return runnable[name] }

// IsCLI 判断任务能否从命令行直接执行。
func IsCLI(name string) bool { return runnable[name] && !schedulerOnly[name] }

// Names 返回已实现的任务名（无序）。
func Names() []string {
	out := make([]string, 0, len(runnable))
	for n := range runnable {
		if IsCLI(n) {
			out = append(out, n)
		}
	}
	return out
}
