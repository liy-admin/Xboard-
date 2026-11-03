package model

import "time"

// MailTemplate 对应 v2_mail_templates（后台自定义邮件模板，时间列为 datetime）。
type MailTemplate struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (MailTemplate) TableName() string { return "v2_mail_templates" }
