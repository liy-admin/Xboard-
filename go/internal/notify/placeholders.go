// Package notify 承载邮件、验证码与 Telegram 通知（对应 MailService、CaptchaService、EmailCodeService、TelegramService）。
package notify

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// placeholderRe 与 MailService::renderPlaceholders 使用的正则一致：{{ key }} 或 {{ key|default }}。
var placeholderRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)(?:\|([^}]*))?\s*\}\}`)

// RenderPlaceholders 渲染邮件模板占位符。未提供或为空的变量使用默认值，没有默认值则保留原文。
func RenderPlaceholders(tpl string, vars map[string]any) string {
	if tpl == "" || len(vars) == 0 {
		return tpl
	}
	var b strings.Builder
	last := 0
	for _, m := range placeholderRe.FindAllStringSubmatchIndex(tpl, -1) {
		b.WriteString(tpl[last:m[0]])
		last = m[1]
		key := tpl[m[2]:m[3]]
		var def *string
		if m[4] >= 0 {
			d := strings.TrimSpace(tpl[m[4]:m[5]])
			def = &d
		}
		v, ok := vars[key]
		if !ok || v == nil || v == "" {
			if def != nil {
				b.WriteString(*def)
			} else {
				b.WriteString(tpl[m[0]:m[1]])
			}
			continue
		}
		b.WriteString(scalarString(v))
	}
	b.WriteString(tpl[last:])
	return b.String()
}

func scalarString(v any) string {
	switch x := v.(type) {
	case bool:
		if x {
			return "1"
		}
		return "0"
	case string:
		return x
	case float64:
		return fmt.Sprint(x)
	case int, int64:
		return fmt.Sprint(x)
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}
