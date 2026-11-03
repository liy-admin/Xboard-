package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// 已取消订单的定时清理。
//
// 规则：只删除状态为「已取消」的订单，并且以取消时间（updated_at，取消时会刷新）为准，
// 距今超过 N 天才删——不是下单时间，所以刚取消的订单不会因为下单早而被误删。
// 已支付、待支付、开通中、已折抵的订单永远不会被这个任务碰。
// 开关与天数在后台「订单管理」里设置；调度器每天 CleanupAt 触发一次，关闭时什么都不做。
// 新功能约定：只做后台按钮 + 调度器，不提供命令行入口（见 schedulerOnly）。

const (
	SettingCleanupEnable = "order_cancelled_cleanup_enable"
	SettingCleanupDays   = "order_cancelled_cleanup_days"
	settingCleanupLast   = "order_cancelled_cleanup_last"

	// DefaultCleanupDays 是缺省保留天数。
	DefaultCleanupDays = 30
	MinCleanupDays     = 1
	MaxCleanupDays     = 3650

	// CleanupAt 是每日自动清理的时刻（服务器本地时区）。选在凌晨低峰，避开 00:00 日志清理与 00:10 统计。
	CleanupAt = "03:30"

	orderStatusCancelled = 2

	cleanupBatch    = 1000   // 每批删除条数，避免一次长事务锁表
	cleanupMaxTotal = 200000 // 单次运行的上限，剩余的留给下一次
)

// CleanupResult 是一次清理的结果，也是「上次清理」展示的内容。
type CleanupResult struct {
	At      int64  `json:"at"`
	Deleted int64  `json:"deleted"`
	Days    int    `json:"days"`
	Source  string `json:"source"` // daily | manual
	Error   string `json:"error,omitempty"`
	// Truncated 为 true 表示命中单次上限，还有剩余没删，下次运行继续。
	Truncated bool `json:"truncated,omitempty"`
}

// NormalizeCleanupDays 把天数限制在合法范围内；非法值（<1）回落为缺省值，过大则取上限。
func NormalizeCleanupDays(d int) int {
	if d < MinCleanupDays {
		return DefaultCleanupDays
	}
	if d > MaxCleanupDays {
		return MaxCleanupDays
	}
	return d
}

// CleanupEnabled 返回自动清理是否开启（缺省关闭：删除数据必须由管理员主动打开）。
func (e *Env) CleanupEnabled() bool { return e.Set.Bool(SettingCleanupEnable, false) }

// CleanupDays 返回保留天数，保证永远是合法值。
func (e *Env) CleanupDays() int {
	return NormalizeCleanupDays(e.Set.Int(SettingCleanupDays, DefaultCleanupDays))
}

// cleanupCutoff 返回截止时间戳：取消时间早于它的已取消订单才会被删除。
func cleanupCutoff(now time.Time, days int) int64 {
	return now.Add(-time.Duration(days) * 24 * time.Hour).Unix()
}

// CountCancelledOlderThan 统计当前满足清理条件的订单数（「待清理」预览，不删除任何数据）。
func (e *Env) CountCancelledOlderThan(days int) (int64, error) {
	var n int64
	err := e.DB.Raw("SELECT COUNT(*) FROM v2_order WHERE status = ? AND updated_at < ?",
		orderStatusCancelled, cleanupCutoff(e.Now(), NormalizeCleanupDays(days))).Scan(&n).Error
	return n, err
}

// CleanupCancelledOrders 分批删除超过 days 天的已取消订单，并记录结果。
// 状态条件写在每一条 DELETE 里，所以即使清理期间订单被改成别的状态（例如被管理员改回），也不会被误删。
func (e *Env) CleanupCancelledOrders(days int, source string) (CleanupResult, error) {
	days = NormalizeCleanupDays(days)
	res := CleanupResult{At: e.Now().Unix(), Days: days, Source: source}
	cutoff := cleanupCutoff(e.Now(), days)
	for res.Deleted < cleanupMaxTotal {
		r := e.DB.Exec("DELETE FROM v2_order WHERE status = ? AND updated_at < ? ORDER BY id LIMIT ?",
			orderStatusCancelled, cutoff, cleanupBatch)
		if r.Error != nil {
			res.Error = r.Error.Error()
			e.saveCleanupResult(res)
			return res, r.Error
		}
		res.Deleted += r.RowsAffected
		if r.RowsAffected < cleanupBatch {
			break
		}
	}
	if res.Deleted >= cleanupMaxTotal {
		res.Truncated = true
	}
	e.saveCleanupResult(res)
	return res, nil
}

// ScheduledCleanupCancelled 是调度器入口：开关关闭时直接返回，不动任何数据。
func (e *Env) ScheduledCleanupCancelled() error {
	if !e.CleanupEnabled() {
		return nil
	}
	res, err := e.CleanupCancelledOrders(e.CleanupDays(), "daily")
	if err != nil {
		return err
	}
	log.Printf("cleanup:cancelled-orders 删除 %d 笔已取消超过 %d 天的订单", res.Deleted, res.Days)
	return nil
}

// LastCleanupResult 读取上一次清理的结果；从未清理过返回 false。
func (e *Env) LastCleanupResult() (CleanupResult, bool) {
	raw := e.Set.Get(settingCleanupLast, "")
	if raw == "" {
		return CleanupResult{}, false
	}
	var r CleanupResult
	if json.Unmarshal([]byte(raw), &r) != nil {
		return CleanupResult{}, false
	}
	return r, true
}

// saveCleanupResult 把结果写进 v2_settings，重启与清空 Redis 都不会丢。写失败只记日志，不影响清理本身。
func (e *Env) saveCleanupResult(r CleanupResult) {
	b, _ := json.Marshal(r)
	err := e.DB.Exec(
		"INSERT INTO v2_settings (name, value, created_at, updated_at) VALUES (?, ?, NOW(), NOW()) "+
			"ON DUPLICATE KEY UPDATE value = VALUES(value), updated_at = NOW()",
		settingCleanupLast, string(b)).Error
	if err != nil {
		log.Printf("cleanup:cancelled-orders 记录结果失败: %v", err)
		return
	}
	e.Set.Invalidate()
}

// DescribeCleanup 供日志与测试使用。
func (r CleanupResult) String() string {
	return fmt.Sprintf("deleted=%d days=%d source=%s", r.Deleted, r.Days, r.Source)
}
