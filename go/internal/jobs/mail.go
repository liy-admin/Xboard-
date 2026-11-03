package jobs

import (
	"strings"
	"time"

	"xboard/internal/notify"
)

// smtpFromSettings 读取后台邮件设置（email_host 等），与 MailService::sendEmail 中的 Config::set 对应。
func (e *Env) smtpFromSettings() notify.SMTP {
	return notify.SMTP{
		Host:       e.Set.Get("email_host", ""),
		Port:       e.Set.Int("email_port", 465),
		Encryption: e.Set.Get("email_encryption", ""),
		Username:   e.Set.Get("email_username", ""),
		Password:   e.Set.Get("email_password", ""),
		From:       e.Set.Get("email_from_address", ""),
		FromName:   e.Set.Get("app_name", "Lets VPN"),
		Timeout:    30 * time.Second,
	}
}

// SendTemplateMail 对应 MailService::sendEmail：优先使用数据库模板，否则使用兜底正文；结果写入 v2_mail_log。
func (e *Env) SendTemplateMail(to, subject, templateName string, values map[string]any) error {
	var tpl struct {
		Subject string
		Content string
	}
	hasDB := e.DB.Raw("SELECT subject, content FROM v2_mail_templates WHERE name = ? LIMIT 1", templateName).
		Scan(&tpl).RowsAffected > 0
	logTemplate := "mail.default." + templateName
	body := notify.DefaultMailBody(templateName, values)
	if hasDB {
		subject, body = notify.RenderDBTemplate(subject, tpl.Subject, tpl.Content, values)
		logTemplate = "db:" + templateName
	}
	err := e.smtpFromSettings().Send(to, subject, body) // 未配置 email_host 时 Send 直接返回错误
	var errText any
	if err != nil {
		errText = err.Error()
	}
	now := e.nowTs()
	if logErr := e.DB.Exec(
		"INSERT INTO v2_mail_log (email, subject, template_name, error, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		to, subject, logTemplate, errText, now, now).Error; logErr != nil {
		return logErr
	}
	return err
}

// mailTrafficWarn 对应 MailService::remindTrafficIsWarnValue：使用率在 [80%, 100%) 之间。
func mailTrafficWarn(u, d, transfer int64) bool {
	ud := u + d
	if ud == 0 || transfer == 0 {
		return false
	}
	p := float64(ud) / float64(transfer) * 100
	return p >= 80 && p < 100
}

// replaceAppName 替换 :app_name 占位符（与 __('...', ['app_name' => ...]) 的效果一致）。
func replaceAppName(s, app string) string { return strings.ReplaceAll(s, ":app_name", app) }
