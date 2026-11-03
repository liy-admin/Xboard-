package api

import (
	"embed"
	"fmt"
	"regexp"
)

// 邮件模板在后台编辑页的默认内容，对应 MailTemplateController::getDefaultContent：
// 优先读取内置的 Blade 视图并转换成 {{变量}} 占位符，视图不存在时用内置的硬编码版式。
//
//go:embed mailviews/*.blade.php
var mailViews embed.FS

var (
	bladeVar   = regexp.MustCompile(`\{\{\s*\$([a-zA-Z_]+)\s*\}\}`)
	bladeNl2br = regexp.MustCompile(`\{!!\s*nl2br\(\$([a-zA-Z_]+)\)\s*!!\}`)
	bladeRaw   = regexp.MustCompile(`\{!!\s*\$([a-zA-Z_]+)\s*!!\}`)
)

// bladeToPlaceholder 对应 bladeToPlaceholder：把 Blade 语法转换为编辑用的 {{var}}。
func bladeToPlaceholder(blade string) string {
	out := bladeVar.ReplaceAllString(blade, "{{$1}}")
	out = bladeNl2br.ReplaceAllString(out, "{{$1}}")
	return bladeRaw.ReplaceAllString(out, "{{$1}}")
}

func defaultMailSubject(name, appName string) string {
	switch name {
	case "verify":
		return appName + " - 邮箱验证码"
	case "notify":
		return appName + " - 站点通知"
	case "remindExpire":
		return appName + " - 服务即将到期"
	case "remindTraffic":
		return appName + " - 流量使用提醒"
	case "mailLogin":
		return appName + " - 邮件登录"
	}
	return appName
}

func defaultMailContent(name string) string {
	if b, err := mailViews.ReadFile("mailviews/" + name + ".blade.php"); err == nil {
		return bladeToPlaceholder(string(b))
	}
	return hardcodedMailDefault(name)
}

func hardcodedMailDefault(name string) string {
	layout := func(title, body string) string {
		return fmt.Sprintf(`<div style="background: #eee">
    <table width="600" border="0" align="center" cellpadding="0" cellspacing="0">
        <tbody>
        <tr>
            <td>
                <div style="background:#fff">
                    <table width="100%%" border="0" cellspacing="0" cellpadding="0">
                        <thead>
                        <tr>
                            <td valign="middle" style="padding-left:30px;background-color:#415A94;color:#fff;padding:20px 40px;font-size: 21px;">{{name}}</td>
                        </tr>
                        </thead>
                        <tbody>
                        <tr style="padding:40px 40px 0 40px;display:table-cell">
                            <td style="font-size:24px;line-height:1.5;color:#000;margin-top:40px">%s</td>
                        </tr>
                        <tr>
                            <td style="font-size:14px;color:#333;padding:24px 40px 0 40px">
                                尊敬的用户您好！<br /><br />%s
                            </td>
                        </tr>
                        </tbody>
                    </table>
                </div>
                <div>
                    <table width="100%%" border="0" cellspacing="0" cellpadding="0">
                        <tbody>
                        <tr>
                            <td style="padding:20px 40px;font-size:12px;color:#999;line-height:20px;background:#f7f7f7"><a href="{{url}}" style="font-size:14px;color:#929292">返回{{name}}</a></td>
                        </tr>
                        </tbody>
                    </table>
                </div>
            </td>
        </tr>
        </tbody>
    </table>
</div>`, title, body)
	}
	switch name {
	case "verify":
		return layout("邮箱验证码", "您的验证码是：{{code}}，请在 5 分钟内进行验证。如果该验证码不为您本人申请，请无视。")
	case "notify":
		return layout("网站通知", "{{content}}")
	case "remindExpire":
		return layout("服务到期提醒", "您的服务即将在24小时内到期，如需继续使用请及时续费。")
	case "remindTraffic":
		return layout("流量使用提醒", "您的流量使用已达到80%，请注意流量使用情况。")
	case "mailLogin":
		return layout("登入到{{name}}", `您正在登入到{{name}}, 请在 5 分钟内点击下方链接进行登入。如果您未授权该登入请求，请无视。<a href="{{link}}">{{link}}</a>`)
	}
	return layout("通知", "{{content}}")
}
