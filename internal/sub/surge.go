package sub

import (
	_ "embed"
	"strings"
)

// Surge 与 Surfboard 配置生成（对应 app/Protocols/Surge.php、Surfboard.php）。
// 两者的代理行格式不同：Surge 为 "name = type,..."，Surfboard 为 "name=type,..."。

//go:embed default.surge.conf
var defaultSurge string

//go:embed default.surfboard.conf
var defaultSurfboard string

var surgeAllowed = map[string]bool{
	"shadowsocks": true, "vmess": true, "trojan": true, "hysteria": true,
	"anytls": true, "socks": true, "http": true,
}

var surgeRequirements = requirements{"surge": {"hysteria": {"protocol_settings.version": {"2": "2398"}}}}

var surgeSSCiphers = []string{"aes-128-gcm", "aes-192-gcm", "aes-256-gcm", "chacha20-ietf-poly1305",
	"2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm"}

var surfboardAllowed = map[string]bool{"shadowsocks": true, "vmess": true, "trojan": true, "anytls": true}

var surfboardSSCiphers = []string{"aes-128-gcm", "aes-192-gcm", "aes-256-gcm", "chacha20-ietf-poly1305",
	"2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305"}

// proxyLine 生成一条代理配置行；sep 为 " = "（Surge）或 "="（Surfboard）。
func proxyLine(s Map, typ, sep string, rest ...string) string {
	head := Str(s["name"]) + sep + typ
	return joinLine(append([]string{head, Str(s["host"]), Str(s["port"])}, rest...))
}

// ssPluginObfs 处理 Surge/Surfboard 的 simple-obfs 插件参数。
func ssPluginObfs(s Map, out []string) []string {
	if !Truthy(Get(s, "protocol_settings.plugin")) || !Truthy(Get(s, "protocol_settings.plugin_opts")) {
		return out
	}
	if Str(Get(s, "protocol_settings.plugin")) != "obfs" {
		return out
	}
	opts := pluginOpts(Str(Get(s, "protocol_settings.plugin_opts")))
	out = append(out, "obfs="+opts["obfs"])
	if v, ok := opts["obfs-host"]; ok {
		out = append(out, "obfs-host="+v)
	}
	if v, ok := opts["path"]; ok {
		out = append(out, "obfs-uri="+v)
	}
	return out
}

// buildSurgeSS 对应 Surge::buildShadowsocks。
func buildSurgeSS(pw string, s Map) string {
	cfg := []string{
		Str(s["name"]) + " = ss", Str(s["host"]), Str(s["port"]),
		"encrypt-method=" + Str(Get(s, "protocol_settings.cipher")),
		"password=" + pw, "tfo=true", "udp-relay=true",
	}
	return joinLine(ssPluginObfs(s, cfg))
}

// buildSurfboardSS 对应 Surfboard::buildShadowsocks（行格式为 name=ss）。
func buildSurfboardSS(pw string, s Map) string {
	cfg := []string{
		Str(s["name"]) + "=ss", Str(s["host"]), Str(s["port"]),
		"encrypt-method=" + Str(Get(s, "protocol_settings.cipher")),
		"password=" + pw, "tfo=true", "udp-relay=true",
	}
	return joinLine(ssPluginObfs(s, cfg))
}

// vmessTLSAndWS 两者共用的 vmess TLS / WS 参数。sep 决定 "name=vmess" 形式。
func vmessTLSAndWS(s Map, cfg []string) []string {
	if Truthy(Get(s, "protocol_settings.tls")) {
		cfg = append(cfg, "tls=true")
		if tls := Get(s, "protocol_settings.tls_settings"); Truthy(tls) {
			if Truthy(Get(tls, "allow_insecure")) {
				cfg = append(cfg, "skip-cert-verify=true")
			}
			if sn := Str(Get(tls, "server_name")); sn != "" {
				cfg = append(cfg, "sni="+sn)
			}
		}
	}
	if Str(Get(s, "protocol_settings.network")) == "ws" {
		cfg = append(cfg, "ws=true")
		if ws := Get(s, "protocol_settings.network_settings"); Truthy(ws) {
			if p := Str(Get(ws, "path")); p != "" {
				cfg = append(cfg, "ws-path="+p)
			}
			if h := Str(Get(ws, "headers.Host")); h != "" {
				cfg = append(cfg, "ws-headers=Host:"+h)
			}
		}
	}
	return cfg
}

func buildSurgeVmess(uuid string, s Map) string {
	cfg := []string{Str(s["name"]) + " = vmess", Str(s["host"]), Str(s["port"]),
		"username=" + uuid, "vmess-aead=true", "tfo=true", "udp-relay=true"}
	return joinLine(vmessTLSAndWS(s, cfg))
}

func buildSurfboardVmess(uuid string, s Map) string {
	cfg := []string{Str(s["name"]) + "=vmess", Str(s["host"]), Str(s["port"]),
		"username=" + uuid, "vmess-aead=true", "tfo=true", "udp-relay=true"}
	return joinLine(vmessTLSAndWS(s, cfg))
}

// trojanSNI 返回 tls_settings.server_name 对应的 sni 项（空则为空串）。
func trojanSNI(s Map) string {
	if sn := Str(Get(s, "protocol_settings.tls_settings.server_name")); sn != "" {
		return "sni=" + sn
	}
	return ""
}

func buildSurgeTrojan(pw string, s Map) string {
	cfg := []string{Str(s["name"]) + " = trojan", Str(s["host"]), Str(s["port"]),
		"password=" + pw, trojanSNI(s), "tfo=true", "udp-relay=true"}
	if Truthy(Get(s, "protocol_settings.tls_settings.allow_insecure")) {
		cfg = append(cfg, "skip-cert-verify=true")
	}
	return joinLine(cfg)
}

func buildSurfboardTrojan(pw string, s Map) string {
	cfg := []string{Str(s["name"]) + "=trojan", Str(s["host"]), Str(s["port"]),
		"password=" + pw, trojanSNI(s), "tfo=true", "udp-relay=true"}
	if Truthy(Get(s, "protocol_settings.tls_settings.allow_insecure")) {
		cfg = append(cfg, "skip-cert-verify=true")
	}
	return joinLine(cfg)
}

// anytlsTail 返回 tls.server_name / tls.allow_insecure 的参数。
func anytlsTail(s Map) []string {
	var out []string
	if sn := Str(Get(s, "protocol_settings.tls.server_name")); sn != "" {
		out = append(out, "sni="+sn)
	}
	if Truthy(Get(s, "protocol_settings.tls.allow_insecure")) {
		out = append(out, "skip-cert-verify=true")
	}
	return out
}

func buildSurgeAnyTLS(pw string, s Map) string {
	cfg := append([]string{Str(s["name"]) + " = anytls", Str(s["host"]), Str(s["port"]), "password=" + pw}, anytlsTail(s)...)
	return joinLine(cfg)
}

func buildSurfboardAnyTLS(pw string, s Map) string {
	cfg := append([]string{Str(s["name"]) + "=anytls", Str(s["host"]), Str(s["port"]), "password=" + pw, "tfo=true", "udp-relay=true"}, anytlsTail(s)...)
	return joinLine(cfg)
}

// buildSurgeHysteria 对应 Surge::buildHysteria；非 v2 配置返回空串。
func buildSurgeHysteria(pw string, s Map) string {
	if Str(Get(s, "protocol_settings.version")) != "2" {
		return ""
	}
	cfg := []string{Str(s["name"]) + " = hysteria2", Str(s["host"]), Str(s["port"]), "password=" + pw}
	if sn := Str(Get(s, "protocol_settings.tls.server_name")); sn != "" {
		cfg = append(cfg, "sni="+sn)
	}
	cfg = append(cfg, "udp-relay=true")
	if up := Str(Get(s, "protocol_settings.bandwidth.up")); up != "" && up != "0" {
		cfg = append(cfg, "upload-bandwidth="+up)
	}
	if down := Str(Get(s, "protocol_settings.bandwidth.down")); down != "" && down != "0" {
		cfg = append(cfg, "download-bandwidth="+down)
	}
	if Truthy(Get(s, "protocol_settings.tls.allow_insecure")) {
		cfg = append(cfg, "skip-cert-verify=true")
	}
	return joinLine(cfg)
}

// buildSurgeSocksLike 对应 Surge::buildSocks / buildHttp，区别仅在类型名与 udp-relay。
func buildSurgeSocksLike(pw string, s Map, plain, tls string, udp bool) string {
	typ := plain
	if Truthy(Get(s, "protocol_settings.tls")) {
		typ = tls
	}
	cfg := []string{Str(s["name"]) + " = " + typ, Str(s["host"]), Str(s["port"]), pw, pw}
	if Truthy(Get(s, "protocol_settings.tls")) {
		if sn := Str(Get(s, "protocol_settings.tls_settings.server_name")); sn != "" {
			cfg = append(cfg, "sni="+sn)
		}
		if Truthy(Get(s, "protocol_settings.tls_settings.allow_insecure")) {
			cfg = append(cfg, "skip-cert-verify=true")
		}
	}
	if udp {
		cfg = append(cfg, "udp-relay=true")
	}
	return joinLine(cfg)
}

// BuildSurge 生成 Surge 配置文件。
func BuildSurge(ctx *Ctx) (string, string, map[string]string) {
	var proxies, names strings.Builder
	for _, s := range filterServers(ctx, surgeAllowed, surgeRequirements) {
		pw := Str(s["password"])
		name := Str(s["name"])
		switch Str(s["type"]) {
		case "shadowsocks":
			if !cipherIn(Str(Get(s, "protocol_settings.cipher")), surgeSSCiphers...) {
				continue
			}
			proxies.WriteString(buildSurgeSS(pw, s))
		case "vmess":
			proxies.WriteString(buildSurgeVmess(pw, s))
		case "trojan":
			proxies.WriteString(buildSurgeTrojan(pw, s))
		case "hysteria":
			proxies.WriteString(buildSurgeHysteria(pw, s))
		case "anytls":
			proxies.WriteString(buildSurgeAnyTLS(pw, s))
		case "socks":
			proxies.WriteString(buildSurgeSocksLike(pw, s, "socks5", "socks5-tls", true))
		case "http":
			proxies.WriteString(buildSurgeSocksLike(pw, s, "http", "https", false))
		default:
			continue
		}
		names.WriteString(name + ", ")
	}
	appName := ctx.AppName
	domain := ctx.Host
	subsURL := ctx.SubscribeURL(ctx.User.Token, domain)
	cfg := replaceTemplate(templateOrDefault(ctx, "surge", defaultSurge),
		"$subs_link", subsURL,
		"$subs_domain", domain,
		"$proxies", proxies.String(),
		"$proxy_group", rtrimComma(names.String()),
		"$subscribe_info", subscribeInfoSurge(ctx, appName),
	)
	return cfg, "application/octet-stream", map[string]string{
		"content-disposition": "attachment;filename*=UTF-8''" + rawurlencode(appName) + ".conf",
	}
}

// BuildSurfboard 生成 Surfboard 配置文件（订阅链接不带域名前缀，与 PHP 一致）。
func BuildSurfboard(ctx *Ctx) (string, string, map[string]string) {
	var proxies, names strings.Builder
	for _, s := range filterServers(ctx, surfboardAllowed, nil) {
		pw := Str(s["password"])
		name := Str(s["name"])
		switch Str(s["type"]) {
		case "shadowsocks":
			if !cipherIn(Str(Get(s, "protocol_settings.cipher")), surfboardSSCiphers...) {
				continue
			}
			proxies.WriteString(buildSurfboardSS(pw, s))
		case "vmess":
			proxies.WriteString(buildSurfboardVmess(pw, s))
		case "trojan":
			proxies.WriteString(buildSurfboardTrojan(pw, s))
		case "anytls":
			proxies.WriteString(buildSurfboardAnyTLS(pw, s))
		default:
			continue
		}
		names.WriteString(name + ", ")
	}
	appName := ctx.AppName
	domain := ctx.Host
	subsURL := ctx.SubscribeURL(ctx.User.Token, "")
	cfg := replaceTemplate(templateOrDefault(ctx, "surfboard", defaultSurfboard),
		"$subs_link", subsURL,
		"$subs_domain", domain,
		"$proxies", proxies.String(),
		"$proxy_group", rtrimComma(names.String()),
		"$subscribe_info", subscribeInfoSurge(ctx, appName),
	)
	return cfg, "text/html; charset=UTF-8", map[string]string{
		"content-disposition": "attachment;filename*=UTF-8''" + rawurlencode(appName) + ".conf",
	}
}
