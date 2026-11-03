// Package nodeparse 解析代理分享链接与订阅内容，对应 app/Services/ProxyParserService.php。
package nodeparse

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Server 为解析结果，字段与 PHP 返回的数组一一对应。
type Server struct {
	Type             string         `json:"type"`
	Code             *string        `json:"code"`
	Name             string         `json:"name"`
	Host             string         `json:"host"`
	Port             any            `json:"port"`
	ServerPort       any            `json:"server_port"`
	Rate             int            `json:"rate"`
	ProtocolSettings map[string]any `json:"protocol_settings"`
}

// ParseError 对应解析失败的行信息。
type ParseError struct {
	Line    int    `json:"line,omitempty"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error"`
}

// Result 对应 parseFromText 的返回值。
type Result struct {
	Servers []Server
	Errors  []ParseError
}

var uuidRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ParseText 按行解析，忽略空行与 # 注释；无法识别的行记录到 Errors。
func ParseText(text string) Result {
	var res Result
	for i, raw := range strings.Split(strings.TrimSpace(text), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		s, ok := parseLine(line)
		if !ok {
			r := []rune(line)
			content := line
			if len(r) > 80 {
				content = string(r[:80])
			}
			res.Errors = append(res.Errors, ParseError{Line: i + 1, Content: content, Error: "无法识别的格式"})
			continue
		}
		res.Servers = append(res.Servers, s)
	}
	return res
}

// TryDecodeBase64 订阅内容若是 base64 且解码后包含链接，则返回解码结果。
func TryDecodeBase64(body string) string {
	trimmed := strings.TrimSpace(body)
	if !strings.Contains(trimmed, "://") {
		if dec, err := base64.StdEncoding.DecodeString(trimmed); err == nil && strings.Contains(string(dec), "://") {
			return string(dec)
		}
		if dec, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && strings.Contains(string(dec), "://") {
			return string(dec)
		}
	}
	return body
}

func parseLine(line string) (Server, bool) {
	switch {
	case strings.HasPrefix(line, "vmess://"):
		return parseVmess(line)
	case strings.HasPrefix(line, "vless://"):
		return parseVless(line)
	case strings.HasPrefix(line, "trojan://"):
		return parseTrojan(line)
	case strings.HasPrefix(line, "ss://"):
		return parseSS(line)
	case strings.HasPrefix(line, "hysteria2://"), strings.HasPrefix(line, "hy2://"):
		return parseHysteria2(line)
	case strings.HasPrefix(line, "hysteria://"):
		return parseHysteria(line)
	case strings.HasPrefix(line, "tuic://"):
		return parseTuic(line)
	case strings.HasPrefix(line, "anytls://"):
		return parseAnyTLS(line)
	}
	return Server{}, false
}

// urlParts 解析链接的 host/port/user/fragment/query，失败返回 false。
type urlParts struct {
	host, port, fragment string
	user                 string
	hasUser              bool
	params               url.Values
}

func splitURL(link string) (urlParts, bool) {
	u, err := url.Parse(link)
	if err != nil || u.Host == "" {
		return urlParts{}, false
	}
	p := urlParts{host: u.Hostname(), port: u.Port(), fragment: u.Fragment, params: u.Query()}
	if u.User != nil {
		p.user = u.User.Username()
		p.hasUser = true
	}
	return p, true
}

func portInt(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func nameOr(fragment, host string) string {
	if fragment != "" {
		return fragment
	}
	return host
}

// extractUUID 只保留合法 UUID，与 PHP extractUuid 一致。
func extractUUID(v string) *string {
	if v == "" || !uuidRe.MatchString(v) {
		return nil
	}
	return &v
}

func uuidOf(p urlParts) *string {
	if !p.hasUser {
		return nil
	}
	return extractUUID(p.user)
}

func q(v url.Values, k string) string { return v.Get(k) }

// buildNetworkSettings 对应 buildNetworkSettings；value 为查询参数或 vmess JSON 的字段访问函数。
func buildNetworkSettings(net string, get func(string) string) any {
	switch net {
	case "ws":
		path := get("path")
		if path == "" {
			path = "/"
		}
		return map[string]any{"path": path, "headers": map[string]any{"Host": get("host")}}
	case "h2", "http":
		path := get("path")
		if path == "" {
			path = "/"
		}
		return map[string]any{"path": path, "host": get("host")}
	case "grpc":
		sn := get("serviceName")
		if sn == "" {
			sn = get("path")
		}
		return map[string]any{"serviceName": sn}
	case "tcp":
		typ := get("headerType")
		if typ == "" {
			typ = get("type")
		}
		if typ == "" {
			typ = "none"
		}
		if typ != "http" {
			return nil
		}
		path := get("path")
		if path == "" {
			path = "/"
		}
		return map[string]any{"header": map[string]any{
			"type":    "http",
			"request": map[string]any{"path": []string{path}, "headers": map[string]any{"Host": []string{get("host")}}},
		}}
	}
	return nil
}

func parseVmess(link string) (Server, bool) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
		if err != nil {
			return Server{}, false
		}
	}
	var d map[string]any
	if json.Unmarshal(raw, &d) != nil {
		return Server{}, false
	}
	str := func(k string) string {
		switch v := d[k].(type) {
		case string:
			return v
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		}
		return ""
	}
	add := str("add")
	if add == "" {
		return Server{}, false
	}
	port := portInt(str("port"), 443)
	tls := 0
	if str("tls") == "tls" {
		tls = 1
	}
	net := str("net")
	if net == "" {
		net = "tcp"
	}
	sni := str("sni")
	if sni == "" {
		sni = str("host")
	}
	var code *string
	if id := extractUUID(str("id")); id != nil {
		code = id
	}
	name := str("ps")
	if name == "" {
		name = add
	}
	get := func(k string) string {
		if k == "host" {
			return str("host")
		}
		return str(k)
	}
	var sniPtr any
	if sni != "" {
		sniPtr = sni
	}
	vps := map[string]any{
		"tls": tls, "network": net,
		"network_settings": buildNetworkSettings(net, get),
		"tls_settings":     map[string]any{"server_name": sniPtr, "allow_insecure": false},
	}
	if u := utlsFrom(str("fp"), false); u != nil {
		vps["utls"] = u
	}
	return Server{
		Type: "vmess", Code: code, Name: name, Host: add, Port: port, ServerPort: port, Rate: 1,
		ProtocolSettings: vps,
	}, true
}

func parseVless(link string) (Server, bool) {
	p, ok := splitURL(link)
	if !ok {
		return Server{}, false
	}
	security := orDefault(q(p.params, "security"), "none")
	net := orDefault(q(p.params, "type"), "tcp")
	// 面板的 tls 取值：0 无、1 TLS、2 Reality。PHP 的解析把 Reality 也记成 1，
	// 导致订阅与测速都按普通 TLS 去连 Reality 服务端而握手失败（EOF），这里按 Reality 存为 2。
	tls := 0
	switch security {
	case "tls":
		tls = 1
	case "reality":
		tls = 2
	}
	insecure := q(p.params, "allowInsecure") == "1" || (q(p.params, "allowInsecure") == "" && q(p.params, "insecure") == "1")
	port := portInt(p.port, 443)
	ps := map[string]any{
		"tls": tls, "network": net,
		"network_settings": buildNetworkSettings(net, func(k string) string { return q(p.params, k) }),
		"flow":             nilIfEmpty(q(p.params, "flow")),
		"tls_settings":     map[string]any{"server_name": nilIfEmpty(q(p.params, "sni")), "allow_insecure": insecure},
	}
	if security == "reality" {
		ps["reality_settings"] = map[string]any{
			"public_key": nilIfEmpty(q(p.params, "pbk")), "short_id": nilIfEmpty(q(p.params, "sid")),
			"server_name": nilIfEmpty(q(p.params, "sni")),
		}
	}
	if u := utlsFrom(q(p.params, "fp"), security == "reality"); u != nil {
		ps["utls"] = u
	}
	return Server{Type: "vless", Code: uuidOf(p), Name: nameOr(p.fragment, p.host), Host: p.host, Port: port, ServerPort: port, Rate: 1, ProtocolSettings: ps}, true
}

func parseTrojan(link string) (Server, bool) {
	p, ok := splitURL(link)
	if !ok {
		return Server{}, false
	}
	net := orDefault(q(p.params, "type"), "tcp")
	insecure := q(p.params, "allowInsecure") == "1" || (q(p.params, "allowInsecure") == "" && q(p.params, "insecure") == "1")
	sni := q(p.params, "sni")
	if sni == "" {
		sni = q(p.params, "peer")
	}
	port := portInt(p.port, 443)
	ps := map[string]any{
		"tls": 1, "network": net,
		"network_settings": buildNetworkSettings(net, func(k string) string { return q(p.params, k) }),
		"server_name":      nilIfEmpty(sni), "allow_insecure": insecure,
		"tls_settings": map[string]any{"server_name": nilIfEmpty(sni), "allow_insecure": insecure},
	}
	if u := utlsFrom(q(p.params, "fp"), false); u != nil {
		ps["utls"] = u
	}
	return Server{Type: "trojan", Code: uuidOf(p), Name: nameOr(p.fragment, p.host), Host: p.host, Port: port, ServerPort: port, Rate: 1,
		ProtocolSettings: ps}, true
}

// parseSS 支持 SIP002（userinfo 为 base64 或明文）与整体 base64 两种形式。
func parseSS(link string) (Server, bool) {
	raw := strings.TrimPrefix(link, "ss://")
	fragment := ""
	if i := strings.Index(raw, "#"); i >= 0 {
		fragment, _ = url.PathUnescape(raw[i+1:])
		raw = raw[:i]
	}
	var userinfo, hostport string
	if strings.Contains(raw, "@") {
		at := strings.LastIndex(raw, "@")
		userinfo = raw[:at]
		hostport = raw[at+1:]
		// 去掉 SIP002 的插件查询串 /?plugin=...
		if i := strings.Index(hostport, "/"); i >= 0 {
			hostport = hostport[:i]
		}
		if dec, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(userinfo, "=")); err == nil && strings.Contains(string(dec), ":") {
			userinfo = string(dec)
		} else if dec, err := base64.StdEncoding.DecodeString(userinfo); err == nil && strings.Contains(string(dec), ":") {
			userinfo = string(dec)
		}
	} else {
		dec, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			dec, err = base64.RawStdEncoding.DecodeString(raw)
			if err != nil {
				return Server{}, false
			}
		}
		s := string(dec)
		at := strings.LastIndex(s, "@")
		if at < 0 {
			return Server{}, false
		}
		userinfo = s[:at]
		hostport = s[at+1:]
	}
	method, password, found := strings.Cut(userinfo, ":")
	if !found {
		return Server{}, false
	}
	host, port := splitHostPort(hostport)
	if host == "" {
		return Server{}, false
	}
	var code *string
	code = extractUUID(password)
	pn := portInt(port, 0)
	return Server{Type: "shadowsocks", Code: code, Name: nameOr(fragment, host), Host: host, Port: pn, ServerPort: pn, Rate: 1,
		ProtocolSettings: map[string]any{"cipher": method}}, true
}

func parseHysteria2(link string) (Server, bool) {
	if strings.HasPrefix(link, "hy2://") {
		link = "hysteria2://" + strings.TrimPrefix(link, "hy2://")
	}
	p, ok := splitURL(link)
	if !ok {
		return Server{}, false
	}
	port := p.port
	if mp := q(p.params, "mport"); mp != "" {
		port = mp
	}
	obfsType := q(p.params, "obfs")
	obfsOpen := obfsType != ""
	if obfsType == "" {
		obfsType = "salamander"
	}
	var portVal any = port
	if n, err := strconv.Atoi(port); err == nil {
		portVal = n
	}
	if port == "" {
		portVal = 443
	}
	var obfsPw any
	if v := q(p.params, "obfs-password"); v != "" {
		obfsPw = v
	}
	var sni any
	if v := q(p.params, "sni"); v != "" {
		sni = v
	}
	return Server{Type: "hysteria", Code: uuidOf(p), Name: nameOr(p.fragment, p.host), Host: p.host,
		Port: portVal, ServerPort: portVal, Rate: 1,
		ProtocolSettings: map[string]any{
			"version": 2,
			"obfs":    map[string]any{"open": obfsOpen, "type": obfsType, "password": obfsPw},
			"tls":     map[string]any{"server_name": sni, "allow_insecure": q(p.params, "insecure") == "1"},
		}}, true
}

func parseHysteria(link string) (Server, bool) {
	p, ok := splitURL(link)
	if !ok {
		return Server{}, false
	}
	port := portInt(p.port, 443)
	var up, down any
	if v := q(p.params, "upmbps"); v != "" {
		up = portInt(v, 0)
	}
	if v := q(p.params, "downmbps"); v != "" {
		down = portInt(v, 0)
	}
	var obfsPw any
	if v := q(p.params, "obfs"); v != "" {
		obfsPw = v
	}
	var alpn any
	if v := q(p.params, "alpn"); v != "" {
		alpn = v
	}
	var peer any
	if v := q(p.params, "peer"); v != "" {
		peer = v
	}
	return Server{Type: "hysteria", Name: nameOr(p.fragment, p.host), Host: p.host, Port: port, ServerPort: port, Rate: 1,
		ProtocolSettings: map[string]any{
			"version":   1,
			"bandwidth": map[string]any{"up": up, "down": down},
			"obfs":      map[string]any{"open": obfsPw != nil, "type": "salamander", "password": obfsPw},
			"alpn":      alpn,
			"tls":       map[string]any{"server_name": peer, "allow_insecure": q(p.params, "insecure") == "1"},
		}}, true
}

func parseTuic(link string) (Server, bool) {
	p, ok := splitURL(link)
	if !ok {
		return Server{}, false
	}
	alpn := []string{"h3"}
	if v := q(p.params, "alpn"); v != "" {
		alpn = strings.Split(v, ",")
	}
	cc := orDefault(q(p.params, "congestion_control"), "cubic")
	mode := orDefault(q(p.params, "udp_relay_mode"), "native")
	insecure := q(p.params, "allow_insecure") == "1" || (q(p.params, "allow_insecure") == "" && q(p.params, "insecure") == "1")
	port := portInt(p.port, 443)
	return Server{Type: "tuic", Code: uuidOf(p), Name: nameOr(p.fragment, p.host), Host: p.host, Port: port, ServerPort: port, Rate: 1,
		ProtocolSettings: map[string]any{
			"version": 5, "congestion_control": cc, "alpn": alpn, "udp_relay_mode": mode,
			"tls": map[string]any{"server_name": nilIfEmpty(q(p.params, "sni")), "allow_insecure": insecure},
		}}, true
}

// splitHostPort 对应 splitHostPort（支持 [ipv6]:port）。
func splitHostPort(hp string) (string, string) {
	if strings.HasPrefix(hp, "[") {
		end := strings.Index(hp, "]")
		if end < 0 {
			return "", ""
		}
		return hp[1:end], strings.TrimLeft(hp[end+1:], ":")
	}
	i := strings.LastIndex(hp, ":")
	if i < 0 {
		return hp, ""
	}
	return hp[:i], hp[i+1:]
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func nilIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// parseAnyTLS 解析 anytls://<password|uuid>@host:port?sni=..&insecure=1#name。
// 结构与 hysteria2/tuic 一致：协议设置里的 tls 是对象（server_name / allow_insecure）。
func parseAnyTLS(link string) (Server, bool) {
	p, ok := splitURL(link)
	if !ok {
		return Server{}, false
	}
	insecure := q(p.params, "insecure") == "1" || q(p.params, "allowInsecure") == "1"
	port := portInt(p.port, 443)
	return Server{Type: "anytls", Code: uuidOf(p), Name: nameOr(p.fragment, p.host), Host: p.host,
		Port: port, ServerPort: port, Rate: 1,
		ProtocolSettings: map[string]any{
			"tls":            map[string]any{"server_name": nilIfEmpty(q(p.params, "sni")), "allow_insecure": insecure},
			"padding_scheme": nil,
		}}, true
}

// utlsFrom 把分享链接里的 fp（客户端指纹）转为面板的 utls 设置。
// Reality 必须带 uTLS（sing-box 否则拒绝启动），链接没写 fp 时默认 chrome。
func utlsFrom(fp string, requireForReality bool) any {
	fp = strings.TrimSpace(fp)
	if fp == "" && !requireForReality {
		return nil
	}
	if fp == "" {
		fp = "chrome"
	}
	return map[string]any{"enabled": true, "fingerprint": fp}
}
