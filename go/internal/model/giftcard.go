package model

import "encoding/json"

// 礼品卡三张表：模板、兑换码、使用记录。JSON 配置列保持原始形态，由业务层按需解析。

// GiftCardTemplate 对应 v2_gift_card_template。type: 1 通用 2 套餐 3 盲盒。
type GiftCardTemplate struct {
	ID              int64           `gorm:"primaryKey" json:"id"`
	Name            string          `json:"name"`
	Description     *string         `json:"description"`
	Type            int             `json:"type"`
	Status          bool            `json:"status"`
	Conditions      json.RawMessage `gorm:"type:json" json:"conditions"`
	Rewards         json.RawMessage `gorm:"type:json" json:"rewards"`
	Limits          json.RawMessage `gorm:"type:json" json:"limits"`
	SpecialConfig   json.RawMessage `gorm:"column:special_config;type:json" json:"special_config"`
	Icon            *string         `json:"icon"`
	BackgroundImage *string         `gorm:"column:background_image" json:"background_image"`
	ThemeColor      string          `gorm:"column:theme_color" json:"theme_color"`
	Sort            int             `json:"sort"`
	AdminID         int64           `gorm:"column:admin_id" json:"-"`
	CreatedAt       int64           `json:"created_at"`
	UpdatedAt       int64           `json:"updated_at"`
}

func (GiftCardTemplate) TableName() string { return "v2_gift_card_template" }

// GiftCardCode 对应 v2_gift_card_code。status: 0 未使用 1 已使用 2 已过期 3 已禁用。
type GiftCardCode struct {
	ID            int64           `gorm:"primaryKey" json:"id"`
	TemplateID    int64           `gorm:"column:template_id" json:"template_id"`
	Code          string          `json:"code"`
	BatchID       *string         `gorm:"column:batch_id" json:"batch_id"`
	Status        int             `json:"status"`
	UserID        *int64          `gorm:"column:user_id" json:"user_id"`
	UsedAt        *int64          `gorm:"column:used_at" json:"used_at"`
	ExpiresAt     *int64          `gorm:"column:expires_at" json:"expires_at"`
	ActualRewards json.RawMessage `gorm:"column:actual_rewards;type:json" json:"actual_rewards"`
	UsageCount    int64           `gorm:"column:usage_count" json:"usage_count"`
	MaxUsage      int64           `gorm:"column:max_usage" json:"max_usage"`
	Metadata      json.RawMessage `gorm:"type:json" json:"metadata"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
}

func (GiftCardCode) TableName() string { return "v2_gift_card_code" }

// GiftCardUsage 对应 v2_gift_card_usage（只有 created_at，没有 updated_at）。
type GiftCardUsage struct {
	ID                int64           `gorm:"primaryKey" json:"id"`
	CodeID            int64           `gorm:"column:code_id" json:"code_id"`
	TemplateID        int64           `gorm:"column:template_id" json:"template_id"`
	UserID            int64           `gorm:"column:user_id" json:"user_id"`
	InviteUserID      *int64          `gorm:"column:invite_user_id" json:"invite_user_id"`
	RewardsGiven      json.RawMessage `gorm:"column:rewards_given;type:json" json:"rewards_given"`
	InviteRewards     json.RawMessage `gorm:"column:invite_rewards;type:json" json:"invite_rewards"`
	UserLevelAtUse    *int64          `gorm:"column:user_level_at_use" json:"user_level_at_use"`
	PlanIDAtUse       *int64          `gorm:"column:plan_id_at_use" json:"plan_id_at_use"`
	MultiplierApplied float64         `gorm:"column:multiplier_applied" json:"multiplier_applied"`
	IPAddress         *string         `gorm:"column:ip_address" json:"-"`
	UserAgent         *string         `gorm:"column:user_agent" json:"-"`
	Notes             *string         `json:"notes"`
	CreatedAt         int64           `json:"created_at"`
}

func (GiftCardUsage) TableName() string { return "v2_gift_card_usage" }
