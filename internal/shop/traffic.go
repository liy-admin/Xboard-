// Package shop 对应 PHP 的 OrderService / PlanService / CouponService / TrafficResetService。
package shop

import (
	"time"
	_ "time/tzdata" // 保证 Asia/Shanghai 在 Windows 等无系统时区库的环境可用

	"gorm.io/gorm"

	"xboard/internal/model"
)

// 流量重置方式（v2_plan.reset_traffic_method）。nil 表示跟随系统设置。
const (
	ResetFirstDayMonth = 0
	ResetMonthly       = 1
	ResetNever         = 2
	ResetFirstDayYear  = 3
	ResetYearly        = 4
)

// 重置类型与来源（v2_traffic_reset_logs）。
const (
	TypeMonthly      = "monthly"
	TypeFirstDayMon  = "first_day_month"
	TypeYearly       = "yearly"
	TypeFirstDayYear = "first_day_year"
	TypeManual       = "manual"

	SourceAuto       = "auto"
	SourceManual     = "manual"
	SourceCron       = "cron"
	SourceUserAccess = "user_access"
	SourceOrder      = "order"
)

// Loc 为应用时区（config/app.php: Asia/Shanghai）。
var Loc = mustLoad("Asia/Shanghai")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// SystemResetMethod 读取后台 reset_traffic_method，默认按月。
func (s *Service) systemResetMethod() int {
	return s.Set.Int("reset_traffic_method", ResetMonthly)
}

// effectiveResetMethod 返回实际生效的重置方式；-1 表示不重置。
func (s *Service) effectiveResetMethod(plan *model.Plan) int {
	if plan == nil {
		return -1
	}
	if plan.ResetTrafficMethod == nil {
		m := s.systemResetMethod()
		if m == ResetNever {
			return -1
		}
		return m
	}
	if *plan.ResetTrafficMethod == ResetNever {
		return -1
	}
	return *plan.ResetTrafficMethod
}

// CalcNextReset 对应 TrafficResetService::calculateNextResetTime。
func (s *Service) CalcNextReset(u *model.User, plan *model.Plan, now time.Time) *time.Time {
	if u.ExpiredAt == nil {
		return nil
	}
	m := s.effectiveResetMethod(plan)
	if m < 0 {
		return nil
	}
	now = now.In(Loc)
	var t time.Time
	switch m {
	case ResetFirstDayMonth:
		t = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, Loc)
	case ResetMonthly:
		t = nextMonthly(time.Unix(*u.ExpiredAt, 0).In(Loc), now)
	case ResetFirstDayYear:
		t = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, Loc)
	case ResetYearly:
		t = nextYearly(time.Unix(*u.ExpiredAt, 0).In(Loc), now)
	default:
		return nil
	}
	return &t
}

// nextMonthly 对应 getNextMonthlyReset：以到期日的"日"作为每月重置日，溢出时取当月最后一天。
func nextMonthly(expired, now time.Time) time.Time {
	h, mi, sec := expired.Clock()
	day := expired.Day()
	cur := time.Date(now.Year(), now.Month(), day, h, mi, sec, 0, Loc)
	if cur.Unix() > now.Unix() {
		return cur
	}
	next := time.Date(now.Year(), now.Month()+1, day, h, mi, sec, 0, Loc)
	if next.Month() != (now.Month()%12)+1 {
		y, m := now.Year(), now.Month()+1
		if now.Month() == 12 {
			y++
			m = 1
		}
		last := time.Date(y, m+1, 0, 0, 0, 0, 0, Loc).Day()
		d := day
		if last < d {
			d = last
		}
		next = time.Date(y, m, d, h, mi, sec, 0, Loc)
	}
	return next
}

// nextYearly 对应 getNextYearlyReset。
func nextYearly(expired, now time.Time) time.Time {
	h, mi, sec := expired.Clock()
	month, day := expired.Month(), expired.Day()
	cur := time.Date(now.Year(), month, day, h, mi, sec, 0, Loc)
	if cur.Unix() > now.Unix() {
		return cur
	}
	next := time.Date(now.Year()+1, month, day, h, mi, sec, 0, Loc)
	if next.Month() != month {
		last := time.Date(now.Year()+1, month+1, 0, 0, 0, 0, 0, Loc).Day()
		d := day
		if last < d {
			d = last
		}
		next = time.Date(now.Year()+1, month, d, h, mi, sec, 0, Loc)
	}
	return next
}

// PerformReset 对应 TrafficResetService::performReset：清零上下行、写重置日志、刷新下次重置时间。
// 在传入的事务内执行，调用方负责提交。
func (s *Service) PerformReset(tx *gorm.DB, u *model.User, source string) error {
	var plan *model.Plan
	if u.PlanID != nil {
		var p model.Plan
		if tx.First(&p, *u.PlanID).Error == nil {
			plan = &p
		}
	}
	oldUp, oldDown := u.U, u.D
	now := time.Now()
	next := s.CalcNextReset(u, plan, now)
	var nextTs any
	if next != nil {
		nextTs = next.Unix()
	}
	resetCount := u.ResetCount + 1
	if err := tx.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]any{
		"u": 0, "d": 0, "last_reset_at": now.Unix(), "reset_count": resetCount, "next_reset_at": nextTs,
	}).Error; err != nil {
		return err
	}
	u.U, u.D, u.ResetCount = 0, 0, resetCount
	ts := now.Unix()
	u.LastResetAt = &ts
	if next != nil {
		n := next.Unix()
		u.NextResetAt = &n
	} else {
		u.NextResetAt = nil
	}
	return tx.Create(&model.TrafficResetLog{
		UserID: u.ID, ResetType: resetTypeFromPlan(s, plan), ResetTime: now,
		OldUpload: oldUp, OldDownload: oldDown, OldTotal: oldUp + oldDown,
		TriggerSource: source,
	}).Error
}

func resetTypeFromPlan(s *Service, plan *model.Plan) string {
	if plan == nil {
		return TypeManual
	}
	m := s.effectiveResetMethod(plan)
	switch m {
	case ResetFirstDayMonth:
		return TypeFirstDayMon
	case ResetMonthly:
		return TypeMonthly
	case ResetFirstDayYear:
		return TypeFirstDayYear
	case ResetYearly:
		return TypeYearly
	}
	return TypeManual
}
