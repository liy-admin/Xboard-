package sub

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// 以下为 Surge / Surfboard / Loon / QuantumultX / Shadowrocket / SIP008 共用的小工具，
// 对应 PHP 中各协议类重复实现的辅助逻辑。

// pluginOpts 解析 plugin_opts（"k=v;k2=v2"），与 PHP 的 explode/filter/mapWithKeys 语义一致：
// 空段被忽略，没有 "=" 的段被忽略，键值两侧去空白，同名以后者为准。
func pluginOpts(raw string) map[string]string {
	out := map[string]string{}
	for _, seg := range strings.Split(raw, ";") {
		if seg == "" || seg == "0" || !strings.Contains(seg, "=") {
			continue
		}
		k, v, _ := strings.Cut(seg, "=")
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

// joinLine 对应 implode(',', array_filter($config)) . "\r\n"：丢弃 PHP 视为假的元素（"" 与 "0"）。
func joinLine(parts []string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" && p != "0" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, ",") + "\r\n"
}

// trafficGB 对应 round($bytes / 1024^3, 2)。
func trafficGB(b int64) float64 { return math.Round(float64(b)/1073741824*100) / 100 }

// phpFloat 按 PHP 的浮点转字符串规则输出（去掉多余的小数位）。
func phpFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// expireText 对应 $user['expired_at'] === NULL ? '长期有效' : date('Y-m-d H:i:s', ...)。
func expireText(exp *int64) string {
	if exp == nil {
		return "长期有效"
	}
	return time.Unix(*exp, 0).Format("2006-01-02 15:04:05")
}

// subscribeInfoSurge 生成 Surge / Surfboard 的 SubscribeInfo 面板字段。
func subscribeInfoSurge(ctx *Ctx, appName string) string {
	u, d, total := ctx.User.U, ctx.User.D, ctx.User.TransferEnab
	up := trafficGB(u)
	down := trafficGB(d)
	used := up + down
	totalGB := trafficGB(total)
	unused := totalGB - used
	return fmt.Sprintf("title=%s订阅信息, content=上传流量：%sGB\\n下载流量：%sGB\\n剩余流量：%sGB\\n套餐流量：%sGB\\n到期时间：%s",
		appName, phpFloat(up), phpFloat(down), phpFloat(unused), phpFloat(totalGB), expireText(ctx.User.ExpiredAt))
}

// userinfoHeader 对应 "upload=..; download=..; total=..; expire=.." 形式的订阅用户信息头。
func userinfoHeader(ctx *Ctx) string {
	exp := ""
	if ctx.User.ExpiredAt != nil {
		exp = strconv.FormatInt(*ctx.User.ExpiredAt, 10)
	}
	return fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%s", ctx.User.U, ctx.User.D, ctx.User.TransferEnab, exp)
}

// replaceTemplate 依次替换模板变量，与 PHP 中 str_replace 的顺序保持一致。
func replaceTemplate(tpl string, pairs ...string) string {
	for i := 0; i+1 < len(pairs); i += 2 {
		tpl = strings.ReplaceAll(tpl, pairs[i], pairs[i+1])
	}
	return tpl
}

// rtrimComma 对应 rtrim($s, ', ')。
func rtrimComma(s string) string { return strings.TrimRight(s, ", ") }

// templateOrDefault 优先取后台模板（v2_subscribe_templates），为空时回退到内置默认。
func templateOrDefault(ctx *Ctx, name, def string) string {
	if ctx.Template != nil {
		if raw := ctx.Template(name); strings.TrimSpace(raw) != "" {
			return raw
		}
	}
	return def
}

// shadowrocketCipher / surge 等协议共用的 SS 加密算法白名单判断。
func cipherIn(c string, list ...string) bool {
	for _, x := range list {
		if c == x {
			return true
		}
	}
	return false
}
