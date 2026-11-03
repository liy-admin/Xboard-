package shop

import (
	"errors"

	"gorm.io/gorm"

	"xboard/internal/model"
)

// CheckCoupon 对应 CouponService::check()，只校验不扣减（/user/coupon/check 使用）。
// planID/period 为 0/"" 时跳过套餐与周期限制，与 PHP 的 `&& $this->planId` 语义一致。
func (s *Service) CheckCoupon(code string, planID, userID int64, period string) (*model.Coupon, error) {
	var c model.Coupon
	if err := s.DB.Where("code = ?", code).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("Invalid coupon")
		}
		return nil, err
	}
	o := &model.Order{PlanID: planID, UserID: userID, Period: period}
	if err := s.checkCoupon(s.DB, &c, o); err != nil {
		return nil, err
	}
	return &c, nil
}

// LegacyPeriodName 对应 PlanService::convertToLegacyPeriod：新周期键转旧版字段名，未知键原样返回。
func LegacyPeriodName(period string) string {
	for legacy, key := range legacyPeriod {
		if key == period {
			return legacy
		}
	}
	return period
}

// LegacyPeriodMapping 返回旧版字段名到新周期键的映射副本（Plan::LEGACY_PERIOD_MAPPING）。
func LegacyPeriodMapping() map[string]string {
	out := make(map[string]string, len(legacyPeriod))
	for k, v := range legacyPeriod {
		out[k] = v
	}
	return out
}
