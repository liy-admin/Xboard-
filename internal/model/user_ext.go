package model

import "encoding/json"

// 用户侧扩展表（邀请、推荐、抽奖、公告、知识库、工单、流量统计、设备绑定）。

type InviteCode struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	UserID    int64  `gorm:"column:user_id" json:"user_id"`
	Code      string `json:"code"`
	Status    bool   `json:"status"`
	PV        int64  `json:"pv"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (InviteCode) TableName() string { return "v2_invite_code" }

// CommissionLog 对应 v2_commission_log。
type CommissionLog struct {
	ID           int64  `gorm:"primaryKey" json:"id"`
	InviteUserID int64  `gorm:"column:invite_user_id" json:"invite_user_id"`
	UserID       int64  `gorm:"column:user_id" json:"user_id"`
	TradeNo      string `gorm:"column:trade_no" json:"trade_no"`
	OrderAmount  int64  `json:"order_amount"`
	GetAmount    int64  `json:"get_amount"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

func (CommissionLog) TableName() string { return "v2_commission_log" }

type Notice struct {
	ID        int64           `gorm:"primaryKey" json:"id"`
	Title     string          `json:"title"`
	Content   string          `json:"content"`
	Show      bool            `json:"show"`
	ImgURL    *string         `gorm:"column:img_url" json:"img_url"`
	URL       *string         `gorm:"column:url" json:"url"`
	Tags      json.RawMessage `gorm:"type:json" json:"tags"`
	Sort      *int            `json:"sort"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

func (Notice) TableName() string { return "v2_notice" }

type Knowledge struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	Language  string `json:"language"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Sort      *int   `json:"sort"`
	Show      bool   `json:"show"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (Knowledge) TableName() string { return "v2_knowledge" }

type LuckyDrawLog struct {
	ID            int64 `gorm:"primaryKey" json:"id"`
	UserID        int64 `gorm:"column:user_id" json:"user_id"`
	IsWin         bool  `gorm:"column:is_win" json:"is_win"`
	HoursCredited int64 `gorm:"column:hours_credited" json:"hours_credited"`
	CreatedAt     int64 `json:"created_at"`
	UpdatedAt     int64 `json:"updated_at"`
}

func (LuckyDrawLog) TableName() string { return "v2_lucky_draw_log" }

type ReferralDurationLog struct {
	ID              int64  `gorm:"primaryKey" json:"id"`
	InviterID       int64  `gorm:"column:inviter_id" json:"inviter_id"`
	InviteeID       int64  `gorm:"column:invitee_id" json:"invitee_id"`
	TradeNo         string `gorm:"column:trade_no" json:"trade_no"`
	SecondsCredited int64  `gorm:"column:seconds_credited" json:"seconds_credited"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

func (ReferralDurationLog) TableName() string { return "v2_referral_duration_log" }

// Ticket 对应 v2_ticket。status: 0 开启 1 关闭；reply_status: 0 待回复 1 已回复。
type Ticket struct {
	ID              int64  `gorm:"primaryKey" json:"id"`
	UserID          int64  `gorm:"column:user_id" json:"user_id"`
	Subject         string `json:"subject"`
	Level           int    `json:"level"`
	Status          int    `json:"status"`
	ReplyStatus     int    `json:"reply_status"`
	LastReplyUserID int64  `gorm:"column:last_reply_user_id" json:"-"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
}

func (Ticket) TableName() string { return "v2_ticket" }

type TicketMessage struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	UserID    int64  `gorm:"column:user_id" json:"user_id"`
	TicketID  int64  `gorm:"column:ticket_id" json:"ticket_id"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (TicketMessage) TableName() string { return "v2_ticket_message" }

// StatUser 对应 v2_stat_user（按用户、节点倍率、时间片记录的流量）。
type StatUser struct {
	ID         int64   `gorm:"primaryKey" json:"id"`
	UserID     int64   `gorm:"column:user_id" json:"user_id"`
	ServerRate float64 `gorm:"column:server_rate" json:"server_rate"`
	U          int64   `json:"u"`
	D          int64   `json:"d"`
	RecordType string  `gorm:"column:record_type" json:"record_type"`
	RecordAt   int64   `gorm:"column:record_at" json:"record_at"`
	CreatedAt  int64   `json:"created_at"`
	UpdatedAt  int64   `json:"updated_at"`
}

func (StatUser) TableName() string { return "v2_stat_user" }

// DeviceBinding 对应 v2_device_binding。
type DeviceBinding struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	DeviceID  string `gorm:"column:device_id" json:"device_id"`
	UserID    int64  `gorm:"column:user_id" json:"user_id"`
	Type      string `json:"type"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (DeviceBinding) TableName() string { return "v2_device_binding" }

// DeviceTrialLog 对应 v2_device_trial_log：每次给设备发放试用套餐记一行。
type DeviceTrialLog struct {
	ID        int64   `gorm:"primaryKey" json:"id"`
	DeviceID  string  `gorm:"column:device_id" json:"device_id"`
	UserID    int64   `gorm:"column:user_id" json:"user_id"`
	Email     string  `json:"email"`
	Source    string  `json:"source"`
	PlanID    *int64  `gorm:"column:plan_id" json:"plan_id"`
	ExpiredAt *int64  `gorm:"column:expired_at" json:"expired_at"`
	IP        *string `gorm:"column:ip" json:"ip"`
	CreatedAt int64   `json:"created_at"`
}

func (DeviceTrialLog) TableName() string { return "v2_device_trial_log" }
