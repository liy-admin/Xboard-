package model

import (
	"encoding/json"
	"time"
)

// Plan 对应 v2_plan。prices 为 JSON：{"monthly": 9.9, "onetime": 99, ...}，单位为元。
type Plan struct {
	ID                 int64           `gorm:"primaryKey" json:"id"`
	GroupID            *int64          `json:"group_id"`
	TransferEnable     *int64          `json:"transfer_enable"`
	Name               string          `json:"name"`
	SpeedLimit         *int64          `json:"speed_limit"`
	DeviceLimit        *int64          `json:"device_limit"`
	Show               bool            `json:"show"`
	Sell               bool            `json:"sell"`
	Sort               *int            `json:"sort"`
	Renew              bool            `json:"renew"`
	Content            *string         `json:"content"`
	Prices             json.RawMessage `gorm:"type:json" json:"prices"`
	ResetTrafficMethod *int            `gorm:"column:reset_traffic_method" json:"reset_traffic_method"`
	CapacityLimit      *int64          `json:"capacity_limit"`
	Tags               json.RawMessage `gorm:"type:json" json:"tags"`
	CreatedAt          int64           `json:"created_at"`
	UpdatedAt          int64           `json:"updated_at"`
}

func (Plan) TableName() string { return "v2_plan" }

// Order 对应 v2_order。金额单位为分（与 PHP 保持一致）。
type Order struct {
	ID                      int64           `gorm:"primaryKey" json:"id"`
	InviteUserID            *int64          `json:"invite_user_id"`
	UserID                  int64           `json:"user_id"`
	PlanID                  int64           `json:"plan_id"`
	CouponID                *int64          `json:"coupon_id"`
	PaymentID               *int64          `json:"payment_id"`
	Type                    int             `json:"type"`
	Period                  string          `json:"period"`
	TradeNo                 string          `gorm:"column:trade_no" json:"trade_no"`
	TotalAmount             int64           `json:"total_amount"`
	HandlingAmount          *int64          `json:"handling_amount"`
	DiscountAmount          int64           `json:"discount_amount"`
	SurplusAmount           int64           `json:"surplus_amount"`
	SurplusOrderIDs         json.RawMessage `gorm:"column:surplus_order_ids;type:json" json:"surplus_order_ids"`
	SurplusCredit           int64           `json:"surplus_credit"`
	BalanceAmount           int64           `json:"balance_amount"`
	Status                  int             `json:"status"`
	CommissionStatus        int             `json:"commission_status"`
	CommissionBalance       int64           `json:"commission_balance"`
	CallbackNo              *string         `json:"callback_no"`
	PaidAt                  *int64          `json:"paid_at"`
	ReferralDurationSeconds *int64          `json:"referral_duration_seconds"`
	ActualCommissionBalance *int64          `gorm:"column:actual_commission_balance" json:"actual_commission_balance"`
	CreatedAt               int64           `json:"created_at"`
	UpdatedAt               int64           `json:"updated_at"`
}

func (Order) TableName() string { return "v2_order" }

// Payment 对应 v2_payment。
type Payment struct {
	ID                 int64    `gorm:"primaryKey" json:"id"`
	UUID               string   `json:"uuid"`
	Name               string   `json:"name"`
	Payment            string   `json:"payment"`
	Icon               *string  `json:"icon"`
	Config             string   `json:"-"`
	Enable             bool     `json:"enable"`
	HandlingFeeFixed   *int64   `json:"handling_fee_fixed"`
	HandlingFeePercent *float64 `json:"handling_fee_percent"`
	NotifyDomain       *string  `json:"notify_domain"`
	Sort               *int     `json:"sort"`
}

func (Payment) TableName() string { return "v2_payment" }

// Coupon 对应 v2_coupon。
type Coupon struct {
	ID               int64           `gorm:"primaryKey" json:"id"`
	Code             string          `json:"code"`
	Name             string          `json:"name"`
	Type             int             `json:"type"`
	Value            int64           `json:"value"`
	Show             bool            `json:"show"`
	LimitUse         *int64          `json:"limit_use"`
	LimitUseWithUser *int64          `json:"limit_use_with_user"`
	LimitPlanIDs     json.RawMessage `gorm:"column:limit_plan_ids;type:json" json:"limit_plan_ids"`
	LimitPeriod      json.RawMessage `gorm:"column:limit_period;type:json" json:"limit_period"`
	StartedAt        int64           `json:"started_at"`
	EndedAt          int64           `json:"ended_at"`
	CreatedAt        int64           `json:"created_at"`
	UpdatedAt        int64           `json:"updated_at"`
}

func (Coupon) TableName() string { return "v2_coupon" }

// TrafficResetLog 对应 v2_traffic_reset_logs。
type TrafficResetLog struct {
	ID            int64           `gorm:"primaryKey"`
	UserID        int64           `gorm:"column:user_id"`
	ResetType     string          `gorm:"column:reset_type"`
	ResetTime     time.Time       `gorm:"column:reset_time"`
	OldUpload     int64           `gorm:"column:old_upload"`
	OldDownload   int64           `gorm:"column:old_download"`
	OldTotal      int64           `gorm:"column:old_total"`
	NewUpload     int64           `gorm:"column:new_upload"`
	NewDownload   int64           `gorm:"column:new_download"`
	NewTotal      int64           `gorm:"column:new_total"`
	TriggerSource string          `gorm:"column:trigger_source"`
	Metadata      json.RawMessage `gorm:"type:json"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (TrafficResetLog) TableName() string { return "v2_traffic_reset_logs" }
