package model

import "encoding/json"

// ServerGroup 对应 v2_server_group。
type ServerGroup struct {
	ID        int64  `gorm:"primaryKey" json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (ServerGroup) TableName() string { return "v2_server_group" }

// ServerRoute 对应 v2_server_route（路由规则：匹配列表 + 动作）。
type ServerRoute struct {
	ID          int64           `gorm:"primaryKey" json:"id"`
	Remarks     string          `json:"remarks"`
	Match       json.RawMessage `gorm:"type:text" json:"match"`
	Action      string          `json:"action"`
	ActionValue *string         `gorm:"column:action_value" json:"action_value"`
	CreatedAt   int64           `json:"created_at"`
	UpdatedAt   int64           `json:"updated_at"`
}

func (ServerRoute) TableName() string { return "v2_server_route" }
