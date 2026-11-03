package model

import (
	"encoding/json"
	"time"
)

// ServerMachine 对应 v2_server_machine：一台宿主机，可挂载多个节点。时间列为 datetime。
type ServerMachine struct {
	ID         int64           `gorm:"primaryKey" json:"id"`
	Name       string          `json:"name"`
	Token      string          `json:"token"`
	Notes      *string         `json:"notes"`
	IsActive   bool            `gorm:"column:is_active" json:"is_active"`
	LastSeenAt *int64          `gorm:"column:last_seen_at" json:"last_seen_at"`
	LoadStatus json.RawMessage `gorm:"column:load_status;type:json" json:"load_status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func (ServerMachine) TableName() string { return "v2_server_machine" }

// ServerMachineLoadHistory 对应 v2_server_machine_load_history。
type ServerMachineLoadHistory struct {
	ID          int64    `gorm:"primaryKey" json:"id"`
	MachineID   int64    `gorm:"column:machine_id" json:"machine_id"`
	CPU         float64  `gorm:"column:cpu" json:"cpu"`
	MemTotal    int64    `gorm:"column:mem_total" json:"mem_total"`
	MemUsed     int64    `gorm:"column:mem_used" json:"mem_used"`
	DiskTotal   int64    `gorm:"column:disk_total" json:"disk_total"`
	DiskUsed    int64    `gorm:"column:disk_used" json:"disk_used"`
	NetInSpeed  *float64 `gorm:"column:net_in_speed" json:"net_in_speed"`
	NetOutSpeed *float64 `gorm:"column:net_out_speed" json:"net_out_speed"`
	RecordedAt  int64    `gorm:"column:recorded_at" json:"recorded_at"`
}

func (ServerMachineLoadHistory) TableName() string { return "v2_server_machine_load_history" }
