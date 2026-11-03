package sub

import (
	"strings"
)

// ClashKind 区分三种 Clash 系输出，字段映射细节在各协议构造函数里按 kind 分支。
type ClashKind int

const (
	ClashKindClash ClashKind = iota
	ClashKindMeta
	ClashKindStash
)

// parsePluginOpts 对应 SS 插件参数解析；meta 下无 '=' 的项记为 true，其余丢弃。
func parsePluginOpts(raw string, meta bool) Map {
	out := Map{}
	for _, pair := range strings.Split(raw, ";") {
		if !Truthy(pair) {
			continue
		}
		if !strings.Contains(pair, "=") {
			if meta {
				out[strings.TrimSpace(pair)] = true
			}
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}
	return out
}

// nonEmpty 等价 array_filter 默认行为：去掉 nil 与空字符串。
func nonEmpty(m Map) Map {
	for k, v := range m {
		if v == nil || v == "" {
			delete(m, k)
		}
	}
	return m
}

func orAny(vals ...any) any {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

// echMihomo 对应 Helper::normalizeEchSettings + toMihomoEchConfig，返回 ech-opts；未启用返回 nil。
func echMihomo(ech any) Map {
	m, ok := ech.(Map)
	if !ok || !Truthy(m["enabled"]) {
		return nil
	}
	cfg := strings.TrimSpace(Str(m["config"]))
	if strings.HasPrefix(cfg, "-----BEGIN") {
		cfg = pemBody(cfg)
	} else {
		cfg = strings.Join(strings.Fields(cfg), "")
	}
	opts := Map{"enable": true, "config": nil, "query-server-name": trimToNull(m["query_server_name"])}
	if cfg != "" {
		opts["config"] = cfg
	}
	return nonEmpty(opts)
}

func pemBody(s string) string {
	const beg, end = "-----BEGIN ECH CONFIGS-----", "-----END ECH CONFIGS-----"
	i := strings.Index(s, beg)
	j := strings.Index(s, end)
	if i < 0 || j < 0 || j < i {
		return ""
	}
	return strings.Join(strings.Fields(s[i+len(beg):j]), "")
}

func clashSmux(m Map, p Map) {
	mux, ok := p["multiplex"].(Map)
	if !ok || !Truthy(mux["enabled"]) {
		return
	}
	s := nonEmpty(Map{"enabled": true, "protocol": orDefault(mux["protocol"], "yamux"),
		"max-connections": mux["max_connections"]})
	if Truthy(mux["padding"]) {
		s["padding"] = true
	}
	if Truthy(Get(mux, "brutal.enabled")) {
		s["brutal-opts"] = Map{"enabled": true, "up": Get(mux, "brutal.up_mbps"), "down": Get(mux, "brutal.down_mbps")}
	}
	m["smux"] = s
}

func utlsFingerprint(m Map, p Map) {
	if u, ok := p["utls"].(Map); ok && Truthy(u["enabled"]) {
		m["client-fingerprint"] = tlsFingerprint(u)
	}
}

func tcpHTTPOpts(p Map, withHost bool) (network string, opts Map) {
	ht := Str(Get(p, "network_settings.header.type"))
	if ht == "" {
		ht = "none"
	}
	if ht != "http" {
		return "tcp", nil
	}
	opts = Map{"path": orDefault(Get(p, "network_settings.header.request.path"), []any{"/"})}
	if withHost {
		if h := Get(p, "network_settings.header.request.headers"); h != nil {
			opts["headers"] = h
		}
	}
	return "http", nonEmpty(opts)
}

// ---- Shadowsocks ----

func clashSS(pw string, s Map, kind ClashKind) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "ss", "server": s["host"], "port": s["port"],
		"cipher": p["cipher"], "password": pw, "udp": true}
	if !(Truthy(p["plugin"]) && Truthy(p["plugin_opts"])) {
		return m
	}
	pl := Str(p["plugin"])
	m["plugin"] = pl
	parsed := parsePluginOpts(Str(p["plugin_opts"]), kind == ClashKindMeta)
	ns := func(k string) any { return Get(p, k) }
	switch {
	case kind == ClashKindMeta && (pl == "obfs" || pl == "obfs-local"):
		m["plugin"] = "obfs"
		m["plugin-opts"] = nonEmpty(Map{
			"mode": orAny(parsed["obfs"], parsed["mode"], "http"),
			"host": orAny(parsed["obfs-host"], parsed["host"], "www.bing.com"),
		})
	case kind == ClashKindMeta && pl == "v2ray-plugin":
		o := Map{"mode": orAny(parsed["mode"], "websocket"),
			"tls":  parsed["tls"] != nil || parsed["server"] != nil,
			"host": parsed["host"], "path": orAny(parsed["path"], "/")}
		if parsed["mux"] != nil {
			o["mux"] = true
		}
		if parsed["host"] != nil {
			o["headers"] = Map{"Host": parsed["host"]}
		}
		m["plugin-opts"] = compact(o)
	case kind == ClashKindMeta && pl == "shadow-tls":
		v := 2
		if parsed["version"] != nil {
			v = Int(parsed["version"])
		}
		m["plugin-opts"] = compact(Map{"host": parsed["host"], "password": parsed["password"], "version": v})
	case kind == ClashKindMeta && pl == "restls":
		m["plugin-opts"] = compact(Map{"host": parsed["host"], "password": parsed["password"],
			"restls-script": orAny(parsed["restls-script"], "123")})
	case kind == ClashKindClash && pl == "obfs":
		m["plugin-opts"] = nonEmpty(Map{
			"mode": orAny(parsed["obfs"], ns("obfs"), "http"),
			"host": orAny(parsed["obfs-host"], ns("obfs_settings.host"), ""),
		})
		if parsed["path"] != nil {
			m["plugin-opts"].(Map)["path"] = parsed["path"]
		}
	case kind == ClashKindStash && pl == "obfs":
		o := Map{"mode": parsed["obfs"], "host": parsed["obfs-host"]}
		if parsed["path"] != nil {
			o["path"] = parsed["path"]
		}
		m["plugin-opts"] = o
	case (kind == ClashKindClash || kind == ClashKindStash) && pl == "v2ray-plugin":
		m["plugin-opts"] = Map{"mode": orAny(parsed["mode"], "websocket"),
			"tls":  Str(parsed["tls"]) == "true",
			"host": orAny(parsed["host"], ""), "path": orAny(parsed["path"], "/")}
	default:
		m["plugin-opts"] = parsed
	}
	return m
}

// ---- VMess ----

func clashVmess(uuid string, s Map, kind ClashKind) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "vmess", "server": s["host"], "port": s["port"],
		"uuid": uuid, "alterId": 0, "cipher": "auto", "udp": true}
	tls := Truthy(p["tls"])
	sn := Get(p, "tls_settings.server_name")
	insecure := Truthy(Get(p, "tls_settings.allow_insecure"))
	switch kind {
	case ClashKindClash:
		if tls {
			m["tls"], m["skip-cert-verify"] = true, insecure
			if Truthy(sn) {
				m["servername"] = sn
			}
		}
	case ClashKindMeta:
		if tls {
			m["tls"], m["skip-cert-verify"], m["servername"] = true, insecure, sn
			if e := echMihomo(Get(p, "tls_settings.ech")); e != nil {
				m["ech-opts"] = e
			}
		}
		utlsFingerprint(m, p)
		clashSmux(m, p)
	case ClashKindStash:
		m["tls"], m["skip-cert-verify"] = tls, insecure
		if Truthy(sn) {
			m["servername"] = sn
		}
	}
	ns := func(k string) any { return Get(p, "network_settings."+k) }
	switch Str(p["network"]) {
	case "tcp":
		withHost := kind != ClashKindStash
		if kind == ClashKindStash {
			ht := Str(ns("header.type"))
			if ht == "" {
				ht = "tcp"
			}
			m["network"] = map[bool]string{true: "http", false: "tcp"}[ht == "http"]
			if ht == "http" {
				h := Map{"path": orDefault(ns("header.request.path"), []any{"/"})}
				if host := ns("header.request.headers.Host"); Truthy(host) {
					h["headers"] = Map{"Host": host}
				}
				m["http-opts"] = h
			}
			break
		}
		net, opts := tcpHTTPOpts(p, withHost)
		m["network"] = net
		if opts != nil {
			m["http-opts"] = opts
		}
	case "ws":
		m["network"] = "ws"
		wo := Map{}
		if v := ns("path"); Truthy(v) || kind == ClashKindStash {
			wo["path"] = v
		}
		if h := ns("headers.Host"); Truthy(h) {
			wo["headers"] = Map{"Host": h}
		}
		if len(wo) > 0 {
			m["ws-opts"] = wo
		}
	case "grpc":
		m["network"] = "grpc"
		if v := ns("serviceName"); Truthy(v) || kind == ClashKindStash {
			m["grpc-opts"] = Map{"grpc-service-name": v}
		}
	case "h2":
		if kind == ClashKindClash {
			break
		}
		m["network"] = "h2"
		if kind == ClashKindStash {
			m["tls"] = true
		}
		h2 := Map{}
		if v := ns("path"); Truthy(v) {
			h2["path"] = v
		}
		if v := ns("host"); Truthy(v) {
			h2["host"] = asList(v)
		}
		m["h2-opts"] = h2
	case "httpupgrade":
		if kind == ClashKindMeta {
			m["network"] = "ws"
			wo := Map{"v2ray-http-upgrade": true}
			if v := ns("path"); Truthy(v) {
				wo["path"] = v
			}
			if v := ns("host"); Truthy(v) {
				wo["headers"] = Map{"Host": v}
			}
			m["ws-opts"] = wo
		}
	}
	return m
}

func asList(v any) any {
	if _, ok := v.([]any); ok {
		return v
	}
	return []any{v}
}

// ---- VLESS（Clash 无 VLESS；Meta 与 Stash 映射不同） ----

func clashVless(uuid string, s Map, kind ClashKind) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "vless", "server": s["host"], "port": s["port"], "uuid": uuid, "udp": true}
	ns := func(k string) any { return Get(p, "network_settings."+k) }
	if kind == ClashKindMeta {
		m["alterId"], m["cipher"] = 0, "auto"
		m["flow"] = p["flow"]
		enc := "none"
		if Get(p, "encryption.enabled") == true {
			enc = Str(orAny(Get(p, "encryption.encryption"), "none"))
		}
		m["encryption"], m["tls"] = enc, false
		switch Int(p["tls"]) {
		case 1:
			m["tls"] = true
			m["skip-cert-verify"] = Truthy(Get(p, "tls_settings.allow_insecure"))
			if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
				m["servername"] = sn
			}
			if e := echMihomo(Get(p, "tls_settings.ech")); e != nil {
				m["ech-opts"] = e
			}
			utlsFingerprint(m, p)
		case 2:
			m["tls"] = true
			m["skip-cert-verify"] = Truthy(Get(p, "reality_settings.allow_insecure"))
			m["servername"] = Get(p, "reality_settings.server_name")
			m["reality-opts"] = Map{"public-key": Get(p, "reality_settings.public_key"), "short-id": Get(p, "reality_settings.short_id")}
			utlsFingerprint(m, p)
		}
		switch Str(p["network"]) {
		case "tcp":
			net, opts := tcpHTTPOpts(p, true)
			m["network"] = net
			if opts != nil {
				m["http-opts"] = opts
			}
		case "ws":
			m["network"] = "ws"
			wo := Map{}
			if v := ns("path"); Truthy(v) {
				wo["path"] = v
			}
			if v := ns("headers.Host"); Truthy(v) {
				wo["headers"] = Map{"Host": v}
			}
			if len(wo) > 0 {
				m["ws-opts"] = wo
			}
		case "grpc":
			m["network"] = "grpc"
			if v := ns("serviceName"); Truthy(v) {
				m["grpc-opts"] = Map{"grpc-service-name": v}
			}
		case "h2":
			m["network"] = "h2"
			h2 := Map{}
			if v := ns("path"); Truthy(v) {
				h2["path"] = v
			}
			if v := ns("host"); Truthy(v) {
				h2["host"] = asList(v)
			}
			m["h2-opts"] = h2
		case "httpupgrade":
			m["network"] = "ws"
			wo := Map{"v2ray-http-upgrade": true}
			if v := ns("path"); Truthy(v) {
				wo["path"] = v
			}
			if v := ns("host"); Truthy(v) {
				wo["headers"] = Map{"Host": v}
			}
			m["ws-opts"] = wo
		case "xhttp":
			m["network"] = "xhttp"
			xo := Map{}
			if v := ns("path"); Truthy(v) {
				xo["path"] = v
			}
			if v := ns("host"); Truthy(v) {
				xo["host"] = v
			}
			if v := ns("mode"); Truthy(v) {
				xo["mode"] = v
			}
			if len(xo) > 0 {
				m["xhttp-opts"] = xo
			}
		}
		clashSmux(m, p)
		return m
	}
	// Stash
	if fp := tlsFingerprint(p["utls"]); fp != "" {
		m["client-fingerprint"] = fp
	}
	switch Int(p["tls"]) {
	case 1:
		m["tls"] = true
		m["skip-cert-verify"] = Get(p, "tls_settings.allow_insecure")
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			m["servername"] = sn
		}
	case 2:
		m["tls"] = true
		m["skip-cert-verify"] = Truthy(Get(p, "reality_settings.allow_insecure"))
		if sn := Get(p, "reality_settings.server_name"); Truthy(sn) {
			m["servername"], m["sni"] = sn, sn
		}
		m["flow"] = p["flow"]
		m["reality-opts"] = Map{"public-key": Get(p, "reality_settings.public_key"), "short-id": Get(p, "reality_settings.short_id")}
	}
	switch Str(p["network"]) {
	case "tcp":
		net, opts := tcpHTTPOpts(p, true)
		m["network"] = net
		if opts != nil {
			m["http-opts"] = opts
		}
	case "ws":
		m["network"] = "ws"
		wo := Map{"path": ns("path")}
		if h := ns("headers.Host"); Truthy(h) {
			wo["headers"] = Map{"Host": h}
		}
		m["ws-opts"] = wo
	case "grpc":
		m["network"] = "grpc"
		m["grpc-opts"] = Map{"grpc-service-name": ns("serviceName")}
	case "h2":
		m["network"] = "h2"
		h2 := Map{}
		if v := ns("path"); Truthy(v) {
			h2["path"] = v
		}
		if v := ns("host"); Truthy(v) {
			h2["host"] = asList(v)
		}
		m["h2-opts"] = h2
	}
	return m
}

// ---- Trojan ----

func clashTrojan(pw string, s Map, kind ClashKind) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "trojan", "server": s["host"], "port": s["port"], "password": pw, "udp": true}
	mode := 1
	if p["tls"] != nil {
		mode = Int(p["tls"])
	}
	switch kind {
	case ClashKindClash:
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			m["sni"] = sn
		}
		m["skip-cert-verify"] = Truthy(Get(p, "tls_settings.allow_insecure"))
		m["network"] = "tcp"
		switch Str(p["network"]) {
		case "ws":
			m["network"] = "ws"
			wo := Map{"path": Get(p, "network_settings.path")}
			if h := Get(p, "network_settings.headers.Host"); Truthy(h) {
				wo["headers"] = Map{"Host": h}
			}
			m["ws-opts"] = wo
		case "grpc":
			m["network"] = "grpc"
			if v := Get(p, "network_settings.serviceName"); Truthy(v) {
				m["grpc-opts"] = Map{"grpc-service-name": v}
			}
		}
	case ClashKindMeta:
		if mode == 2 {
			m["skip-cert-verify"] = Truthy(Get(p, "reality_settings.allow_insecure"))
			if sn := Get(p, "reality_settings.server_name"); Truthy(sn) {
				m["sni"] = sn
			}
			m["reality-opts"] = Map{"public-key": Get(p, "reality_settings.public_key"), "short-id": Get(p, "reality_settings.short_id")}
		} else {
			m["skip-cert-verify"] = Truthy(Get(p, "tls_settings.allow_insecure"))
			if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
				m["sni"] = sn
			}
			if e := echMihomo(Get(p, "tls_settings.ech")); e != nil {
				m["ech-opts"] = e
			}
		}
		utlsFingerprint(m, p)
		clashSmux(m, p)
		m["network"] = "tcp"
		switch Str(p["network"]) {
		case "ws":
			m["network"] = "ws"
			wo := Map{"path": Get(p, "network_settings.path")}
			if h := Get(p, "network_settings.headers.Host"); Truthy(h) {
				wo["headers"] = Map{"Host": h}
			}
			m["ws-opts"] = wo
		case "grpc":
			m["network"] = "grpc"
			if v := Get(p, "network_settings.serviceName"); Truthy(v) {
				m["grpc-opts"] = Map{"grpc-service-name": v}
			}
		case "httpupgrade":
			m["network"] = "ws"
			wo := Map{"v2ray-http-upgrade": true}
			if v := Get(p, "network_settings.path"); Truthy(v) {
				wo["path"] = v
			}
			if v := Get(p, "network_settings.host"); Truthy(v) {
				wo["headers"] = Map{"Host": v}
			}
			m["ws-opts"] = wo
		}
	case ClashKindStash:
		if mode == 2 {
			m["tls"] = true
			m["skip-cert-verify"] = Truthy(Get(p, "reality_settings.allow_insecure"))
			if sn := Get(p, "reality_settings.server_name"); Truthy(sn) {
				m["sni"] = sn
			}
			m["reality-opts"] = Map{"public-key": Get(p, "reality_settings.public_key"), "short-id": Get(p, "reality_settings.short_id")}
		} else {
			if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
				m["sni"] = sn
			}
			m["skip-cert-verify"] = Truthy(Get(p, "tls_settings.allow_insecure"))
		}
		switch Str(p["network"]) {
		case "tcp":
			net, opts := tcpHTTPOpts(p, false)
			m["network"] = net
			if opts != nil {
				m["http-opts"] = opts
			}
		case "ws":
			m["network"] = "ws"
			wo := Map{"path": Get(p, "network_settings.path")}
			if h := Get(p, "network_settings.headers.Host"); Truthy(h) {
				wo["headers"] = Map{"Host": h}
			}
			m["ws-opts"] = wo
		case "grpc":
			m["network"] = "grpc"
			if v := Get(p, "network_settings.serviceName"); Truthy(v) {
				m["grpc-opts"] = Map{"grpc-service-name": v}
			}
		}
	}
	return m
}

// ---- Hysteria / TUIC / AnyTLS ----

func clashHysteria(pw string, s Map, kind ClashKind) Map {
	p := ps(s)
	version := 0
	if p["version"] != nil {
		version = Int(p["version"])
	}
	if kind == ClashKindMeta {
		m := Map{"name": s["name"], "server": s["host"], "port": s["port"],
			"sni": Get(p, "tls.server_name"), "up": Get(p, "bandwidth.up"), "down": Get(p, "bandwidth.down"),
			"skip-cert-verify": Truthy(Get(p, "tls.allow_insecure"))}
		if s["ports"] != nil {
			m["ports"] = s["ports"]
		}
		if h := p["hop_interval"]; Truthy(h) {
			m["hop-interval"] = Int(h)
		}
		if version == 2 {
			m["type"], m["password"] = "hysteria2", pw
			if Truthy(Get(p, "obfs.open")) {
				m["obfs"], m["obfs-password"] = Get(p, "obfs.type"), Get(p, "obfs.password")
			}
		} else {
			m["type"], m["auth_str"], m["protocol"] = "hysteria", pw, "udp"
			if Truthy(Get(p, "obfs.open")) {
				m["obfs"] = Get(p, "obfs.password")
			}
			m["fast-open"], m["disable_mtu_discovery"] = true, true
		}
		return m
	}
	m := Map{"name": s["name"], "server": s["host"], "port": s["port"],
		"up-speed": Get(p, "bandwidth.up"), "down-speed": Get(p, "bandwidth.down"),
		"skip-cert-verify": Get(p, "tls.allow_insecure")}
	if sn := Get(p, "tls.server_name"); Truthy(sn) {
		m["sni"] = sn
	}
	if s["ports"] != nil {
		m["ports"] = s["ports"]
	}
	if version == 2 {
		m["type"], m["auth"], m["fast-open"] = "hysteria2", pw, true
		if Truthy(Get(p, "obfs.open")) {
			m["obfs"] = orAny(Get(p, "obfs.type"), "salamander")
			m["obfs-password"] = Get(p, "obfs.password")
		}
	} else {
		m["type"], m["auth-str"], m["protocol"] = "hysteria", pw, "udp"
		if Truthy(Get(p, "obfs.open")) {
			m["obfs"] = Get(p, "obfs.password")
		}
	}
	return m
}

func clashTuic(pw string, s Map, kind ClashKind) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "tuic", "server": s["host"], "port": s["port"]}
	if kind == ClashKindMeta {
		m["udp"] = true
		if Int(p["version"]) == 4 {
			m["token"] = pw
		} else {
			m["uuid"], m["password"] = pw, pw
		}
		m["skip-cert-verify"] = Truthy(Get(p, "tls.allow_insecure"))
		if sn := Get(p, "tls.server_name"); Truthy(sn) {
			m["sni"] = sn
		}
		if a := p["alpn"]; Truthy(a) {
			m["alpn"] = a
		}
		m["congestion-controller"] = orAny(p["congestion_control"], "cubic")
		m["udp-relay-mode"] = orAny(p["udp_relay_mode"], "native")
		return m
	}
	m["congestion-controller"] = orAny(p["congestion_control"], "cubic")
	m["udp-relay-mode"] = orAny(p["udp_relay_mode"], "native")
	m["alpn"] = orAny(p["alpn"], []any{"h3"})
	m["reduce-rtt"], m["fast-open"] = true, true
	m["heartbeat-interval"], m["request-timeout"], m["max-udp-relay-packet-size"] = 10000, 8000, 1500
	m["version"] = orAny(p["version"], 5)
	if Int(p["version"]) == 4 {
		m["token"] = pw
	} else {
		m["uuid"], m["password"] = pw, pw
	}
	m["skip-cert-verify"] = Truthy(Get(p, "tls.allow_insecure"))
	if sn := Get(p, "tls.server_name"); Truthy(sn) {
		m["sni"] = sn
	}
	return m
}

func clashAnyTLS(pw string, s Map, kind ClashKind) Map {
	p := ps(s)
	if kind == ClashKindMeta {
		m := Map{"name": s["name"], "type": "anytls", "server": s["host"], "port": s["port"], "password": pw, "udp": true}
		if sn := Get(p, "tls.server_name"); Truthy(sn) {
			m["sni"] = sn
		}
		if Truthy(Get(p, "tls.allow_insecure")) {
			m["skip-cert-verify"] = true
		}
		if e := echMihomo(Get(p, "tls.ech")); e != nil {
			m["ech-opts"] = e
		}
		return m
	}
	return Map{"name": s["name"], "type": "anytls", "server": s["host"], "port": s["port"], "password": pw,
		"sni": Get(p, "tls.server_name"), "skip-cert-verify": Truthy(Get(p, "tls.allow_insecure")), "udp": true}
}

// ---- SOCKS5 / HTTP / Mieru ----

func clashSocks5(pw string, s Map, kind ClashKind) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "socks5", "server": s["host"], "port": s["port"],
		"username": pw, "password": pw, "udp": true}
	if Truthy(p["tls"]) {
		m["tls"] = true
		m["skip-cert-verify"] = Truthy(Get(p, "tls_settings.allow_insecure"))
		if kind == ClashKindStash {
			if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
				m["sni"] = sn
			}
		}
	}
	return m
}

func clashHTTP(pw string, s Map, kind ClashKind) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "http", "server": s["host"], "port": s["port"],
		"username": pw, "password": pw}
	if Truthy(p["tls"]) {
		m["tls"] = true
		m["skip-cert-verify"] = Truthy(Get(p, "tls_settings.allow_insecure"))
		if sn := Get(p, "tls_settings.server_name"); Truthy(sn) {
			m["sni"] = sn
		}
	}
	return m
}

func clashMieru(pw string, s Map) Map {
	p := ps(s)
	m := Map{"name": s["name"], "type": "mieru", "server": s["host"], "port": s["port"],
		"username": pw, "password": pw, "transport": strings.ToUpper(Str(orAny(p["transport"], "TCP")))}
	if s["ports"] != nil {
		m["port-range"] = s["ports"]
	}
	return m
}
