// Package model 映射现有 Laravel 迁移产生的表（同库原地替换），仅声明已用字段。
package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID             int64   `gorm:"primaryKey" json:"id"`
	InviteUserID   *int64  `json:"invite_user_id"`
	ReferralCode   *string `gorm:"column:referral_code" json:"-"`
	TelegramID     *int64  `json:"telegram_id"`
	Email          string  `json:"email"`
	Password       string  `json:"-"`
	PasswordAlgo   *string `json:"-"`
	PasswordSalt   *string `json:"-"`
	Balance        int64   `json:"balance"`
	CommissionBal  int64   `gorm:"column:commission_balance" json:"commission_balance"`
	U              int64   `json:"u"`
	D              int64   `json:"d"`
	TransferEnab   int64   `gorm:"column:transfer_enable" json:"transfer_enable"`
	Banned         bool    `json:"banned"`
	IsAdmin        bool    `json:"is_admin"`
	IsStaff        bool    `json:"is_staff"`
	LastLoginAt    *int64  `json:"last_login_at"`
	UUID           string  `gorm:"column:uuid" json:"uuid"`
	GroupID        *int64  `json:"group_id"`
	PlanID         *int64  `json:"plan_id"`
	SpeedLimit     *int64  `json:"speed_limit"`
	DeviceLimit    *int64  `json:"device_limit"`
	Token          string  `json:"token"`
	ExpiredAt      *int64  `json:"expired_at"`
	EncKey         *string `json:"-"`
	DeviceID       *string `json:"-"`
	CreatedAt      int64   `json:"created_at"`
	UpdatedAt      int64   `json:"updated_at"`
	Discount       *int64  `json:"discount"`
	CommissionType int64   `gorm:"column:commission_type" json:"commission_type"`
	CommissionRate *int64  `gorm:"column:commission_rate" json:"commission_rate"`
	RemindExpire   *bool   `gorm:"column:remind_expire" json:"remind_expire"`
	RemindTraffic  *bool   `gorm:"column:remind_traffic" json:"remind_traffic"`
	NextResetAt    *int64  `json:"next_reset_at"`
	LastResetAt    *int64  `json:"last_reset_at"`
	ResetCount     int64   `json:"reset_count"`
}

func (User) TableName() string { return "v2_user" }

type Setting struct {
	ID    int64
	Name  string
	Value *string
}

func (Setting) TableName() string { return "v2_settings" }

type Server struct {
	ID               int64           `gorm:"primaryKey" json:"id"`
	Type             string          `json:"type"`
	Code             *string         `json:"code"`
	ParentID         *int64          `json:"parent_id"`
	GroupIDs         json.RawMessage `gorm:"column:group_ids;type:json" json:"group_ids"`
	RouteIDs         json.RawMessage `gorm:"column:route_ids;type:json" json:"route_ids"`
	Name             string          `json:"name"`
	Rate             float64         `json:"rate"`
	Tags             json.RawMessage `gorm:"type:json" json:"tags"`
	Host             string          `json:"host"`
	Port             string          `json:"port"`
	ServerPort       int             `json:"server_port"`
	ProtocolSettings json.RawMessage `gorm:"column:protocol_settings;type:json" json:"protocol_settings"`
	CustomOutbounds  json.RawMessage `gorm:"column:custom_outbounds;type:json" json:"custom_outbounds"`
	CustomRoutes     json.RawMessage `gorm:"column:custom_routes;type:json" json:"custom_routes"`
	CertConfig       json.RawMessage `gorm:"column:cert_config;type:json" json:"cert_config"`
	Show             bool            `json:"show"`
	Enabled          *bool           `gorm:"column:enabled" json:"enabled"`
	MachineID        *int64          `gorm:"column:machine_id" json:"machine_id"`
	Sort             *int            `json:"sort"`
	CreatedAt        time.Time       `json:"created_at"`
	TransferEnable   *int64          `gorm:"column:transfer_enable" json:"transfer_enable"`
	U                int64           `json:"u"`
	D                int64           `json:"d"`
	RateTimeEnable   bool            `gorm:"column:rate_time_enable" json:"rate_time_enable"`
	RateTimeRanges   json.RawMessage `gorm:"column:rate_time_ranges;type:json" json:"rate_time_ranges"`
	CountryCode      *string         `gorm:"column:country_code" json:"country_code"`
	LatencyMs        *int64          `gorm:"column:latency_ms" json:"latency_ms"`
	ExitIP           *string         `gorm:"column:exit_ip" json:"exit_ip"`
	LastLatencyAt    *int64          `gorm:"column:last_latency_at" json:"last_latency_at"`
	UpdatedAt        time.Time       `gorm:"column:updated_at" json:"-"`
}

type SubscribeTemplate struct {
	ID      int64
	Name    string
	Content *string
}

func (SubscribeTemplate) TableName() string { return "v2_subscribe_templates" }

func (Server) TableName() string { return "v2_server" }

// PersonalAccessToken 兼容 Laravel Sanctum，现有登录会话在切换后继续有效。
type PersonalAccessToken struct {
	ID            int64
	TokenableType string
	TokenableID   int64
	Name          string
	Token         string
	Abilities     *string
	LastUsedAt    *string
	ExpiresAt     *string
	CreatedAt     *string
	UpdatedAt     *string
}

func (PersonalAccessToken) TableName() string { return "personal_access_tokens" }
