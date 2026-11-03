package api

import (
	"github.com/gin-gonic/gin"

	"xboard/internal/jobs"
	"xboard/internal/resp"
)

// 已取消订单自动清理：管理端接口。规则与调度见 jobs/order_cleanup.go。

// orderCleanupStatus 返回当前设置、每日执行时刻、满足清理条件的订单数（预览）和上次清理结果。
func (a *App) orderCleanupStatus(c *gin.Context) {
	env := a.jobsEnv()
	days := env.CleanupDays()
	pending, err := env.CountCancelledOlderThan(days)
	if err != nil {
		resp.Fail(c, 500, "统计失败")
		return
	}
	out := gin.H{
		"enable":  env.CleanupEnabled(),
		"days":    days,
		"auto_at": jobs.CleanupAt,
		"pending": pending,
		"min":     jobs.MinCleanupDays,
		"max":     jobs.MaxCleanupDays,
		"last":    nil,
	}
	if last, ok := env.LastCleanupResult(); ok {
		out["last"] = last
	}
	resp.OK(c, out)
}

// cleanupDaysFrom 读取并校验请求里的天数；缺省时返回 def。天数非法返回 ok=false，不静默改成别的值。
func cleanupDaysFrom(m map[string]any, def int) (int, bool) {
	if _, present := m["days"]; !present {
		return def, true
	}
	d, ok := mInt(m, "days")
	if !ok || d < jobs.MinCleanupDays || d > jobs.MaxCleanupDays {
		return 0, false
	}
	return int(d), true
}

// orderCleanupSave 保存开关与保留天数。
func (a *App) orderCleanupSave(c *gin.Context) {
	m := bodyMap(c)
	env := a.jobsEnv()
	days, ok := cleanupDaysFrom(m, env.CleanupDays())
	if !ok {
		resp.Fail(c, 422, "保留天数必须是 1 到 3650 之间的整数")
		return
	}
	enable, has := mBool(m, "enable")
	if !has {
		enable = env.CleanupEnabled()
	}
	if err := a.settingUpsert(jobs.SettingCleanupEnable, enable); err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	if err := a.settingUpsert(jobs.SettingCleanupDays, days); err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	a.Set.Invalidate()
	resp.OK(c, gin.H{"enable": enable, "days": days})
}

// orderCleanupRun 立即执行一次清理（不论自动清理开关是否开启）。
// body.days 缺省时用已保存的天数；用于管理员在页面上确认后手动清理。
func (a *App) orderCleanupRun(c *gin.Context) {
	m := bodyMap(c)
	env := a.jobsEnv()
	days, ok := cleanupDaysFrom(m, env.CleanupDays())
	if !ok {
		resp.Fail(c, 422, "保留天数必须是 1 到 3650 之间的整数")
		return
	}
	res, err := env.CleanupCancelledOrders(days, "manual")
	if err != nil {
		resp.Fail(c, 500, "清理失败："+err.Error())
		return
	}
	resp.OK(c, res)
}
