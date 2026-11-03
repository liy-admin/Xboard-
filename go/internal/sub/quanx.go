package sub

import (
	"encoding/base64"
	"strings"
)

// QuantumultX 配置生成（对应 app/Protocols/QuantumultX.php）。输出整体 base64。

var quanxAllowed = map[string]bool{
	"shadowsocks": true, "vmess": true, "vless": true, "trojan": true, "anytls": true, "socks": true, "http": true,
}

// quanxTLS 为 applyTransportSettings 的 TLS 数据来源（trojan 使用 tls_settings 的快照）。
type quanxTLS struct {
	allowInsecure bool
	serverName    string
}

// quanxFirst 对应 is_array($v) ? ($v[0] ?? '/') : $v，缺省为 "/"。
func quanxFirst(v any, def string) string {
	if arr, ok := v.([]any); ok {
		if len(arr) == 0 {
			return def
		}
		return Str(arr[0])
	}
	if v == nil {
		return def
	}
	return Str(v)
}

// quanxTransport 对应 applyTransportSettings。nativeTLS 为 true 时使用 over-tls / tls-host 写法。
func quanxTransport(cfg []string, s Map, nativeTLS bool, tlsData *quanxTLS) []string {
	ps := "protocol_settings"
	tlsMode := Int(Get(s, ps+".tls"))
	network := Str(Get(s, ps+".network"))
	if network == "" {
		network = "tcp"
	}
	var host string
	isWs := network == "ws"
	switch network {
	case "ws":
		if tlsMode != 0 {
			cfg = append(cfg, "obfs=wss")
		} else {
			cfg = append(cfg, "obfs=ws")
		}
		if p := Str(Get(s, ps+".network_settings.path")); p != "" {
			cfg = append(cfg, "obfs-uri="+p)
		}
		host = Str(Get(s, ps+".network_settings.headers.Host"))
	case "tcp":
		headerType := Str(Get(s, ps+".network_settings.header.type"))
		if headerType == "" {
			headerType = "tcp"
		}
		if headerType == "http" {
			cfg = append(cfg, "obfs=http")
			paths := Get(s, ps+".network_settings.header.request.path")
			if paths == nil {
				paths = []any{"/"}
			}
			cfg = append(cfg, "obfs-uri="+quanxFirst(paths, "/"))
			host = quanxFirst(Get(s, ps+".network_settings.header.request.headers.Host"), "")
		} else if tlsMode != 0 {
			if nativeTLS {
				cfg = append(cfg, "over-tls=true")
			} else {
				cfg = append(cfg, "obfs=over-tls")
			}
		}
	}
	switch tlsMode {
	case 2:
		if host == "" {
			host = Str(Get(s, ps+".reality_settings.server_name"))
		}
		if pk := Str(Get(s, ps+".reality_settings.public_key")); pk != "" {
			cfg = append(cfg, "reality-base64-pubkey="+pk)
		}
		if sid := Str(Get(s, ps+".reality_settings.short_id")); sid != "" {
			cfg = append(cfg, "reality-hex-shortid="+sid)
		}
	case 1:
		allow, sn := false, ""
		if tlsData != nil {
			allow, sn = tlsData.allowInsecure, tlsData.serverName
		} else {
			allow = Truthy(Get(s, ps+".tls_settings.allow_insecure"))
			sn = Str(Get(s, ps+".tls_settings.server_name"))
		}
		if allow {
			cfg = append(cfg, "tls-verification=false")
		} else {
			cfg = append(cfg, "tls-verification=true")
		}
		if host == "" {
			host = sn
		}
	}
	if host != "" {
		if nativeTLS && !isWs {
			cfg = append(cfg, "tls-host="+host)
		} else {
			cfg = append(cfg, "obfs-host="+host)
		}
	}
	return cfg
}

// quanxCommon 对应 applyCommonSettings。
func quanxCommon(cfg []string, s Map) []string {
	cfg = append(cfg, "fast-open=true")
	if Str(s["type"]) != "http" {
		cfg = append(cfg, "udp-relay=true")
	}
	return append(cfg, "tag="+Str(s["name"]))
}

func quanxSS(pw string, s Map) string {
	addr := wrapIPv6(Str(s["host"]))
	cfg := []string{"shadowsocks=" + addr + ":" + Str(s["port"]),
		"method=" + Str(Get(s, "protocol_settings.cipher")), "password=" + pw}
	if Truthy(Get(s, "protocol_settings.plugin")) && Truthy(Get(s, "protocol_settings.plugin_opts")) {
		if Str(Get(s, "protocol_settings.plugin")) == "obfs" {
			opts := pluginOpts(Str(Get(s, "protocol_settings.plugin_opts")))
			if v, ok := opts["obfs"]; ok {
				cfg = append(cfg, "obfs="+v)
			}
			if v, ok := opts["obfs-host"]; ok {
				cfg = append(cfg, "obfs-host="+v)
			}
			if v, ok := opts["path"]; ok {
				cfg = append(cfg, "obfs-uri="+v)
			}
		}
	}
	cfg = quanxCommon(cfg, s)
	return joinLine(cfg)
}

func quanxVmess(uuid string, s Map) string {
	addr := wrapIPv6(Str(s["host"]))
	method := Str(Get(s, "protocol_settings.cipher"))
	if method == "" {
		method = "auto"
	}
	cfg := []string{"vmess=" + addr + ":" + Str(s["port"]), "method=" + method, "password=" + uuid}
	cfg = quanxTransport(cfg, s, false, nil)
	return joinLine(quanxCommon(cfg, s))
}

func quanxVless(uuid string, s Map) string {
	addr := wrapIPv6(Str(s["host"]))
	cfg := []string{"vless=" + addr + ":" + Str(s["port"]), "method=none", "password=" + uuid}
	cfg = quanxTransport(cfg, s, false, nil)
	if flow := Str(Get(s, "protocol_settings.flow")); flow != "" {
		cfg = append(cfg, "vless-flow="+flow)
	}
	return joinLine(quanxCommon(cfg, s))
}

func quanxTrojan(pw string, s Map) string {
	addr := wrapIPv6(Str(s["host"]))
	cfg := []string{"trojan=" + addr + ":" + Str(s["port"]), "password=" + pw}
	tls := &quanxTLS{
		allowInsecure: Truthy(Get(s, "protocol_settings.tls_settings.allow_insecure")),
		serverName:    Str(Get(s, "protocol_settings.tls_settings.server_name")),
	}
	cfg = quanxTransport(cfg, s, true, tls)
	return joinLine(quanxCommon(cfg, s))
}

func quanxAnyTLS(pw string, s Map) string {
	addr := wrapIPv6(Str(s["host"]))
	cfg := []string{"anytls=" + addr + ":" + Str(s["port"]), "password=" + pw,
		"udp-relay=true", "tag=" + Str(s["name"]), "over-tls=true"}
	if !Truthy(Get(s, "protocol_settings.tls.allow_insecure")) {
		cfg = append(cfg, "tls-verification=true")
	}
	if sn := Str(Get(s, "protocol_settings.tls.server_name")); sn != "" {
		cfg = append(cfg, "tls-host="+sn)
	}
	return joinLine(cfg)
}

func quanxSocksLike(kind, pw string, s Map) string {
	addr := wrapIPv6(Str(s["host"]))
	cfg := []string{kind + "=" + addr + ":" + Str(s["port"]), "username=" + pw, "password=" + pw}
	cfg = quanxTransport(cfg, s, true, nil)
	return joinLine(quanxCommon(cfg, s))
}

// BuildQuantumultX 生成 QuantumultX 订阅（整体 base64）。
func BuildQuantumultX(ctx *Ctx) (string, string, map[string]string) {
	var sb strings.Builder
	for _, s := range filterServers(ctx, quanxAllowed, nil) {
		pw := Str(s["password"])
		switch Str(s["type"]) {
		case "shadowsocks":
			sb.WriteString(quanxSS(pw, s))
		case "vmess":
			sb.WriteString(quanxVmess(pw, s))
		case "vless":
			sb.WriteString(quanxVless(pw, s))
		case "trojan":
			sb.WriteString(quanxTrojan(pw, s))
		case "anytls":
			sb.WriteString(quanxAnyTLS(pw, s))
		case "socks":
			sb.WriteString(quanxSocksLike("socks5", pw, s))
		case "http":
			sb.WriteString(quanxSocksLike("http", pw, s))
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(sb.String())), "text/plain", map[string]string{
		"subscription-userinfo": userinfoHeader(ctx),
	}
}
