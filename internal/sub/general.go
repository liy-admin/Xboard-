package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// 对应 App\Protocols\General：输出 base64(URI 列表)，供 v2rayN 等通用客户端使用。

var generalAllowed = map[string]bool{
	"vmess": true, "vless": true, "shadowsocks": true, "trojan": true, "hysteria": true,
	"anytls": true, "socks": true, "tuic": true, "http": true,
}

// generalRequirements: client.type.field -> value -> 最低版本
var generalRequirements = map[string]map[string]map[string]map[string]string{
	"v2rayng": {"hysteria": {"protocol_settings.version": {"2": "1.9.5"}}},
	"v2rayn":  {"hysteria": {"protocol_settings.version": {"2": "6.31"}}},
}

func BuildGeneral(ctx *Ctx) (body string, contentType string, headers map[string]string) {
	var sb strings.Builder
	for _, s := range filterServers(ctx, generalAllowed, generalRequirements) {
		pw := Str(s["password"])
		switch Str(s["type"]) {
		case "vmess":
			sb.WriteString(genVmess(pw, s))
		case "vless":
			sb.WriteString(genVless(pw, s))
		case "shadowsocks":
			sb.WriteString(genSS(pw, s))
		case "trojan":
			sb.WriteString(genTrojan(pw, s))
		case "hysteria":
			sb.WriteString(genHysteria(pw, s))
		case "anytls":
			sb.WriteString(genAnyTLS(pw, s))
		case "socks":
			sb.WriteString(genSocks(pw, s))
		case "tuic":
			sb.WriteString(genTuic(pw, s))
		case "http":
			sb.WriteString(genHTTP(pw, s))
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(sb.String())), "text/plain", map[string]string{
		"subscription-userinfo": userInfo(ctx),
	}
}

func userInfo(ctx *Ctx) string {
	exp := ""
	if ctx.User.ExpiredAt != nil {
		exp = fmt.Sprint(*ctx.User.ExpiredAt)
	}
	return fmt.Sprintf("upload=%d; download=%d; total=%d; expire=%s", ctx.User.U, ctx.User.D, ctx.User.TransferEnab, exp)
}

func ps(s Map) Map {
	if m, ok := s["protocol_settings"].(Map); ok {
		return m
	}
	return Map{}
}

func genSS(password string, s Map) string {
	p := ps(s)
	name := rawurlencode(Str(s["name"]))
	enc := base64.StdEncoding.EncodeToString([]byte(Str(p["cipher"]) + ":" + password))
	enc = strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(enc)
	u := fmt.Sprintf("ss://%s@%s:%s", enc, wrapIPv6(Str(s["host"])), Str(s["port"]))
	if Truthy(p["plugin"]) && Truthy(p["plugin_opts"]) {
		u += "/?plugin=" + rawurlencode(Str(p["plugin"])+";"+Str(p["plugin_opts"]))
	}
	return u + "#" + name + "\r\n"
}

func genVmess(uuid string, s Map) string {
	p := ps(s)
	cfg := Map{
		"v": "2", "ps": s["name"], "add": s["host"], "port": Str(s["port"]), "id": uuid,
		"aid": "0", "net": p["network"], "type": "none", "host": "", "path": "", "tls": "",
	}
	if Truthy(p["tls"]) {
		cfg["tls"] = "tls"
	}
	if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
		cfg["sni"] = sn
	}
	if fp := tlsFingerprint(p["utls"]); fp != "" {
		cfg["fp"] = fp
	}
	ns := func(k string) any { return Get(p, "network_settings."+k) }
	switch Str(p["network"]) {
	case "tcp":
		ht := Str(ns("header.type"))
		if ht == "" {
			ht = "none"
		}
		if ht != "none" {
			cfg["type"] = ht
			cfg["path"] = randPick(ns("header.request.path"), []any{"/"})
			if h := ns("header.request.headers.Host"); Truthy(h) {
				cfg["host"] = randPick(h, []any{"/"})
			} else {
				cfg["host"] = nil
			}
		}
	case "ws":
		cfg["type"] = "ws"
		if v := ns("path"); Truthy(v) {
			cfg["path"] = v
		}
		if v := ns("headers.Host"); Truthy(v) {
			cfg["host"] = v
		}
	case "grpc":
		cfg["type"] = "grpc"
		if v := ns("serviceName"); Truthy(v) {
			cfg["path"] = v
		}
	case "h2":
		cfg["net"], cfg["type"] = "h2", "h2"
		if v := ns("path"); Truthy(v) {
			cfg["path"] = v
		}
		if v := ns("host"); Truthy(v) {
			cfg["host"] = joinHost(v)
		}
	case "httpupgrade":
		cfg["net"], cfg["type"] = "httpupgrade", "httpupgrade"
		if v := ns("path"); Truthy(v) {
			cfg["path"] = v
		}
		cfg["host"] = orDefault(ns("host"), s["host"])
	case "xhttp":
		cfg["net"], cfg["type"] = "xhttp", "xhttp"
		if v := ns("path"); Truthy(v) {
			cfg["path"] = v
		}
		cfg["host"] = orDefault(ns("host"), s["host"])
		cfg["mode"] = orDefault(ns("mode"), "auto")
		if ex, ok := ns("extra").(Map); ok && len(ex) > 0 {
			b, _ := json.Marshal(ex)
			cfg["extra"] = string(b)
		} else if ns("extra") != nil {
			cfg["extra"] = nil
		}
	}
	b, _ := json.Marshal(cfg)
	return "vmess://" + base64.StdEncoding.EncodeToString(b) + "\r\n"
}

func orDefault(v, def any) any {
	if v == nil {
		return def
	}
	return v
}

func joinHost(v any) string {
	if a, ok := v.([]any); ok {
		parts := make([]string, len(a))
		for i, x := range a {
			parts[i] = Str(x)
		}
		return strings.Join(parts, ",")
	}
	return Str(v)
}

// xhttpExtra 返回 JSON 字符串或 nil（对应 `is_array && !empty ? json_encode : null`）。
func xhttpExtra(v any) any {
	if m, ok := v.(Map); ok && len(m) > 0 {
		b, _ := json.Marshal(m)
		return string(b)
	}
	return nil
}

func genVless(uuid string, s Map) string {
	p := ps(s)
	enc := "none"
	if Get(p, "encryption.enabled") == true {
		if e := Str(Get(p, "encryption.encryption")); e != "" {
			enc = e
		}
	}
	q := Query{}
	q.Add("mode", "multi")
	security := ""
	secIdx := len(q)
	q.Add("security", "")
	q.Add("encryption", enc)
	q.Add("type", p["network"])
	q.Add("flow", p["flow"])
	set := func(k string, v any) {
		for i := range q {
			if q[i].k == k {
				q[i].v = Str(v)
				return
			}
		}
		q.Add(k, v)
	}
	switch Int(p["tls"]) {
	case 1:
		security = "tls"
		if fp := tlsFingerprint(p["utls"]); fp != "" {
			q.Add("fp", fp)
		}
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			q.Add("sni", sn)
		}
		if Truthy(Get(p, "tls_settings.allow_insecure")) {
			q.Add("allowInsecure", "1")
		}
	case 2:
		security = "reality"
		q.Add("pbk", Get(p, "reality_settings.public_key"))
		q.Add("sid", Get(p, "reality_settings.short_id"))
		q.Add("sni", Get(p, "reality_settings.server_name"))
		q.Add("servername", Get(p, "reality_settings.server_name"))
		q.Add("spx", "/")
		if fp := tlsFingerprint(p["utls"]); fp != "" {
			q.Add("fp", fp)
		}
	}
	q[secIdx].v = security
	ns := func(k string) any { return Get(p, "network_settings."+k) }
	switch Str(p["network"]) {
	case "ws":
		if v := ns("path"); Truthy(v) {
			set("path", v)
		}
		if v := ns("headers.Host"); Truthy(v) {
			set("host", v)
		}
	case "grpc":
		if v := ns("serviceName"); Truthy(v) {
			set("serviceName", v)
		}
	case "h2":
		set("type", "http")
		if v := ns("path"); Truthy(v) {
			set("path", v)
		}
		if v := ns("host"); Truthy(v) {
			set("host", joinHost(v))
		}
	case "kcp":
		if v := ns("seed"); Truthy(v) {
			set("path", v)
		}
		t := Str(ns("header.type"))
		if t == "" {
			t = "none"
		}
		set("type", t)
	case "httpupgrade":
		if v := ns("path"); Truthy(v) {
			set("path", v)
		}
		set("host", orDefault(ns("host"), s["host"]))
	case "xhttp":
		if v := ns("path"); Truthy(v) {
			set("path", v)
		}
		set("host", orDefault(ns("host"), s["host"]))
		set("mode", orDefault(ns("mode"), "auto"))
		if ex := xhttpExtra(ns("extra")); ex != nil {
			set("extra", ex)
		}
	}
	return fmt.Sprintf("vless://%s@%s:%s?%s#%s\r\n", uuid, wrapIPv6(Str(s["host"])), Str(s["port"]),
		q.Encode(), url.QueryEscape(Str(s["name"])))
}

func genTrojan(password string, s Map) string {
	p := ps(s)
	q := Query{}
	mode := 1
	if p["tls"] != nil {
		mode = Int(p["tls"])
	}
	switch mode {
	case 2:
		q.Add("security", "reality")
		q.Add("pbk", Get(p, "reality_settings.public_key"))
		q.Add("sid", Get(p, "reality_settings.short_id"))
		q.Add("sni", Get(p, "reality_settings.server_name"))
	default:
		q.Add("allowInsecure", Truthy(Get(p, "tls_settings.allow_insecure")))
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			q.Add("peer", sn)
			q.Add("sni", sn)
		}
	}
	if fp := tlsFingerprint(p["utls"]); fp != "" {
		q.Add("fp", fp)
	}
	ns := func(k string) any { return Get(p, "network_settings."+k) }
	switch Str(p["network"]) {
	case "ws":
		q.Add("type", "ws")
		if v := ns("path"); Truthy(v) {
			q.Add("path", v)
		}
		if v := ns("headers.Host"); Truthy(v) {
			q.Add("host", v)
		}
	case "grpc":
		q.Add("type", "grpc")
		if v := ns("serviceName"); Truthy(v) {
			q.Add("serviceName", v)
		}
	case "h2":
		q.Add("type", "http")
		if v := ns("path"); Truthy(v) {
			q.Add("path", v)
		}
		if v := ns("host"); Truthy(v) {
			q.Add("host", joinHost(v))
		}
	case "httpupgrade":
		q.Add("type", "httpupgrade")
		if v := ns("path"); Truthy(v) {
			q.Add("path", v)
		}
		q.Add("host", orDefault(ns("host"), s["host"]))
	case "xhttp":
		q.Add("type", "xhttp")
		if v := ns("path"); Truthy(v) {
			q.Add("path", v)
		}
		q.Add("host", orDefault(ns("host"), s["host"]))
		q.Add("mode", orDefault(ns("mode"), "auto"))
		if ex := xhttpExtra(ns("extra")); ex != nil {
			q.Add("extra", ex)
		}
	}
	return fmt.Sprintf("trojan://%s@%s:%s?%s#%s\r\n", password, wrapIPv6(Str(s["host"])), Str(s["port"]),
		q.Encode(), rawurlencode(Str(s["name"])))
}

func genHysteria(password string, s Map) string {
	p := ps(s)
	q := Query{}
	version := 2
	if p["version"] != nil {
		version = Int(p["version"])
	}
	if sn := Get(p, "tls.server_name"); Truthy(sn) {
		q.Add("sni", sn)
	}
	if Truthy(Get(p, "tls.allow_insecure")) {
		q.Add("insecure", "1")
	} else {
		q.Add("insecure", "0")
	}
	name, addr := rawurlencode(Str(s["name"])), wrapIPv6(Str(s["host"]))
	if version == 2 {
		if Truthy(Get(p, "obfs.open")) {
			q.Add("obfs", "salamander")
			q.Add("obfs-password", Get(p, "obfs.password"))
		}
		if s["ports"] != nil {
			q.Add("mport", s["ports"])
		}
		return fmt.Sprintf("hysteria2://%s@%s:%s?%s#%s\r\n", password, addr, Str(s["port"]), q.Encode(), name)
	}
	q.Add("protocol", "udp")
	q.Add("auth", password)
	if v := Get(p, "bandwidth.up"); Truthy(v) {
		q.Add("upmbps", v)
	}
	if v := Get(p, "bandwidth.down"); Truthy(v) {
		q.Add("downmbps", v)
	}
	if Truthy(Get(p, "obfs.open")) && Truthy(Get(p, "obfs.password")) {
		q.Add("obfs", "xplus")
		q.Add("obfsParam", Get(p, "obfs.password"))
	}
	return fmt.Sprintf("hysteria://%s:%s?%s#%s\r\n", addr, Str(s["port"]), q.Encode(), name)
}

func genTuic(password string, s Map) string {
	p := ps(s)
	q := Query{}
	if v := Get(p, "tls.server_name"); Truthy(v) {
		q.Add("sni", v)
	}
	if v := p["alpn"]; Truthy(v) {
		q.Add("alpn", joinHost(v))
	}
	q.Add("congestion_control", orDefault(p["congestion_control"], "cubic"))
	q.Add("udp-relay-mode", orDefault(p["udp_relay_mode"], "native"))
	if Truthy(Get(p, "tls.allow_insecure")) {
		q.Add("insecure", "1")
	}
	u := fmt.Sprintf("tuic://%s:%s@%s:%s", password, password, wrapIPv6(Str(s["host"])), Str(s["port"]))
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	return u + "#" + rawurlencode(Str(s["name"])) + "\r\n"
}

func genAnyTLS(password string, s Map) string {
	p := ps(s)
	q := Query{}
	q.Add("sni", Get(p, "tls.server_name"))
	q.Add("insecure", Get(p, "tls.allow_insecure"))
	return fmt.Sprintf("anytls://%s@%s:%s?%s#%s\r\n", password, wrapIPv6(Str(s["host"])), Str(s["port"]),
		q.Encode(), rawurlencode(Str(s["name"])))
}

func genSocks(password string, s Map) string {
	cred := base64.StdEncoding.EncodeToString([]byte(password + ":" + password))
	return fmt.Sprintf("socks://%s@%s:%s#%s\r\n", cred, wrapIPv6(Str(s["host"])), Str(s["port"]), rawurlencode(Str(s["name"])))
}

func genHTTP(password string, s Map) string {
	p := ps(s)
	cred := base64.StdEncoding.EncodeToString([]byte(password + ":" + password))
	u := fmt.Sprintf("http://%s@%s:%s", cred, wrapIPv6(Str(s["host"])), Str(s["port"]))
	if Truthy(p["tls"]) {
		q := Query{}
		q.Add("security", "tls")
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			q.Add("sni", sn)
		}
		if Truthy(Get(p, "tls_settings.allow_insecure")) {
			q.Add("allowInsecure", "1")
		} else {
			q.Add("allowInsecure", "0")
		}
		u += "?" + q.Encode()
	}
	return u + "#" + rawurlencode(Str(s["name"])) + "\r\n"
}
