package sub

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Shadowrocket 分享链接生成（对应 app/Protocols/Shadowrocket.php）。输出整体 base64，首行为流量状态。

var shadowrocketAllowed = map[string]bool{
	"shadowsocks": true, "vmess": true, "vless": true, "trojan": true, "hysteria": true,
	"tuic": true, "anytls": true, "socks": true,
}

var shadowrocketRequirements = requirements{
	"shadowrocket": {
		"hysteria": {"protocol_settings.version": {"2": "1993"}},
		"trojan":   {"protocol_settings.network": strictNet("tcp", "ws", "grpc", "h2", "httpupgrade")},
	},
}

// stdB64 对应 PHP base64_encode（标准字母表，带填充）。
func stdB64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// safeB64 对应 str_replace(['+','/','='], ['-','_',”], base64_encode(...))。
func safeB64(s string) string {
	return strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(stdB64(s))
}

// firstOrSelf 对应 is_array($v) ? $v[0] : $v（PHP 中字符串取下标得到首字符，这里按整串处理，见 PROGRESS.md）。
func firstOrSelf(v any) any {
	if arr, ok := v.([]any); ok {
		if len(arr) == 0 {
			return nil
		}
		return arr[0]
	}
	return v
}

// randOrDefault 对应 Arr::random($v ?? $def)，v 为数组时随机取一项，标量原样返回。
func randOrDefault(v any, def string) any {
	if v == nil {
		return def
	}
	if arr, ok := v.([]any); ok {
		if len(arr) == 0 {
			return def
		}
		return randPick(arr, nil)
	}
	return v
}

// srTCPHeader 处理 tcp 伪装（header.type 非 none 时），写入 obfs/path/obfsParam。
func srTCPHeader(q *orderedQuery, s Map) {
	ps := "protocol_settings.network_settings.header"
	if Str(Get(s, ps+".type")) == "" {
		// data_get 默认 'none'，缺失时视为 none
		return
	}
	t := Str(Get(s, ps+".type"))
	if t == "none" {
		return
	}
	q.set("obfs", t)
	q.set("path", randOrDefault(Get(s, ps+".request.path"), "/"))
	q.set("obfsParam", randOrDefault(Get(s, ps+".request.headers.Host"), "www.example.com"))
}

// srServerHost 返回 data_get($settings, $path, $server['host']) 的语义：键缺失或为 null 时回退服务器地址。
func srServerHost(s Map, path string) any {
	if v := Get(s, path); v != nil {
		return v
	}
	return s["host"]
}

// srUserinfo 对应 base64_encode('auto:' . $uuid . '@' . host . ':' . port)。
func srUserinfo(uuid string, s Map) string {
	return stdB64("auto:" + uuid + "@" + wrapIPv6(Str(s["host"])) + ":" + Str(s["port"]))
}

func srShadowsocks(pw string, s Map) string {
	name := rawurlencode(Str(s["name"]))
	str := safeB64(Str(Get(s, "protocol_settings.cipher")) + ":" + pw)
	uri := fmt.Sprintf("ss://%s@%s:%s", str, wrapIPv6(Str(s["host"])), Str(s["port"]))
	plugin := Str(Get(s, "protocol_settings.plugin"))
	if plugin == "obfs" {
		plugin = "obfs-local"
	}
	opts := Str(Get(s, "protocol_settings.plugin_opts"))
	if plugin != "" && opts != "" {
		uri += "/?plugin=" + plugin + ";" + rawurlencode(opts)
	}
	return uri + "#" + name + "\r\n"
}

func srVmess(uuid string, s Map) string {
	q := newQuery()
	q.set("tfo", 1).set("remark", Str(s["name"])).set("alterId", 0)
	ps := "protocol_settings"
	if Truthy(Get(s, ps+".tls")) {
		q.set("tls", 1)
		if Truthy(Get(s, ps+".tls_settings")) {
			if Truthy(Get(s, ps+".tls_settings.allow_insecure")) {
				q.set("allowInsecure", intFlag(Get(s, ps+".tls_settings.allow_insecure")))
			}
			if Truthy(Get(s, ps+".tls_settings.server_name")) {
				q.set("peer", Get(s, ps+".tls_settings.server_name"))
			}
		}
	}
	switch Str(Get(s, ps+".network")) {
	case "tcp":
		srTCPHeader(q, s)
	case "ws":
		q.set("obfs", "websocket")
		q.set("path", Get(s, ps+".network_settings.path"))
		if h := Get(s, ps+".network_settings.headers.Host"); Truthy(h) {
			q.set("obfsParam", h)
		}
	case "grpc":
		q.set("obfs", "grpc")
		q.set("path", Get(s, ps+".network_settings.serviceName"))
		q.set("host", srServerHostOr(Get(s, ps+".tls_settings.server_name"), s))
	case "httpupgrade":
		q.set("obfs", "httpupgrade")
		if p := Get(s, ps+".network_settings.path"); Truthy(p) {
			q.set("path", p)
		}
		if h := srServerHost(s, ps+".network_settings.host"); Truthy(h) {
			q.set("obfsParam", h)
		}
	case "h2":
		q.set("obfs", "h2")
		if p := Get(s, ps+".network_settings.path"); Truthy(p) {
			q.set("path", p)
		}
		if h := Get(s, ps+".network_settings.host"); Truthy(h) {
			v := firstOrDefault(h, s["host"])
			q.set("obfsParam", v).set("peer", v)
		}
	case "xhttp":
		srXHTTP(q, s)
	}
	return "vmess://" + srUserinfo(uuid, s) + "?" + q.encode(true) + "\r\n"
}

// srServerHostOr 对应 $tls_settings.server_name ?? $server['host']（仅 null 时回退）。
func srServerHostOr(v any, s Map) any {
	if v != nil {
		return v
	}
	return s["host"]
}

// firstOrDefault 对应 $host[0] ?? $default。
func firstOrDefault(h any, def any) any {
	if arr, ok := h.([]any); ok {
		if len(arr) == 0 {
			return def
		}
		return arr[0]
	}
	return h
}

func srXHTTP(q *orderedQuery, s Map) {
	ps := "protocol_settings.network_settings"
	q.set("obfs", "xhttp")
	if p := Get(s, ps+".path"); Truthy(p) {
		q.set("path", p)
	}
	if h := srServerHost(s, ps+".host"); Truthy(h) {
		q.set("obfsParam", h)
	}
	mode := Get(s, ps+".mode")
	if mode == nil {
		mode = "auto"
	}
	if Truthy(mode) {
		q.set("mode", mode)
	}
}

func srVless(uuid string, s Map) string {
	q := newQuery()
	q.set("tfo", 1).set("remark", Str(s["name"]))
	ps := "protocol_settings"
	xtls := map[string]int{"none": 0, "xtls-rprx-direct": 1, "xtls-rprx-vision": 2}
	if flow := Str(Get(s, ps+".flow")); flow != "" {
		if v, ok := xtls[flow]; ok {
			q.set("tls", 1).set("xtls", v)
		}
	}
	switch Int(Get(s, ps+".tls")) {
	case 1:
		q.set("tls", 1)
		q.set("allowInsecure", intFlag(Get(s, ps+".tls_settings.allow_insecure")))
		if sn := Get(s, ps+".tls_settings.server_name"); Truthy(sn) {
			q.set("peer", sn)
		}
		if fp := tlsFingerprint(Get(s, ps+".utls")); fp != "" {
			q.set("fp", fp)
		}
	case 2:
		q.set("tls", 1)
		q.set("sni", Get(s, ps+".reality_settings.server_name"))
		q.set("pbk", Get(s, ps+".reality_settings.public_key"))
		q.set("sid", Get(s, ps+".reality_settings.short_id"))
		if fp := tlsFingerprint(Get(s, ps+".utls")); fp != "" {
			q.set("fp", fp)
		}
	}
	switch Str(Get(s, ps+".network")) {
	case "tcp":
		srTCPHeader(q, s)
	case "ws":
		q.set("obfs", "websocket")
		if p := Get(s, ps+".network_settings.path"); Truthy(p) {
			q.set("path", p)
		}
		if h := Get(s, ps+".network_settings.headers.Host"); Truthy(h) {
			q.set("obfsParam", h)
		}
	case "grpc":
		q.set("obfs", "grpc")
		q.set("path", Get(s, ps+".network_settings.serviceName"))
		q.set("host", srServerHostOr(Get(s, ps+".tls_settings.server_name"), s))
	case "kcp":
		q.set("obfs", "kcp")
		if seed := Get(s, ps+".network_settings.seed"); Truthy(seed) {
			q.set("path", seed)
		}
		typ := Get(s, ps+".network_settings.header.type")
		if typ == nil {
			typ = "none"
		}
		q.set("type", typ)
	case "h2":
		q.set("obfs", "h2")
		if p := Get(s, ps+".network_settings.path"); Truthy(p) {
			q.set("path", p)
		}
		if h := srServerHost(s, ps+".network_settings.host"); Truthy(h) {
			q.set("obfsParam", h)
		}
	case "httpupgrade":
		q.set("obfs", "httpupgrade")
		if p := Get(s, ps+".network_settings.path"); Truthy(p) {
			q.set("path", p)
		}
		if h := srServerHost(s, ps+".network_settings.host"); Truthy(h) {
			q.set("obfsParam", h)
		}
	case "xhttp":
		srXHTTP(q, s)
	}
	return "vless://" + srUserinfo(uuid, s) + "?" + q.encode(true) + "\r\n"
}

func srTrojan(pw string, s Map) string {
	name := rawurlencode(Str(s["name"]))
	ps := "protocol_settings"
	q := newQuery()
	mode := Int(Get(s, ps+".tls"))
	if Get(s, ps+".tls") == nil {
		mode = 1
	}
	if mode == 2 {
		q.set("security", "reality")
		q.set("pbk", Get(s, ps+".reality_settings.public_key"))
		q.set("sid", Get(s, ps+".reality_settings.short_id"))
		q.set("sni", Get(s, ps+".reality_settings.server_name"))
	} else {
		q.set("allowInsecure", intFlag(Get(s, ps+".tls_settings.allow_insecure")))
		if sn := Get(s, ps+".tls_settings.server_name"); Truthy(sn) {
			q.set("peer", sn)
		}
	}
	switch Str(Get(s, ps+".network")) {
	case "grpc":
		q.set("obfs", "grpc")
		q.set("path", Get(s, ps+".network_settings.serviceName"))
	case "ws":
		host := Get(s, ps+".network_settings.headers.Host")
		path := Get(s, ps+".network_settings.path")
		q.set("plugin", fmt.Sprintf("obfs-local;obfs=websocket;obfs-host=%s;obfs-uri=%s", srStr(host), srStr(path)))
	case "h2":
		q.set("obfs", "h2")
		if p := Get(s, ps+".network_settings.path"); Truthy(p) {
			q.set("path", p)
		}
		if h := srServerHost(s, ps+".network_settings.host"); Truthy(h) {
			q.set("obfsParam", firstOrSelf(h))
		}
	case "httpupgrade":
		q.set("obfs", "httpupgrade")
		if p := Get(s, ps+".network_settings.path"); Truthy(p) {
			q.set("path", p)
		}
		if h := srServerHost(s, ps+".network_settings.host"); Truthy(h) {
			q.set("obfsParam", h)
		}
	case "xhttp":
		srXHTTP(q, s)
	}
	addr := wrapIPv6(Str(s["host"]))
	return fmt.Sprintf("trojan://%s@%s:%s?%s&tfo=1#%s\r\n", pw, addr, Str(s["port"]), q.encode(false), name)
}

// srStr 把可能为 nil 的值转为字符串（插值时 null 为空串）。
func srStr(v any) string { return Str(v) }

func srHysteria(pw string, s Map) string {
	ps := "protocol_settings"
	switch Int(Get(s, ps+".version")) {
	case 1:
		q := newQuery()
		q.set("auth", pw)
		q.set("upmbps", Get(s, ps+".bandwidth.up"))
		q.set("downmbps", Get(s, ps+".bandwidth.down"))
		q.set("protocol", "udp")
		q.set("fastopen", 1)
		if sn := Get(s, ps+".tls.server_name"); Truthy(sn) {
			q.set("peer", sn)
		}
		if Truthy(Get(s, ps+".obfs.open")) {
			q.set("obfs", "xplus")
			q.set("obfsParam", Get(s, ps+".obfs.password"))
		}
		q.set("insecure", Get(s, ps+".tls.allow_insecure"))
		if p, ok := s["ports"]; ok && p != nil {
			q.set("mport", p)
		}
		return fmt.Sprintf("hysteria://%s:%s?%s#%s\r\n", wrapIPv6(Str(s["host"])), Str(s["port"]), q.encode(false), Str(s["name"]))
	case 2:
		q := newQuery()
		q.set("obfs", "none")
		q.set("fastopen", 1)
		if sn := Get(s, ps+".tls.server_name"); Truthy(sn) {
			q.set("peer", sn)
		}
		if Truthy(Get(s, ps+".obfs.open")) {
			q.set("obfs", Get(s, ps+".obfs.type"))
			q.set("obfs-password", Get(s, ps+".obfs.password"))
		}
		q.set("insecure", Get(s, ps+".tls.allow_insecure"))
		if hop := Get(s, ps+".hop_interval"); hop != nil {
			q.set("keepalive", hop)
		}
		if p, ok := s["ports"]; ok && p != nil {
			q.set("mport", p)
		}
		return fmt.Sprintf("hysteria2://%s@%s:%s?%s#%s\r\n", pw, wrapIPv6(Str(s["host"])), Str(s["port"]), q.encode(false), Str(s["name"]))
	}
	return ""
}

func srTUIC(pw string, s Map) string {
	ps := "protocol_settings"
	name := rawurlencode(Str(s["name"]))
	q := newQuery()
	q.set("alpn", Get(s, ps+".alpn"))
	q.set("sni", Get(s, ps+".tls.server_name"))
	q.set("insecure", Get(s, ps+".tls.allow_insecure"))
	cc := Get(s, ps+".congestion_control")
	if cc == nil {
		cc = "cubic"
	}
	q.set("congestion_control", cc)
	if Str(Get(s, ps+".version")) == "4" {
		q.set("token", pw)
	} else {
		q.set("uuid", pw)
		q.set("password", pw)
	}
	return fmt.Sprintf("tuic://%s:%s?%s#%s\r\n", wrapIPv6(Str(s["host"])), Str(s["port"]), q.encode(false), name)
}

func srAnyTLS(pw string, s Map) string {
	ps := "protocol_settings"
	name := rawurlencode(Str(s["name"]))
	q := newQuery()
	q.set("sni", Get(s, ps+".tls.server_name"))
	q.set("insecure", Get(s, ps+".tls.allow_insecure"))
	return fmt.Sprintf("anytls://%s@%s:%s?%s#%s\r\n", pw, wrapIPv6(Str(s["host"])), Str(s["port"]), q.encode(false), name)
}

func srSocks(pw string, s Map) string {
	name := rawurlencode(Str(s["name"]))
	addr := wrapIPv6(Str(s["host"]))
	return "socks://" + stdB64(pw+":"+pw+"@"+addr+":"+Str(s["port"])) + "?method=auto#" + name + "\r\n"
}

// BuildShadowrocket 生成 Shadowrocket 订阅（整体 base64），首行为流量与到期状态。
func BuildShadowrocket(ctx *Ctx) (string, string, map[string]string) {
	u := ctx.User
	expired := "N/A"
	if u.ExpiredAt != nil {
		expired = fmtDate(*u.ExpiredAt)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("STATUS=🚀↑:%sGB,↓:%sGB,TOT:%sGB💡Expires:%s\r\n",
		phpFloat(trafficGB(u.U)), phpFloat(trafficGB(u.D)), phpFloat(trafficGB(u.TransferEnab)), expired))
	for _, s := range filterServers(ctx, shadowrocketAllowed, shadowrocketRequirements) {
		pw := Str(s["password"])
		switch Str(s["type"]) {
		case "shadowsocks":
			sb.WriteString(srShadowsocks(pw, s))
		case "vmess":
			sb.WriteString(srVmess(pw, s))
		case "vless":
			sb.WriteString(srVless(pw, s))
		case "trojan":
			sb.WriteString(srTrojan(pw, s))
		case "hysteria":
			sb.WriteString(srHysteria(pw, s))
		case "tuic":
			sb.WriteString(srTUIC(pw, s))
		case "anytls":
			sb.WriteString(srAnyTLS(pw, s))
		case "socks":
			sb.WriteString(srSocks(pw, s))
		}
	}
	return stdB64(sb.String()), "text/plain", nil
}
