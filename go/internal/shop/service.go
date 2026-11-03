package shop

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xboard/internal/model"
	"xboard/internal/setting"
)

// Service 持有订单/套餐/优惠券逻辑所需的依赖。
type Service struct {
	DB  *gorm.DB
	Set *setting.Service
}

// 周期键（v2_plan.prices 的 key）。
const (
	PeriodWeekly      = "weekly"
	PeriodMonthly     = "monthly"
	PeriodQuarterly   = "quarterly"
	PeriodHalfYearly  = "half_yearly"
	PeriodYearly      = "yearly"
	PeriodTwoYearly   = "two_yearly"
	PeriodThreeYearly = "three_yearly"
	PeriodOnetime     = "onetime"
	PeriodReset       = "reset_traffic"
)

// 订单类型（v2_order.type）。
const (
	TypeNewPurchase  = 1
	TypeRenewal      = 2
	TypeUpgrade      = 3
	TypeResetTraffic = 4
)

// 订单状态（v2_order.status）。
const (
	StatusPending    = 0
	StatusProcessing = 1
	StatusCancelled  = 2
	StatusCompleted  = 3
	StatusDiscounted = 4
)

// periodMonths 对应 OrderService::STR_TO_TIME。
var periodMonths = map[string]int{
	PeriodMonthly: 1, PeriodQuarterly: 3, PeriodHalfYearly: 6, PeriodYearly: 12,
	PeriodTwoYearly: 24, PeriodThreeYearly: 36,
}

// periodDays 对应 Plan::getAvailablePeriods 中的 days。
var periodDays = map[string]int{
	PeriodWeekly: 7, PeriodMonthly: 30, PeriodQuarterly: 90, PeriodHalfYearly: 180,
	PeriodYearly: 365, PeriodTwoYearly: 730, PeriodThreeYearly: 1095,
	PeriodOnetime: -1, PeriodReset: -1,
}

// legacyPeriod 对应 Plan::LEGACY_PERIOD_MAPPING（旧字段名 -> 新周期键）。
var legacyPeriod = map[string]string{
	"week_price": PeriodWeekly, "month_price": PeriodMonthly, "quarter_price": PeriodQuarterly,
	"half_year_price": PeriodHalfYearly, "year_price": PeriodYearly, "two_year_price": PeriodTwoYearly,
	"three_year_price": PeriodThreeYearly, "onetime_price": PeriodOnetime, "reset_price": PeriodReset,
}

// PeriodKey 对应 PlanService::getPeriodKey：旧格式转新格式，新格式原样返回。
func PeriodKey(p string) string {
	if _, ok := periodDays[p]; ok {
		return p
	}
	if n, ok := legacyPeriod[p]; ok {
		return n
	}
	return p
}

// PlanPrices 解析 v2_plan.prices。
//
// prices 里未开售的周期是显式 null。PHP 的 `$prices[$key] ?? null` 把它当作「不可购买」，
// 所以这里必须丢弃 null，而不能解码成 0——否则下单校验会把它当成 0 元可购买。
func PlanPrices(p *model.Plan) map[string]float64 {
	raw := map[string]*float64{}
	_ = json.Unmarshal(p.Prices, &raw)
	out := make(map[string]float64, len(raw))
	for k, v := range raw {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}

// HasCapacity 对应 PlanService::hasCapacity。
func (s *Service) HasCapacity(p *model.Plan) bool {
	if p.CapacityLimit == nil {
		return true
	}
	var n int64
	s.DB.Model(&model.User{}).
		Where("plan_id = ? AND (expired_at >= ? OR expired_at IS NULL)", p.ID, time.Now().Unix()).
		Count(&n)
	return *p.CapacityLimit-n > 0
}

// UnlimitedTraffic 流量上限为 0 视为无限流量。
func UnlimitedTraffic(u *model.User) bool { return u.TransferEnab == 0 }

// IsAvailable 对应 UserService::isAvailable：未封禁且未过期。
//
// 与 PHP 的差异：PHP 要求 transfer_enable 非零，流量为 0 的用户一律不可用（订阅返回 403）；
// 这里按约定把 0 视为无限流量，不再因此拒绝。
func IsAvailable(u *model.User) bool {
	return !u.Banned && (u.ExpiredAt == nil || *u.ExpiredAt > time.Now().Unix())
}

// HasPlan 用户当前是否持有可用套餐（有套餐且未封禁、未过期）。
// 流量为 0（无限）的账号不等于有会员：没有套餐的用户即使 IsAvailable 也不算会员。
func HasPlan(u *model.User) bool { return u.PlanID != nil && IsAvailable(u) }

// ValidatePurchase 对应 PlanService::validatePurchase。
func (s *Service) ValidatePurchase(u *model.User, p *model.Plan, period string) error {
	key := PeriodKey(period)
	price, ok := PlanPrices(p)[key]
	if !ok {
		return errors.New("This payment period cannot be purchased, please choose another period")
	}
	_ = price
	if key == PeriodReset {
		if !IsAvailable(u) || u.PlanID == nil || *u.PlanID != p.ID {
			return errors.New("Subscription has expired or no active subscription, unable to purchase Data Reset Package")
		}
		return nil
	}
	if (u.PlanID == nil || *u.PlanID != p.ID) && !s.HasCapacity(p) {
		return errors.New("Current product is sold out")
	}
	show, renew := p.Show, p.Renew
	samePlan := u.PlanID != nil && *u.PlanID == p.ID
	if !show && (!renew || !samePlan) {
		return errors.New("This subscription has been sold out, please choose another subscription")
	}
	if !renew && samePlan {
		return errors.New("This subscription cannot be renewed, please change to another subscription")
	}
	if !show && renew && !IsAvailable(u) {
		return errors.New("This subscription has expired, please change to another subscription")
	}
	return nil
}

// AddBalance 对应 UserService::addBalance（行锁 + 余额不得为负）。
func (s *Service) AddBalance(tx *gorm.DB, userID, delta int64) error {
	var u model.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, userID).Error; err != nil {
		return err
	}
	nb := u.Balance + delta
	if nb < 0 {
		return errors.New("Insufficient balance")
	}
	return tx.Model(&model.User{}).Where("id = ?", userID).Update("balance", nb).Error
}

// ---- 优惠券（CouponService） ----

// UseCoupon 校验并扣减优惠券，写入订单折扣金额。
func (s *Service) UseCoupon(tx *gorm.DB, o *model.Order, code string) (*model.Coupon, error) {
	var c model.Coupon
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code = ?", code).First(&c).Error; err != nil {
		return nil, errors.New("Invalid coupon")
	}
	if err := s.checkCoupon(tx, &c, o); err != nil {
		return nil, err
	}
	switch c.Type {
	case 1:
		o.DiscountAmount = c.Value
	case 2:
		o.DiscountAmount = int64(float64(o.TotalAmount) * (float64(c.Value) / 100))
	}
	if o.DiscountAmount > o.TotalAmount {
		o.DiscountAmount = o.TotalAmount
	}
	if c.LimitUse != nil {
		if *c.LimitUse <= 0 {
			return nil, errors.New("Coupon failed")
		}
		if err := tx.Model(&model.Coupon{}).Where("id = ?", c.ID).Update("limit_use", *c.LimitUse-1).Error; err != nil {
			return nil, err
		}
	}
	return &c, nil
}

func (s *Service) checkCoupon(tx *gorm.DB, c *model.Coupon, o *model.Order) error {
	if !c.Show {
		return errors.New("Invalid coupon")
	}
	if c.LimitUse != nil && *c.LimitUse <= 0 {
		return errors.New("This coupon is no longer available")
	}
	now := time.Now().Unix()
	if now < c.StartedAt {
		return errors.New("This coupon has not yet started")
	}
	if now > c.EndedAt {
		return errors.New("This coupon has expired")
	}
	var planIDs []int64
	_ = json.Unmarshal(c.LimitPlanIDs, &planIDs)
	if len(planIDs) > 0 && o.PlanID != 0 && !containsInt(planIDs, o.PlanID) {
		return errors.New("The coupon code cannot be used for this subscription")
	}
	var periods []string
	_ = json.Unmarshal(c.LimitPeriod, &periods)
	for i := range periods {
		periods[i] = PeriodKey(periods[i]) // 与 Coupon::getLimitPeriodAttribute 一致：读取时统一为新版键
	}
	if len(periods) > 0 && o.Period != "" && !containsStr(periods, PeriodKey(o.Period)) {
		return errors.New("The coupon code cannot be used for this period")
	}
	if c.LimitUseWithUser != nil {
		var used int64
		tx.Model(&model.Order{}).
			Where("coupon_id = ? AND user_id = ? AND status NOT IN (0, 2)", c.ID, o.UserID).Count(&used)
		if used >= *c.LimitUseWithUser {
			return errors.New("The coupon can only be used per person limit")
		}
	}
	return nil
}

func containsInt(xs []int64, v int64) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func containsStr(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// ApplyInvite 对应 OrderService::setInvite：为手动分配的订单记录邀请人与佣金基数（在事务内调用）。
func (s *Service) ApplyInvite(tx *gorm.DB, u *model.User, o *model.Order) error {
	return s.setInvite(tx, u, o)
}
