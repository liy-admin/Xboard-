package notify

import (
	"fmt"
	"strings"
)

// 邮件模板渲染，对应 MailService::sendEmail 中的 DB 模板分支与 buildSafeVars。

// EscapeHTML 对应 Laravel e()：转义 & < > " 与单引号（双重转义，与 e() 默认行为一致）。
func EscapeHTML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#039;").Replace(s)
}

// Nl2br 对应 PHP nl2br：在每个换行符前插入 <br />。
func Nl2br(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "<br />\r\n")
	return strings.ReplaceAll(s, "\n", "<br />\n")
}

// BuildSafeVars 对应 MailService::buildSafeVars：标量统一转义；content 按 content_mode 决定是否转义与 nl2br。
func BuildSafeVars(values map[string]any) map[string]string {
	out := map[string]string{}
	for k, v := range values {
		switch v.(type) {
		case string, bool, float64, int, int64:
			out[k] = EscapeHTML(scalarString(v))
		}
	}
	if c, ok := values["content"].(string); ok {
		if mode, _ := values["content_mode"].(string); mode == "text" {
			out["content"] = Nl2br(EscapeHTML(c))
		} else {
			out["content"] = c
		}
	}
	return out
}

// RenderDBTemplate 使用 v2_mail_templates 中的 subject/content 渲染。subject 渲染结果为空时沿用原主题。
func RenderDBTemplate(subject, tplSubject, tplContent string, values map[string]any) (string, string) {
	vars := BuildSafeVars(values)
	renderedSubject := RenderPlaceholdersStr(tplSubject, vars)
	if renderedSubject != "" {
		subject = renderedSubject
	}
	return subject, RenderPlaceholdersStr(tplContent, vars)
}

// RenderPlaceholdersStr 与 RenderPlaceholders 相同，但变量已是字符串。
func RenderPlaceholdersStr(tpl string, vars map[string]string) string {
	anyVars := make(map[string]any, len(vars))
	for k, v := range vars {
		anyVars[k] = v
	}
	return RenderPlaceholders(tpl, anyVars)
}

// DefaultMailBody 在没有数据库模板时的兜底正文。PHP 此时使用 resources/views/mail/default 下的 Blade 视图，
// 这里不复用 Blade 的整页版式，但每种模板都带上其关键内容与相同文案（验证码、登录链接、提醒说明）。
func DefaultMailBody(templateName string, values map[string]any) string {
	safe := BuildSafeVars(values)
	name, url := safe["name"], safe["url"]
	var title, text, extra string
	switch templateName {
	case "verify":
		title = "邮箱验证码"
		text = "请使用以下验证码完成验证，有效期 5 分钟。如非本人操作，请忽略此邮件。"
		extra = fmt.Sprintf(`<p style="font-size:28px;font-weight:700;letter-spacing:6px">%s</p>`, safe["code"])
	case "mailLogin":
		title = "登录确认"
		text = fmt.Sprintf("点击下方链接登录到 %s，链接有效期 5 分钟。如非本人操作，请忽略此邮件。", name)
		extra = fmt.Sprintf(`<p><a href="%s">确认登录</a></p><p>如果链接无法点击，请复制以下地址到浏览器中打开：<br>%s</p>`, safe["link"], safe["link"])
	case "remindExpire":
		title = "订阅即将到期"
		text = "您的订阅服务将在 24 小时内到期。为避免服务中断，请及时续费。如您已完成续费，请忽略此提醒。"
	case "remindTraffic":
		title = "流量使用提醒"
		text = "您本月的套餐流量已使用 80% 以上。请合理安排使用，避免提前耗尽。如需更多流量，可前往面板升级套餐。"
	default:
		title = "网站通知"
		text = safe["content"]
	}
	body := fmt.Sprintf(`<div style="font-family:sans-serif"><h3>%s</h3><h4>%s</h4><p>%s</p>%s`, name, title, text, extra)
	if url != "" {
		body += fmt.Sprintf(`<p><a href="%s">%s</a></p>`, url, url)
	}
	return body + `<p style="color:#888">此邮件由系统自动发送，请勿直接回复。</p></div>`
}
