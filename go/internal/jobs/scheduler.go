package jobs

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// schedule 描述一个定时任务，对应 app/Console/Kernel.php 中的一行。
// Every 为固定间隔（分钟级对齐）；DailyAt 为每日固定时刻（本地时区）。二者只设一个。
type schedule struct {
	Name    string
	Every   time.Duration
	DailyAt string // "HH:MM"
	// TimeFn 非空时，每次检查都用它返回的 "HH:MM" 覆盖 DailyAt（时刻来自后台设置，改动无需重启）。
	TimeFn  func(*Env) string
	Overlap time.Duration
}

// DefaultSchedules 与 Kernel::schedule 中已迁移的任务一致。
// xboard:statistics、send:remindMail、check:server-latency 依赖的模块尚未迁移，见 PROGRESS.md。
var DefaultSchedules = []schedule{
	{Name: "check:order", Every: time.Minute, Overlap: 5 * time.Minute},
	{Name: "check:commission", Every: time.Minute, Overlap: 5 * time.Minute},
	{Name: "check:ticket", Every: time.Minute, Overlap: 5 * time.Minute},
	{Name: "check:traffic-exceeded", Every: time.Minute, Overlap: 10 * time.Minute},
	{Name: "reset:traffic", Every: time.Minute, Overlap: 10 * time.Minute},
	{Name: "reset:log", DailyAt: "00:00", Overlap: 30 * time.Minute},
	{Name: "cleanup:online-status", Every: 5 * time.Minute, Overlap: 5 * time.Minute},
	{Name: "check:server-latency", Every: time.Minute, Overlap: 5 * time.Minute},
	{Name: "server:import-subscriptions", Every: time.Hour, Overlap: 30 * time.Minute},
	{Name: "xboard:statistics", DailyAt: "00:10", Overlap: 30 * time.Minute},
	// 可用国家快照：每天一次，供没有会员的用户查看（管理员可在后台手动更新）。
	{Name: "refresh:countries", DailyAt: DefaultCountryRefreshTime, TimeFn: func(e *Env) string { return e.CountryRefreshTime() }, Overlap: 30 * time.Minute},
	// 已取消订单清理：每天一次；开关在后台「订单管理」，关闭时任务直接返回，不动任何数据。
	{Name: "cleanup:cancelled-orders", DailyAt: CleanupAt, Overlap: 30 * time.Minute},
	{Name: "send:remindMail", DailyAt: "11:30", Overlap: 30 * time.Minute},
	// BalalaSub 插件注册的任务：每分钟触发，命令内部按插件启用状态、开关与间隔决定是否执行。
	{Name: "balala-sub:import", Every: time.Minute, Overlap: 30 * time.Minute},
}

// resolve 套用动态时刻（TimeFn），返回本次检查实际使用的任务表项。
func (s schedule) resolve(e *Env) schedule {
	if s.TimeFn != nil {
		s.DailyAt = s.TimeFn(e)
	}
	return s
}

// due 判断 s 在时刻 t 是否触发，并返回事件键（用于跨实例去重，等价于 onOneServer）。
func (s schedule) due(t time.Time) (bool, string) {
	if s.DailyAt != "" {
		if t.Format("15:04") == s.DailyAt {
			return true, t.Format("2006-01-02")
		}
		return false, ""
	}
	if s.Every > 0 {
		slot := t.Unix() / int64(s.Every.Seconds())
		return t.Unix()%int64(s.Every.Seconds()) < 1, strconv.FormatInt(slot, 10)
	}
	return false, ""
}

// Scheduler 是内置调度器：每秒检查一次到期任务，每个事件在多实例间只执行一次（Redis 锁）。
type Scheduler struct {
	Env       *Env
	Redis     *redis.Client
	Schedules []schedule
	last      map[string]string
}

// NewScheduler 构造默认任务表的调度器。
func NewScheduler(env *Env, rdb *redis.Client) *Scheduler {
	return &Scheduler{Env: env, Redis: rdb, Schedules: DefaultSchedules, last: map[string]string{}}
}

// Run 阻塞运行直到 ctx 结束。
func (s *Scheduler) Run(ctx context.Context) {
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-tk.C:
			s.tick(ctx, t)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context, t time.Time) {
	if t.Second() == 0 && s.Redis != nil {
		s.Redis.Set(ctx, "SCHEDULE_LAST_CHECK_AT", t.Unix(), 10*time.Minute)
	}
	for _, sc := range s.Schedules {
		sc = sc.resolve(s.Env)
		ok, key := sc.due(t)
		if !ok || s.last[sc.Name] == key {
			continue
		}
		s.last[sc.Name] = key
		lock := fmt.Sprintf("xboard:cron:%s:%s", sc.Name, key)
		got, err := s.Redis.SetNX(ctx, lock, t.Unix(), 10*time.Minute).Result()
		if err != nil || !got {
			continue // 其他实例已抢到，或 Redis 不可用时跳过本次
		}
		name := sc.Name
		go func() {
			start := time.Now()
			var err error
			if name == "check:server-latency" {
				// 调度器按 server_latency_probe_interval 节流；CLI 手动执行 check:server-latency 不受限。
				err = s.Env.ScheduledLatency()
			} else {
				err = s.Env.Run(name)
			}
			if err != nil {
				log.Printf("job %s failed: %v", name, err)
				return
			}
			log.Printf("job %s done in %s", name, time.Since(start).Round(time.Millisecond))
		}()
	}
}
