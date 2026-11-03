package shop

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xboard/internal/model"
)

func lockForUpdate() clause.Locking { return clause.Locking{Strength: "UPDATE"} }

// 佣金类型（v2_user.commission_type）。
const (
	CommissionSystem  = 0
	CommissionPeriod  = 1
	CommissionOnetime = 2
)

// GbToTransfer 对应 Helper::gbToTransfer。
func GbToTransfer(gb int64) int64 {
	if gb <= 0 {
		return 0
	}
	maxGb := int64(math.MaxInt64 / 1073741824)
	if gb > maxGb {
		return math.MaxInt64
	}
	return gb * 1073741824
}

// TransferToGB 对应 Helper::transferToGB。
func TransferToGB(b int64) float64 { return float64(b) / 1073741824 }

// TradeNo 对应 Helper::generateOrderNo：时间戳 + 微秒 + 5 位随机数。
func TradeNo() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(90000))
	now := time.Now()
	return fmt.Sprintf("%s%06d%05d", now.Format("20060102150405"), now.Nanosecond()/1000, n.Int64()+10000)
}

// CreateOrder 对应 OrderService::createFromRequest。调用方需已校验周期与套餐。
func (s *Service) CreateOrder(u *model.User, p *model.Plan, period string, couponCode string) (*model.Order, error) {
	if err := s.ValidatePurchase(u, p, period); err != nil {
		return nil, err
	}
	key := PeriodKey(period)
	price, ok := PlanPrices(p)[key]
	if !ok {
		return nil, errors.New("This payment period cannot be purchased, please choose another period")
	}
	o := &model.Order{
		UserID: u.ID, PlanID: p.ID, Period: key, TradeNo: TradeNo(),
		TotalAmount: int64(price * 100),
	}
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if couponCode != "" {
			c, err := s.UseCoupon(tx, o, couponCode)
			if err != nil {
				return err
			}
			o.CouponID = &c.ID
		}
		s.setVipDiscount(u, o)
		if err := s.setOrderType(u, p, o); err != nil {
			return err
		}
		if err := s.setInvite(tx, u, o); err != nil {
			return err
		}
		if u.Balance != 0 && o.TotalAmount > 0 {
			if err := s.handleUserBalance(tx, u, o); err != nil {
				return err
			}
		}
		o.CreatedAt = time.Now().Unix()
		o.UpdatedAt = o.CreatedAt
		return tx.Create(o).Error
	})
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *Service) setVipDiscount(u *model.User, o *model.Order) {
	if u.Discount != nil && *u.Discount != 0 {
		o.DiscountAmount += int64(math.Round(float64(o.TotalAmount) * float64(*u.Discount) / 100))
	}
	o.TotalAmount -= o.DiscountAmount
}

// setOrderType 对应 OrderService::setOrderType，并计算升级时的剩余价值。
func (s *Service) setOrderType(u *model.User, p *model.Plan, o *model.Order) error {
	now := time.Now().Unix()
	if o.Period == PeriodReset {
		o.Type = TypeResetTraffic
		return nil
	}
	notExpired := u.ExpiredAt == nil || *u.ExpiredAt > now
	if u.PlanID != nil && o.PlanID != *u.PlanID && notExpired {
		if !s.Set.Bool("plan_change_enable", true) {
			return errors.New("目前不允许更改订阅，请联系客服或提交工单操作")
		}
		o.Type = TypeUpgrade
		// 有到期时间的周期套餐：剩余时间在开通时按价值折算成新套餐的时长叠加（见 buyByPeriod），
		// 不再折抵金额，否则剩余时间会被丢弃、只换回一笔折扣。
		// 一次性套餐没有到期时间，没有时长可叠加，仍按剩余流量价值折抵金额。
		if u.ExpiredAt == nil && s.Set.Bool("surplus_enable", true) {
			s.surplusValue(u, o)
			if o.SurplusAmount >= o.TotalAmount {
				o.SurplusCredit = o.SurplusAmount - o.TotalAmount
				o.TotalAmount = 0
			} else {
				o.TotalAmount -= o.SurplusAmount
			}
		}
		return nil
	}
	if notExpired && u.PlanID != nil && o.PlanID == *u.PlanID {
		o.Type = TypeRenewal
		return nil
	}
	o.Type = TypeNewPurchase
	return nil
}

// setInvite 对应 OrderService::setInvite：计算邀请人佣金与推荐时长奖励。
func (s *Service) setInvite(tx *gorm.DB, u *model.User, o *model.Order) error {
	if u.InviteUserID != nil && *u.InviteUserID != 0 && o.TotalAmount <= 0 {
		return nil
	}
	o.InviteUserID = u.InviteUserID
	if u.InviteUserID == nil {
		return nil
	}
	var inviter model.User
	if err := tx.First(&inviter, *u.InviteUserID).Error; err != nil {
		return nil
	}
	ct := int(inviter.CommissionType)
	if ct == CommissionSystem {
		if s.Set.Bool("commission_first_time_enable", true) {
			ct = CommissionOnetime
		} else {
			ct = CommissionPeriod
		}
	}
	isCommission := false
	switch ct {
	case CommissionPeriod:
		isCommission = true
	case CommissionOnetime:
		var n int64
		tx.Model(&model.Order{}).Where("user_id = ? AND status NOT IN (0, 2)", u.ID).Count(&n)
		isCommission = n == 0
	}
	if !isCommission {
		return nil
	}
	rate := float64(s.Set.Int("invite_commission", 10))
	if inviter.CommissionRate != nil && *inviter.CommissionRate != 0 {
		rate = float64(*inviter.CommissionRate)
	}
	o.CommissionBalance = int64(math.Round(float64(o.TotalAmount) * rate / 100))

	months := periodMonths[PeriodKey(o.Period)]
	now := time.Now()
	var secs int64
	switch {
	case months > 0:
		secs = now.AddDate(0, months, 0).Unix() - now.Unix()
	case PeriodKey(o.Period) == PeriodWeekly:
		secs = now.AddDate(0, 0, 7).Unix() - now.Unix()
	}
	if secs > 0 {
		pct := float64(s.Set.Int("referral_duration_percent", 20))
		v := int64(math.Round(float64(secs) * pct / 100))
		o.ReferralDurationSeconds = &v
	}
	return nil
}

func (s *Service) handleUserBalance(tx *gorm.DB, u *model.User, o *model.Order) error {
	remaining := u.Balance - o.TotalAmount
	if remaining >= 0 {
		if err := s.AddBalance(tx, u.ID, -o.TotalAmount); err != nil {
			return err
		}
		o.BalanceAmount = o.TotalAmount
		o.TotalAmount = 0
		return nil
	}
	if err := s.AddBalance(tx, u.ID, -u.Balance); err != nil {
		return err
	}
	o.BalanceAmount = u.Balance
	o.TotalAmount -= u.Balance
	return nil
}

// surplusValue 对应 OrderService::getSurplusValue，计算升级时可折抵的剩余金额。
func (s *Service) surplusValue(u *model.User, o *model.Order) {
	var orders []model.Order
	if u.ExpiredAt == nil {
		var last model.Order
		err := s.DB.Where("user_id = ? AND period = ? AND status = ?", u.ID, PeriodOnetime, StatusCompleted).
			Order("id DESC").First(&last).Error
		if err != nil {
			return
		}
		nowGB := TransferToGB(u.TransferEnab)
		if nowGB == 0 {
			return
		}
		paid := last.TotalAmount + last.BalanceAmount
		if paid == 0 {
			return
		}
		unit := float64(paid) / nowGB
		notUsed := nowGB - TransferToGB(u.U+u.D)
		res := unit * notUsed
		if res < 0 {
			res = 0
		}
		o.SurplusAmount = int64(res)
		s.DB.Where("user_id = ? AND period != ? AND status = ?", u.ID, PeriodReset, StatusCompleted).Find(&orders)
		o.SurplusOrderIDs = idsJSON(orders)
		return
	}
	s.DB.Where("user_id = ? AND period NOT IN ? AND status = ?", u.ID, []string{PeriodReset, PeriodOnetime}, StatusCompleted).
		Find(&orders)
	if len(orders) == 0 {
		o.SurplusAmount = 0
		o.SurplusOrderIDs = json.RawMessage("[]")
		return
	}
	var amountSum int64
	var daySum int64
	first := orders[0].CreatedAt
	for _, it := range orders {
		amountSum += it.TotalAmount + it.BalanceAmount + it.SurplusAmount - it.SurplusCredit
		if d := periodDays[PeriodKey(it.Period)]; d > 0 {
			daySum += int64(d)
		}
		if it.CreatedAt < first {
			first = it.CreatedAt
		}
	}
	expiredAt := first + daySum*86400
	total := expiredAt - first
	now := time.Now().Unix()
	remain := expiredAt - now
	if remain < 0 {
		remain = 0
	}
	cycleRatio := 0.0
	if total > 0 {
		cycleRatio = float64(remain) / float64(total)
	}
	var plan model.Plan
	totalTraffic := 0.0
	if u.PlanID != nil && s.DB.First(&plan, *u.PlanID).Error == nil && plan.TransferEnable != nil {
		totalTraffic = float64(*plan.TransferEnable) * (float64(daySum) / 30)
	}
	used := TransferToGB(u.U + u.D)
	remainTraffic := math.Max(0, totalTraffic-used)
	trafficRatio := 0.0
	if totalTraffic > 0 {
		trafficRatio = remainTraffic / totalTraffic
	}
	ratio := cycleRatio
	if s.Set.Int("change_order_event_id", 0) == 1 {
		ratio = math.Min(cycleRatio, trafficRatio)
	}
	o.SurplusAmount = int64(math.Max(0, float64(amountSum)*ratio))
	o.SurplusOrderIDs = idsJSON(orders)
}

func derefInt(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func idsJSON(orders []model.Order) json.RawMessage {
	ids := make([]int64, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.ID)
	}
	b, _ := json.Marshal(ids)
	return b
}

// Open 对应 OrderService::open：开通套餐（新购/续费/升级/一次性/重置包）。
func (s *Service) Open(o *model.Order) error {
	var plan model.Plan
	if err := s.DB.First(&plan, o.PlanID).Error; err != nil {
		return err
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var u model.User
		if err := tx.Clauses(lockForUpdate()).First(&u, o.UserID).Error; err != nil {
			return err
		}
		if o.SurplusCredit != 0 {
			u.Balance += o.SurplusCredit
		}
		if len(o.SurplusOrderIDs) > 0 {
			var ids []int64
			_ = json.Unmarshal(o.SurplusOrderIDs, &ids)
			if len(ids) > 0 {
				tx.Model(&model.Order{}).Where("id IN ?", ids).Update("status", StatusDiscounted)
			}
		}
		switch o.Period {
		case PeriodOnetime:
			s.buyOneTime(tx, &u, &plan)
		case PeriodReset:
			if err := s.PerformReset(tx, &u, SourceOrder); err != nil {
				return err
			}
		default:
			if err := s.buyByPeriod(tx, &u, o, &plan); err != nil {
				return err
			}
		}
		u.SpeedLimit = plan.SpeedLimit
		u.DeviceLimit = plan.DeviceLimit
		if err := tx.Save(&u).Error; err != nil {
			return err
		}
		o.Status = StatusCompleted
		o.UpdatedAt = time.Now().Unix()
		return tx.Model(o).Updates(map[string]any{"status": o.Status, "updated_at": o.UpdatedAt}).Error
	})
}

func (s *Service) buyByPeriod(tx *gorm.DB, u *model.User, o *model.Order, p *model.Plan) error {
	if o.Type == TypeUpgrade {
		// 换套餐：旧套餐剩余的时间按价值折算成新套餐的时长，新周期从折算后的时间点往后加。
		now := time.Now().Unix()
		var carry int64
		if u.ExpiredAt != nil {
			var old *model.Plan
			if u.PlanID != nil {
				var op model.Plan
				if tx.First(&op, *u.PlanID).Error == nil {
					old = &op
				}
			}
			carry = ConvertRemaining(*u.ExpiredAt-now, old, p, PeriodKey(o.Period))
		}
		base := now + carry
		u.ExpiredAt = &base
	}
	u.TransferEnab = GbToTransfer(derefInt(p.TransferEnable))
	if u.ExpiredAt == nil || o.Type == TypeNewPurchase {
		if err := s.PerformReset(tx, u, SourceOrder); err != nil {
			return err
		}
	}
	u.PlanID = &p.ID
	u.GroupID = p.GroupID
	exp, err := periodExpiry(o.Period, u.ExpiredAt)
	if err != nil {
		return err
	}
	u.ExpiredAt = &exp
	return nil
}

func (s *Service) buyOneTime(tx *gorm.DB, u *model.User, p *model.Plan) {
	_ = s.PerformReset(tx, u, SourceOrder)
	u.TransferEnab = GbToTransfer(derefInt(p.TransferEnable))
	u.PlanID = &p.ID
	u.GroupID = p.GroupID
	u.ExpiredAt = nil
}

// PlanDailyPrice 返回套餐每天的价格（元）：优先用 preferKey 周期的价格，
// 该周期未定价时取各周期中最低的日单价；套餐没有任何周期定价时返回 0。
func PlanDailyPrice(p *model.Plan, preferKey string) float64 {
	prices := PlanPrices(p)
	if v := prices[preferKey]; v > 0 {
		if d := periodDays[preferKey]; d > 0 {
			return v / float64(d)
		}
	}
	best := 0.0
	for k, v := range prices {
		d := periodDays[k]
		if d <= 0 || v <= 0 {
			continue
		}
		if dp := v / float64(d); best == 0 || dp < best {
			best = dp
		}
	}
	return best
}

// ConvertRemaining 把旧套餐剩余的秒数按日单价折算成新套餐的秒数（价值不变）。
//   - 旧套餐已不存在：剩余时间原样保留；
//   - 旧套餐没有定价（如免费试用）：剩余时间没有价值，不折算，避免用试用时长换付费天数；
//   - 新套餐没有定价：无法换算，原样保留。
func ConvertRemaining(remain int64, oldPlan, newPlan *model.Plan, preferKey string) int64 {
	if remain <= 0 {
		return 0
	}
	if oldPlan == nil {
		return remain
	}
	od := PlanDailyPrice(oldPlan, preferKey)
	if od <= 0 {
		return 0
	}
	nd := PlanDailyPrice(newPlan, preferKey)
	if nd <= 0 {
		return remain
	}
	return int64(float64(remain) * od / nd)
}

// periodExpiry 对应 OrderService::getTime：从 max(now, 当前到期) 起按周期顺延。
func periodExpiry(period string, cur *int64) (int64, error) {
	now := time.Now()
	base := now
	if cur != nil && *cur > now.Unix() {
		base = time.Unix(*cur, 0)
	}
	base = base.In(Loc)
	key := PeriodKey(period)
	if m, ok := periodMonths[key]; ok {
		return base.AddDate(0, m, 0).Unix(), nil
	}
	if key == PeriodWeekly {
		return base.AddDate(0, 0, 7).Unix(), nil
	}
	return 0, errors.New("无效的套餐周期")
}

// Paid 对应 OrderService::paid：PENDING -> PROCESSING 原子切换后同步开通。
func (s *Service) Paid(tradeNo, callbackNo string) error {
	now := time.Now().Unix()
	res := s.DB.Model(&model.Order{}).
		Where("trade_no = ? AND status = ?", tradeNo, StatusPending).
		Updates(map[string]any{"status": StatusProcessing, "paid_at": now, "callback_no": callbackNo, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	var o model.Order
	if err := s.DB.Where("trade_no = ?", tradeNo).First(&o).Error; err != nil {
		return err
	}
	err := s.Open(&o)
	if PaidHook != nil {
		PaidHook(&o)
	}
	return err
}

// PaidHook 在订单付款成功后调用（对应 payment.notify.success 事件），由 API 层注入 Telegram 通知。
var PaidHook func(o *model.Order)

// Cancel 对应 OrderService::cancel：取消订单并退还余额抵扣部分。
func (s *Service) Cancel(o *model.Order) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(o).Updates(map[string]any{"status": StatusCancelled, "updated_at": time.Now().Unix()}).Error; err != nil {
			return err
		}
		if o.BalanceAmount != 0 {
			return s.AddBalance(tx, o.UserID, o.BalanceAmount)
		}
		return nil
	})
}
