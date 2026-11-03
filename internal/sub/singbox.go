package sub

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

//go:embed default.sing-box.json
var defaultSingBox []byte

var singboxAllowed = map[string]bool{
	"shadowsocks": true, "trojan": true, "vmess": true, "vless": true, "hysteria": true,
	"tuic": true, "anytls": true, "socks": true, "http": true,
}

var echRule = map[string]string{"1": "1.5.0"}

var singboxRequirements = requirements{
	"sing-box": {
		"vless": {
			"protocol_settings.flow":                     {"xtls-rprx-vision": "1.5.0"},
			"protocol_settings.tls":                      {"2": "1.6.0"},
			"protocol_settings.tls_settings.ech.enabled": echRule,
			"protocol_settings.network":                  {"xhttp": "9999.0.0"},
		},
		"vmess": {
			"protocol_settings.tls_settings.ech.enabled": echRule,
			"protocol_settings.network":                  {"xhttp": "9999.0.0"},
		},
		"trojan": {
			"protocol_settings.tls_settings.ech.enabled": echRule,
			"protocol_settings.network":                  {"xhttp": "9999.0.0"},
		},
		"hysteria": {"protocol_settings.version": {"2": "1.5.0"}, "protocol_settings.tls.ech.enabled": echRule},
		"tuic":     {"protocol_settings.tls.ech.enabled": echRule},
		"anytls":   {"protocol_settings.tls.ech.enabled": {"1": "1.12.0"}},
		"socks":    {"protocol_settings.tls_settings.ech.enabled": echRule},
		"http":     {"protocol_settings.tls_settings.ech.enabled": echRule},
	},
}

// BuildSingBox 对应 App\Protocols\SingBox::handle。
func BuildSingBox(ctx *Ctx) (any, map[string]string, error) {
	raw := ""
	if ctx.Template != nil {
		raw = ctx.Template("singbox")
	}
	var cfg Map
	data := []byte(raw)
	if strings.TrimSpace(raw) == "" {
		data = defaultSingBox
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("singbox template: %w", err)
	}
	b := &sbBuilder{ctx: ctx, cfg: cfg}
	servers := filterServers(ctx, singboxAllowed, singboxRequirements)
	b.buildOutbounds(servers)
	b.adaptForVersion()
	b.ensureDefaultDomainResolver()
	hdr := map[string]string{
		"profile-title":           "base64:" + b64(ctx.AppName),
		"subscription-userinfo":   userInfo(ctx),
		"profile-update-interval": "24",
	}
	return b.cfg, hdr, nil
}

type sbBuilder struct {
	ctx *Ctx
	cfg Map
}

// outboundFor 把一个节点转换为 sing-box outbound；协议不支持时返回 nil。
func (b *sbBuilder) outboundFor(s Map) Map {
	pw := Str(s["password"])
	p := ps(s)
	switch Str(s["type"]) {
	case "shadowsocks":
		return b.ss(pw, s)
	case "trojan":
		return b.trojan(pw, s)
	case "vmess":
		return b.vmess(pw, s)
	case "vless":
		switch Str(p["network"]) {
		case "tcp", "ws", "grpc", "http", "quic", "httpupgrade":
			return b.vless(pw, s)
		}
	case "hysteria":
		return b.hysteria(pw, s)
	case "tuic":
		return b.tuic(pw, s)
	case "anytls":
		return b.anytls(pw, s)
	case "socks":
		return b.socks(pw, s)
	case "http":
		return b.http(pw, s)
	}
	return nil
}

func (b *sbBuilder) buildOutbounds(servers []Map) {
	outbounds, _ := b.cfg["outbounds"].([]any)
	var proxies []any
	for _, s := range servers {
		if o := b.outboundFor(s); o != nil {
			proxies = append(proxies, o)
		}
	}
	var allTags []string
	for _, p := range proxies {
		allTags = append(allTags, Str(p.(Map)["tag"]))
	}
	for _, o := range outbounds {
		ob, ok := o.(Map)
		if !ok {
			continue
		}
		t := Str(ob["type"])
		if t != "urltest" && t != "selector" {
			continue
		}
		include, exclude, fallback := ob["include"], ob["exclude"], ob["fallback"]
		delete(ob, "include")
		delete(ob, "exclude")
		delete(ob, "fallback")
		tags := append([]string(nil), allTags...)
		if s := Str(include); s != "" {
			tags = filterTags(tags, s, true)
		}
		if s := Str(exclude); s != "" {
			tags = filterTags(tags, s, false)
		}
		if len(tags) == 0 && fallback != nil {
			tags = resolveFallback(fallback, allTags, outbounds)
		}
		if len(tags) > 0 {
			arr, _ := ob["outbounds"].([]any)
			for _, t := range tags {
				arr = append(arr, t)
			}
			ob["outbounds"] = arr
		}
	}
	b.cfg["outbounds"] = append(outbounds, proxies...)
}

var patCache = map[string]*regexp.Regexp{}

// matchesPattern 复刻 PHP 的 `~pattern~ui` 与带分隔符模式；非法模式返回 false。
func matchesPattern(pattern, subject string) bool {
	re, ok := patCache[pattern]
	if !ok {
		t := strings.TrimSpace(pattern)
		src, flags := t, "i"
		if len(t) >= 2 && strings.ContainsRune("/#~@%", rune(t[0])) {
			if i := strings.LastIndex(t, string(t[0])); i > 0 {
				src = t[1:i]
				flags = ""
				for _, f := range t[i+1:] {
					if f == 'i' || f == 'm' || f == 's' {
						flags += string(f)
					}
				}
			}
		}
		if flags != "" {
			src = "(?" + flags + ")" + src
		}
		re, _ = regexp.Compile(src)
		patCache[pattern] = re
	}
	return re != nil && re.MatchString(subject)
}

func filterTags(tags []string, pat string, keep bool) []string {
	var out []string
	for _, t := range tags {
		if matchesPattern(pat, t) == keep {
			out = append(out, t)
		}
	}
	return out
}

func resolveFallback(fb any, allTags []string, outbounds []any) []string {
	var cands []any
	if a, ok := fb.([]any); ok {
		cands = a
	} else {
		cands = []any{fb}
	}
	templ := map[string]bool{}
	for _, o := range outbounds {
		if m, ok := o.(Map); ok {
			templ[Str(m["tag"])] = true
		}
	}
	for _, c := range cands {
		s, ok := c.(string)
		if !ok || s == "" {
			continue
		}
		for _, t := range allTags {
			if t == s {
				return []string{s}
			}
		}
		if templ[s] {
			return []string{s}
		}
		if m := filterTags(allTags, s, true); len(m) > 0 {
			return m
		}
	}
	return nil
}

func (b *sbBuilder) supports(name, min string) bool {
	return b.ctx.ClientName == name && b.ctx.ClientVersion != "" && versionCompare(b.ctx.ClientVersion, min) >= 0
}

var singboxUA = regexp.MustCompile(`(?i)sing-box\s+v?(\d+(?:\.\d+){0,2})`)

func (b *sbBuilder) coreVersion() string {
	if m := singboxUA.FindStringSubmatch(b.ctx.UserAgent); m != nil {
		return m[1]
	}
	if b.ctx.ClientVersion == "" {
		return ""
	}
	if b.ctx.ClientName == "sing-box" {
		return b.ctx.ClientVersion
	}
	return "1.13.0"
}

func (b *sbBuilder) adaptForVersion() {
	v := b.coreVersion()
	if v == "" {
		return
	}
	if versionCompare(v, "1.13.0") >= 0 {
		b.upgradeSpecial()
	}
	if versionCompare(v, "1.11.0") < 0 {
		b.downgradeActions()
		b.restoreInbound()
	}
	if versionCompare(v, "1.12.0") < 0 {
		b.dnsLegacy()
	}
	if versionCompare(v, "1.10.0") < 0 {
		b.tunLegacy()
	}
}

func (b *sbBuilder) rules() []any {
	r, _ := Get(b.cfg, "route.rules").([]any)
	return r
}

func (b *sbBuilder) upgradeSpecial() {
	removed := map[string]string{}
	var keep []any
	for _, o := range b.cfg["outbounds"].([]any) {
		m, _ := o.(Map)
		if t := Str(m["type"]); t == "block" || t == "dns" {
			removed[Str(m["tag"])] = t
			continue
		}
		keep = append(keep, o)
	}
	b.cfg["outbounds"] = keep
	if len(removed) == 0 {
		return
	}
	for _, r := range b.rules() {
		m, _ := r.(Map)
		t, ok := removed[Str(m["outbound"])]
		if m["outbound"] == nil || !ok {
			continue
		}
		delete(m, "outbound")
		if t == "dns" {
			m["action"] = "hijack-dns"
		} else {
			m["action"] = "reject"
		}
	}
}

func (b *sbBuilder) downgradeActions() {
	needDNS, needBlock := false, false
	for _, r := range b.rules() {
		m, _ := r.(Map)
		switch Str(m["action"]) {
		case "hijack-dns":
			delete(m, "action")
			m["outbound"] = "dns-out"
			needDNS = true
		case "reject":
			delete(m, "action")
			m["outbound"] = "block"
			needBlock = true
		}
	}
	obs, _ := b.cfg["outbounds"].([]any)
	if needBlock {
		obs = append(obs, Map{"type": "block", "tag": "block"})
	}
	if needDNS {
		obs = append(obs, Map{"type": "dns", "tag": "dns-out"})
	}
	b.cfg["outbounds"] = obs
}

func (b *sbBuilder) restoreInbound() {
	ins, _ := b.cfg["inbounds"].([]any)
	for _, i := range ins {
		m, _ := i.(Map)
		if Str(m["type"]) == "tun" {
			m["endpoint_independent_nat"] = true
		}
		if Truthy(m["sniff"]) {
			m["sniff_override_destination"] = true
		}
	}
}

func (b *sbBuilder) dnsLegacy() {
	servers, _ := Get(b.cfg, "dns.servers").([]any)
	for _, s := range servers {
		m, _ := s.(Map)
		t, has := m["type"]
		if !has {
			continue
		}
		host := Str(m["server"])
		switch Str(t) {
		case "https":
			m["address"] = "https://" + host + "/dns-query"
		case "tls":
			m["address"] = "tls://" + host
		case "tcp":
			m["address"] = "tcp://" + host
		case "quic":
			m["address"] = "quic://" + host
		case "block":
			m["address"] = "rcode://refused"
		case "rcode":
			rc := Str(m["rcode"])
			if rc == "" {
				rc = "success"
			}
			m["address"] = "rcode://" + rc
			delete(m, "rcode")
		default:
			m["address"] = host
		}
		delete(m, "type")
		delete(m, "server")
	}
}

func (b *sbBuilder) tunLegacy() {
	ins, _ := b.cfg["inbounds"].([]any)
	for _, i := range ins {
		m, _ := i.(Map)
		addrs, ok := m["address"].([]any)
		if Str(m["type"]) != "tun" || !ok {
			continue
		}
		for _, a := range addrs {
			if strings.Contains(Str(a), ":") {
				m["inet6_address"] = a
			} else {
				m["inet4_address"] = a
			}
		}
		delete(m, "address")
	}
}

// ---- outbound builders ----

func (b *sbBuilder) ss(pw string, s Map) Map {
	p := ps(s)
	m := Map{"tag": s["name"], "type": "shadowsocks", "server": s["host"], "server_port": s["port"],
		"method": p["cipher"], "password": pw}
	if Truthy(p["plugin"]) && Truthy(p["plugin_opts"]) {
		m["plugin"], m["plugin_opts"] = p["plugin"], p["plugin_opts"]
	}
	return m
}

func (b *sbBuilder) vmess(uuid string, s Map) Map {
	p := ps(s)
	m := Map{"tag": s["name"], "type": "vmess", "server": s["host"], "server_port": s["port"],
		"uuid": uuid, "security": "auto", "alter_id": 0}
	if Truthy(p["tls"]) {
		tls := Map{"enabled": true, "insecure": Truthy(Get(p, "tls_settings.allow_insecure"))}
		appendUtls(tls, p)
		appendEch(tls, Get(p, "tls_settings.ech"))
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			tls["server_name"] = sn
		}
		m["tls"] = tls
	}
	appendMultiplex(m, p)
	if t := transport(p, s); t != nil {
		m["transport"] = t
	}
	return m
}

func (b *sbBuilder) vless(uuid string, s Map) Map {
	p := ps(s)
	m := Map{"type": "vless", "tag": s["name"], "server": s["host"], "server_port": s["port"],
		"uuid": uuid, "packet_encoding": "xudp"}
	if f := p["flow"]; Truthy(f) {
		m["flow"] = f
	}
	if Truthy(p["tls"]) {
		mode := Int(p["tls"])
		insecure := Truthy(Get(p, "tls_settings.allow_insecure"))
		if mode == 2 {
			insecure = Truthy(Get(p, "reality_settings.allow_insecure"))
		}
		tls := Map{"enabled": true, "insecure": insecure}
		appendUtls(tls, p)
		switch mode {
		case 1:
			appendEch(tls, Get(p, "tls_settings.ech"))
			if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
				tls["server_name"] = sn
			}
		case 2:
			tls["server_name"] = Get(p, "reality_settings.server_name")
			tls["reality"] = Map{"enabled": true, "public_key": Get(p, "reality_settings.public_key"),
				"short_id": Get(p, "reality_settings.short_id")}
		}
		m["tls"] = tls
	}
	appendMultiplex(m, p)
	if t := transport(p, s); t != nil {
		m["transport"] = t
	}
	return m
}

func (b *sbBuilder) trojan(pw string, s Map) Map {
	p := ps(s)
	m := Map{"tag": s["name"], "type": "trojan", "server": s["host"], "server_port": s["port"], "password": pw}
	mode := 1
	if p["tls"] != nil {
		mode = Int(p["tls"])
	}
	tls := Map{"enabled": true}
	if mode == 2 {
		tls["insecure"] = Truthy(Get(p, "reality_settings.allow_insecure"))
		tls["server_name"] = Get(p, "reality_settings.server_name")
		tls["reality"] = Map{"enabled": true, "public_key": Get(p, "reality_settings.public_key"),
			"short_id": Get(p, "reality_settings.short_id")}
	} else {
		tls["insecure"] = Truthy(Get(p, "tls_settings.allow_insecure"))
		appendEch(tls, Get(p, "tls_settings.ech"))
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			tls["server_name"] = sn
		}
	}
	appendUtls(tls, p)
	m["tls"] = tls
	appendMultiplex(m, p)
	if t := transport(p, s); t != nil {
		m["transport"] = t
	}
	return m
}

func (b *sbBuilder) hysteria(pw string, s Map) Map {
	p := ps(s)
	tls := Map{"enabled": true, "insecure": Truthy(Get(p, "tls.allow_insecure"))}
	m := Map{"server": s["host"], "server_port": s["port"], "tag": s["name"], "tls": tls}
	if b.supports("sing-box", "1.11.0") {
		if s["ports"] != nil {
			m["server_ports"] = []any{strings.ReplaceAll(Str(s["ports"]), "-", ":")}
		}
		if p["hop_interval"] != nil {
			m["hop_interval"] = Str(p["hop_interval"]) + "s"
		}
	}
	if sn := Get(p, "tls.server_name"); Truthy(sn) {
		tls["server_name"] = sn
	}
	appendEch(tls, Get(p, "tls.ech"))
	m["up_mbps"], m["down_mbps"] = Get(p, "bandwidth.up"), Get(p, "bandwidth.down")
	version := 1
	if p["version"] != nil {
		version = Int(p["version"])
	}
	if version == 2 {
		m["type"], m["password"] = "hysteria2", pw
		if Truthy(Get(p, "obfs.open")) {
			m["obfs"] = Map{"type": Get(p, "obfs.type"), "password": Get(p, "obfs.password")}
		}
	} else {
		m["type"], m["auth_str"], m["obfs"] = "hysteria", pw, Get(p, "obfs.password")
		m["disable_mtu_discovery"] = true
	}
	return compact(m)
}

func (b *sbBuilder) tuic(pw string, s Map) Map {
	p := ps(s)
	alpn := p["alpn"]
	if alpn == nil {
		alpn = []any{"h3"}
	}
	tls := Map{"enabled": true, "insecure": Truthy(Get(p, "tls.allow_insecure")), "alpn": alpn}
	m := Map{"type": "tuic", "tag": s["name"], "server": s["host"], "server_port": s["port"],
		"congestion_control": orDefault(p["congestion_control"], "cubic"),
		"udp_relay_mode":     orDefault(p["udp_relay_mode"], "native"),
		"zero_rtt_handshake": true, "heartbeat": "10s", "tls": tls}
	if sn := Get(p, "tls.server_name"); Truthy(sn) {
		tls["server_name"] = sn
	}
	appendEch(tls, Get(p, "tls.ech"))
	if Int(p["version"]) == 4 {
		m["token"] = pw
	} else {
		m["uuid"], m["password"] = pw, pw
	}
	return m
}

func (b *sbBuilder) anytls(pw string, s Map) Map {
	p := ps(s)
	alpn := p["alpn"]
	if alpn == nil {
		alpn = []any{"h3"}
	}
	tls := Map{"enabled": true, "insecure": Truthy(Get(p, "tls.allow_insecure")), "alpn": alpn}
	m := Map{"type": "anytls", "tag": s["name"], "server": s["host"], "password": pw,
		"server_port": s["port"], "tls": tls}
	if sn := Get(p, "tls.server_name"); Truthy(sn) {
		tls["server_name"] = sn
	}
	appendEch(tls, Get(p, "tls.ech"))
	return m
}

func (b *sbBuilder) socks(pw string, s Map) Map {
	p := ps(s)
	m := Map{"type": "socks", "tag": s["name"], "server": s["host"], "server_port": s["port"],
		"version": "5", "username": pw, "password": pw}
	if Truthy(p["udp_over_tcp"]) {
		m["udp_over_tcp"] = true
	}
	return m
}

func (b *sbBuilder) http(pw string, s Map) Map {
	p := ps(s)
	m := Map{"type": "http", "tag": s["name"], "server": s["host"], "server_port": s["port"],
		"username": pw, "password": pw}
	if v := p["path"]; Truthy(v) {
		m["path"] = v
	}
	if v := p["headers"]; Truthy(v) {
		m["headers"] = v
	}
	if Truthy(p["tls"]) {
		tls := Map{"enabled": true, "insecure": Truthy(Get(p, "tls_settings.allow_insecure"))}
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			tls["server_name"] = sn
		}
		appendEch(tls, Get(p, "tls_settings.ech"))
		m["tls"] = tls
	}
	return m
}

func transport(p Map, s Map) Map {
	ns := func(k string) any { return Get(p, "network_settings."+k) }
	var t Map
	switch Str(p["network"]) {
	case "tcp":
		if Str(ns("header.type")) == "http" {
			t = Map{"type": "http", "path": randPick(ns("header.request.path"), []any{"/"}),
				"host": orDefault(ns("header.request.headers.Host"), []any{})}
		}
	case "ws":
		t = Map{"type": "ws", "path": ns("path"), "max_early_data": 0}
		if h := ns("headers.Host"); Truthy(h) {
			t["headers"] = Map{"Host": h}
		}
	case "grpc":
		t = Map{"type": "grpc", "service_name": ns("serviceName")}
	case "h2":
		t = Map{"type": "http", "host": ns("host"), "path": ns("path")}
	case "httpupgrade":
		t = Map{"type": "httpupgrade", "path": ns("path"), "host": orDefault(ns("host"), s["host"]), "headers": ns("headers")}
	case "quic":
		t = Map{"type": "quic"}
	}
	if t == nil {
		return nil
	}
	return compact(t)
}

func appendMultiplex(m Map, p Map) {
	mux, ok := p["multiplex"].(Map)
	if !ok || !Truthy(mux["enabled"]) {
		return
	}
	o := Map{"enabled": true, "protocol": orDefault(mux["protocol"], "yamux"),
		"max_connections": mux["max_connections"], "min_streams": mux["min_streams"],
		"max_streams": mux["max_streams"], "padding": Truthy(mux["padding"])}
	if Truthy(Get(mux, "brutal.enabled")) {
		o["brutal"] = Map{"enabled": true, "up_mbps": Get(mux, "brutal.up_mbps"), "down_mbps": Get(mux, "brutal.down_mbps")}
	}
	m["multiplex"] = compact(o)
}

func appendUtls(tls Map, p Map) {
	if u, ok := p["utls"].(Map); ok && Truthy(u["enabled"]) {
		tls["utls"] = Map{"enabled": true, "fingerprint": tlsFingerprint(u)}
	}
}

func appendEch(tls Map, ech any) {
	m, ok := ech.(Map)
	if !ok || !Truthy(m["enabled"]) {
		return
	}
	o := Map{"enabled": true}
	if c := trimToNull(m["config"]); c != nil {
		o["config"] = []any{c}
	}
	if q := trimToNull(m["query_server_name"]); q != nil {
		o["query_server_name"] = q
	}
	tls["ech"] = o
}

// ensureDefaultDomainResolver 在缺省时补上 route.default_domain_resolver，指向模板里不经代理的直连 DNS。
//
// 两个原因：
//  1. 代理节点自己的域名必须走直连 DNS 解析（经代理解析代理地址会死锁）；sing-box 1.12 起推荐显式声明。
//  2. hiddify-app（LetsVPN）的 ProfilePatcher 在配置里没有该字段时，会给每个代理出站写入
//     domain_resolver=<这里的直连 DNS 标签>；而 hiddify-core 组装最终配置时会换掉整套 DNS，
//     这个标签随之消失，所有出站以 "domain resolver not found" 建失败、被替换成 hinvalid，
//     表现为「已连接但所有节点都不能用」。声明了全局解析器后补丁不会触发。
//
// 该字段 1.12.0 才有，已知客户端版本低于 1.12 时不写，避免旧核心报未知字段。
func (b *sbBuilder) ensureDefaultDomainResolver() {
	if v := b.coreVersion(); v != "" && versionCompare(v, "1.12.0") < 0 {
		return
	}
	route, _ := b.cfg["route"].(Map)
	if route == nil {
		return
	}
	if cur, ok := route["default_domain_resolver"]; ok && cur != nil && cur != "" {
		return
	}
	servers, _ := Get(b.cfg, "dns.servers").([]any)
	for _, s := range servers {
		m, _ := s.(Map)
		if Str(m["detour"]) == "direct" && Str(m["tag"]) != "" {
			route["default_domain_resolver"] = Str(m["tag"])
			return
		}
	}
}
