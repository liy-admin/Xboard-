package sub

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// 对应 App\Protocols\Clash / ClashMeta / Stash：读取 YAML 模板，填入节点与策略组，输出 YAML。

//go:embed default.clash.yaml
var defaultClash []byte

// appClashTemplate 对应 resources/rules/app.clash.yaml：client/app/getConfig（旧版桌面客户端）使用的专用模板。
//
//go:embed app.clash.yaml
var appClashTemplate []byte

// AppClashTemplate 返回内嵌的 App 专用 Clash 模板。
func AppClashTemplate() string { return string(appClashTemplate) }

var clashAllowed = map[string]bool{
	"shadowsocks": true, "vmess": true, "trojan": true, "socks": true, "http": true,
}

var clashSSCiphers = map[string]bool{
	"aes-128-gcm": true, "aes-192-gcm": true, "aes-256-gcm": true, "chacha20-ietf-poly1305": true,
}

var metaAllowed = map[string]bool{
	"shadowsocks": true, "vmess": true, "trojan": true, "vless": true, "hysteria": true,
	"tuic": true, "anytls": true, "socks": true, "http": true, "mieru": true,
}

var stashAllowed = map[string]bool{
	"shadowsocks": true, "vmess": true, "vless": true, "hysteria": true, "trojan": true,
	"tuic": true, "anytls": true, "socks": true, "http": true,
}

// strictNet 生成 "strict" 白名单规则：字段取值不在表内即不兼容。
func strictNet(vals ...string) map[string]string {
	m := map[string]string{"__strict__": "1"}
	for _, v := range vals {
		m[v] = "0.0.0"
	}
	return m
}

func metaRequirements() requirements {
	req := requirements{"*": {
		"vless":  {"protocol_settings.network": strictNet("tcp", "ws", "grpc", "http", "h2", "httpupgrade", "xhttp")},
		"vmess":  {"protocol_settings.network": strictNet("tcp", "ws", "grpc", "http", "h2", "httpupgrade")},
		"trojan": {"protocol_settings.network": strictNet("tcp", "ws", "grpc", "httpupgrade")},
	}}
	add := func(client, typ, field string, vals map[string]string) {
		if req[client] == nil {
			req[client] = map[string]map[string]map[string]string{}
		}
		if req[client][typ] == nil {
			req[client][typ] = map[string]map[string]string{}
		}
		req[client][typ][field] = vals
	}
	add("nekobox", "hysteria", "protocol_settings.version", map[string]string{"1": "0.0.0", "2": "1.2.7"})
	add("clashmetaforandroid", "hysteria", "protocol_settings.version", map[string]string{"2": "2.9.0"})
	add("nekoray", "hysteria", "protocol_settings.version", map[string]string{"2": "3.24"})
	add("verge", "hysteria", "protocol_settings.version", map[string]string{"2": "1.3.8"})
	add("ClashX Meta", "hysteria", "protocol_settings.version", map[string]string{"2": "1.3.5"})
	add("flclash", "hysteria", "protocol_settings.version", map[string]string{"2": "0.8.0"})
	ech := map[string]string{"1": "1.19.9"}
	for _, c := range []string{"meta", "verge", "flclash", "nekobox", "clashmetaforandroid"} {
		add(c, "vmess", "protocol_settings.tls_settings.ech.enabled", ech)
		add(c, "vless", "protocol_settings.tls_settings.ech.enabled", ech)
		add(c, "trojan", "protocol_settings.tls_settings.ech.enabled", ech)
		add(c, "anytls", "protocol_settings.tls.ech.enabled", ech)
	}
	return req
}

func stashRequirements() requirements {
	return requirements{
		"*": {
			"trojan": {"protocol_settings.tls": {"2": "9999.0.0"}},
			"vmess":  {"protocol_settings.network": {"httpupgrade": "9999.0.0"}},
		},
		"stash": {
			"vless": {
				"protocol_settings.tls":  {"2": "3.1.0"},
				"protocol_settings.flow": {"xtls-rprx-vision": "3.1.0"},
			},
			"hysteria": {"protocol_settings.version": {"1": "2.0.0", "2": "2.5.0"}},
			"shadowsocks": {"protocol_settings.cipher": {
				"2022-blake3-aes-128-gcm": "3.0.0", "2022-blake3-aes-256-gcm": "3.0.0",
				"2022-blake3-chacha20-poly1305": "3.0.0",
			}},
		},
	}
}

// delimRegex 对应 isRegex/isMatch：仅带分隔符的 PCRE 写法（如 "/HK/i"）才视作正则；普通节点名返回 nil。
func delimRegex(src string) *regexp.Regexp {
	t := strings.TrimSpace(src)
	if len(t) < 2 || !strings.ContainsRune("/#~@%!|", rune(t[0])) {
		return nil
	}
	d := t[0]
	i := strings.LastIndexByte(t, d)
	if i == 0 {
		return nil
	}
	flags := ""
	for _, f := range t[i+1:] {
		if f == 'i' || f == 'm' || f == 's' {
			flags += string(f)
		}
	}
	src2 := t[1:i]
	if flags != "" {
		src2 = "(?" + flags + ")" + src2
	}
	re, err := regexp.Compile(src2)
	if err != nil {
		return nil
	}
	return re
}

func toStrings(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		out = append(out, Str(x))
	}
	return out
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// fillGroup 对应模板策略组填充：含正则项时仅保留非正则项并追加命中节点；否则追加全部节点。
func fillGroup(g Map, names []string) {
	cur := toStrings(g["proxies"])
	hasRegex := false
	if len(names) > 0 {
		for _, src := range cur {
			if delimRegex(src) != nil {
				hasRegex = true
				break
			}
		}
	}
	if hasRegex {
		var keep, matched []string
		for _, src := range cur {
			re := delimRegex(src)
			if re == nil {
				keep = append(keep, src)
				continue
			}
			for _, dst := range names {
				if re.MatchString(dst) {
					matched = append(matched, dst)
				}
			}
		}
		g["proxies"] = toAny(append(keep, matched...))
		return
	}
	g["proxies"] = toAny(append(cur, names...))
}

type clashOut struct {
	Body    []byte
	Headers map[string]string
}

// BuildClash 生成 Clash / ClashMeta / Stash 的 YAML 订阅。
func BuildClash(ctx *Ctx, kind ClashKind) (*clashOut, error) {
	tmplName, allowed, req := "clash", clashAllowed, requirements(nil)
	switch kind {
	case ClashKindMeta:
		tmplName, allowed, req = "clashmeta", metaAllowed, metaRequirements()
	case ClashKindStash:
		tmplName, allowed, req = "stash", stashAllowed, stashRequirements()
	}
	raw := ""
	if ctx.Template != nil {
		raw = ctx.Template(tmplName)
	}
	data := []byte(raw)
	if strings.TrimSpace(raw) == "" {
		data = defaultClash
	}
	var cfg Map
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("clash template: %w", err)
	}
	if cfg == nil {
		cfg = Map{}
	}

	var proxies []any
	var names []string
	for _, s := range filterServers(ctx, allowed, req) {
		p := ps(s)
		pw := Str(s["password"])
		typ := Str(s["type"])
		if kind == ClashKindClash && typ == "shadowsocks" && !clashSSCiphers[Str(p["cipher"])] {
			continue
		}
		var m Map
		switch typ {
		case "shadowsocks":
			m = clashSS(pw, s, kind)
		case "vmess":
			m = clashVmess(pw, s, kind)
		case "trojan":
			m = clashTrojan(pw, s, kind)
		case "socks":
			m = clashSocks5(pw, s, kind)
		case "http":
			m = clashHTTP(pw, s, kind)
		case "vless":
			m = clashVless(pw, s, kind)
		case "hysteria":
			m = clashHysteria(pw, s, kind)
		case "tuic":
			m = clashTuic(pw, s, kind)
		case "anytls":
			m = clashAnyTLS(pw, s, kind)
		case "mieru":
			m = clashMieru(pw, s)
		default:
			continue
		}
		proxies = append(proxies, m)
		names = append(names, Str(s["name"]))
	}

	existing, _ := cfg["proxies"].([]any)
	cfg["proxies"] = append(existing, proxies...)

	groups, _ := cfg["proxy-groups"].([]any)
	var kept []any
	for _, gi := range groups {
		g, ok := gi.(Map)
		if !ok {
			continue
		}
		fillGroup(g, names)
		if len(toStrings(g["proxies"])) > 0 {
			kept = append(kept, g)
		}
	}
	cfg["proxy-groups"] = kept

	if host := ctx.Host; host != "" {
		rules, _ := cfg["rules"].([]any)
		cfg["rules"] = append([]any{"DOMAIN," + host + ",DIRECT"}, rules...)
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	body := strings.ReplaceAll(string(out), "$app_name", ctx.AppName)

	h := map[string]string{
		"content-type":            "text/yaml",
		"subscription-userinfo":   userInfo(ctx),
		"profile-update-interval": "24",
		"content-disposition":     "attachment;filename*=UTF-8''" + rawurlencode(ctx.AppName),
	}
	if kind == ClashKindClash {
		h["profile-web-page-url"] = ctx.AppURL
	}
	return &clashOut{Body: []byte(body), Headers: h}, nil
}
