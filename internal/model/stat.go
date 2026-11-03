package model

// Stat 对应 v2_stat（按日汇总，record_type = d）。金额单位为分。
type Stat struct {
	ID                int64  `gorm:"primaryKey" json:"id"`
	RecordAt          int64  `gorm:"column:record_at" json:"record_at"`
	RecordType        string `gorm:"column:record_type" json:"record_type"`
	OrderCount        int64  `gorm:"column:order_count" json:"order_count"`
	OrderTotal        int64  `gorm:"column:order_total" json:"order_total"`
	CommissionCount   int64  `gorm:"column:commission_count" json:"commission_count"`
	CommissionTotal   int64  `gorm:"column:commission_total" json:"commission_total"`
	PaidCount         int64  `gorm:"column:paid_count" json:"paid_count"`
	PaidTotal         int64  `gorm:"column:paid_total" json:"paid_total"`
	RegisterCount     int64  `gorm:"column:register_count" json:"register_count"`
	InviteCount       int64  `gorm:"column:invite_count" json:"invite_count"`
	TransferUsedTotal string `gorm:"column:transfer_used_total" json:"transfer_used_total"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

func (Stat) TableName() string { return "v2_stat" }

// StatServer 对应 v2_stat_server（节点流量按时间片记录）。
type StatServer struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	ServerID   int64  `gorm:"column:server_id" json:"server_id"`
	U          int64  `json:"u"`
	D          int64  `json:"d"`
	RecordType string `gorm:"column:record_type" json:"record_type"`
	RecordAt   int64  `gorm:"column:record_at" json:"record_at"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

func (StatServer) TableName() string { return "v2_stat_server" }
