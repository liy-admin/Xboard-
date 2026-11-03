package sub

import "strings"

// Loon 配置生成（对应 app/Protocols/Loon.php）。输出为明文，不做 base64。
//
// 与 PHP 的差异：Hysteria2 在没有 SNI 时 PHP 会输出字面量 "(null)"，这里直接省略该字段（见 PROGRESS.md）。

var loonAllowed = map[string]bool{
	"shadowsocks": true, "vmess": true, "trojan": true, "hysteria": true, "vless": true, "anytls": true,
}

var loonRequirements = requirements{
	"loon": {
		"hysteria": {"protocol_settings.version": {"2": "637"}},
		"trojan":   {"protocol_settings.tls": {"0": "3.2.1", "1": "3.2.1", "2": "999.9.9"}},
	},
}

// loonPick 对应 $arr[array_rand($arr)]：数组随机取一项，标量原样返回。
func loonPick(v any) string {
	if arr, ok := v.([]any); ok {
		if len(arr) == 0 {
			return ""
		}
		return Str(randPick(arr, nil))
	}
	return Str(v)
}

// loonFirst 对应 is_array($x) ? $x[0] : $x。
func loonFirst(v any) string {
	if arr, ok := v.([]any); ok {
		if len(arr) == 0 {
			return ""
		}
		return Str(arr[0])
	}
	return Str(v)
}

// loonSSObfs 解析 obfs 插件参数，Loon 使用 obfs-name 而非 obfs。
func loonSSObfs(s Map, cfg []string) []string {
	if !Truthy(Get(s, "protocol_settings.plugin")) || !Truthy(Get(s, "protocol_settings.plugin_opts")) {
		return cfg
	}
	if Str(Get(s, "protocol_settings.plugin")) != "obfs" {
		return cfg
	}
	opts := pluginOpts(Str(Get(s, "protocol_settings.plugin_opts")))
	cfg = append(cfg, "obfs-name="+opts["obfs"])
	if v, ok := opts["obfs-host"]; ok {
		cfg = append(cfg, "obfs-host="+v)
	}
	if v, ok := opts["path"]; ok {
		cfg = append(cfg, "obfs-uri="+v)
	}
	return cfg
}

func buildLoonSS(pw string, s Map) string {
	cfg := []string{Str(s["name"]) + "=Shadowsocks", Str(s["host"]), Str(s["port"]),
		Str(Get(s, "protocol_settings.cipher")), pw, "fast-open=false", "udp=true"}
	return joinLine(loonSSObfs(s, cfg))
}

// loonNetworkVmess 处理 vmess 的 transport 段，tcp 分支会在 header.type 存在时替换 transport 项。
func loonVmessTransport(s Map, cfg []string) []string {
	switch Str(Get(s, "protocol_settings.network")) {
	case "tcp":
		cfg = append(cfg, "transport=tcp")
		tcp := Get(s, "protocol_settings.network_settings")
		if t := Str(Get(tcp, "header.type")); t != "" {
			for i, v := range cfg {
				if v == "transport=tcp" {
					cfg[i] = "transport=" + t
				}
			}
		}
		if p := Get(tcp, "header.request.path"); Truthy(p) {
			cfg = append(cfg, "path="+loonPick(p))
		}
		if h := Get(tcp, "header.request.headers.Host"); Truthy(h) {
			cfg = append(cfg, "host="+loonPick(h))
		}
	case "ws":
		cfg = append(cfg, "transport=ws")
		ws := Get(s, "protocol_settings.network_settings")
		if p := Str(Get(ws, "path")); p != "" {
			cfg = append(cfg, "path="+p)
		}
		if h := Str(Get(ws, "headers.Host")); h != "" {
			cfg = append(cfg, "host="+h)
		}
	case "grpc":
		cfg = append(cfg, "transport=grpc")
		if sn := Str(Get(s, "protocol_settings.network_settings.serviceName")); sn != "" {
			cfg = append(cfg, "grpc-service-name="+sn)
		}
	case "h2":
		cfg = append(cfg, "transport=h2")
		if p := Str(Get(s, "protocol_settings.network_settings.path")); p != "" {
			cfg = append(cfg, "path="+p)
		}
		if h := Get(s, "protocol_settings.network_settings.host"); Truthy(h) {
			cfg = append(cfg, "host="+loonFirst(h))
		}
	case "httpupgrade":
		cfg = append(cfg, "transport=httpupgrade")
		if p := Str(Get(s, "protocol_settings.network_settings.path")); p != "" {
			cfg = append(cfg, "path="+p)
		}
		if h := Str(Get(s, "protocol_settings.network_settings.headers.Host")); h != "" {
			cfg = append(cfg, "host="+h)
		}
	}
	return cfg
}

func buildLoonVmess(uuid string, s Map) string {
	cfg := []string{Str(s["name"]) + "=vmess", Str(s["host"]), Str(s["port"]), "auto", uuid,
		"fast-open=false", "udp=true", "alterId=0"}
	if Truthy(Get(s, "protocol_settings.tls")) {
		cfg = append(cfg, "over-tls=true")
		if tls := Get(s, "protocol_settings.tls_settings"); Truthy(tls) {
			if Truthy(Get(tls, "allow_insecure")) {
				cfg = append(cfg, "skip-cert-verify=true")
			} else {
				cfg = append(cfg, "skip-cert-verify=false")
			}
			if sn := Str(Get(tls, "server_name")); sn != "" {
				cfg = append(cfg, "tls-name="+sn)
			}
		}
	}
	cfg = loonVmessTransport(s, cfg)
	return strings.Join(cfg, ",") + "\r\n"
}

// loonTransportCommon 处理 trojan / vless 共用的 ws/grpc/h2/httpupgrade 段。
// hostDefault 对应 httpupgrade 分支 data_get 的默认 host（trojan 为服务器地址，vless 同样）。
func loonTransportCommon(s Map, cfg []string, net string, hostDefault string) []string {
	ns := "protocol_settings.network_settings"
	switch net {
	case "ws":
		cfg = append(cfg, "transport=ws")
		if p := Str(Get(s, ns+".path")); p != "" {
			cfg = append(cfg, "path="+p)
		}
		if h := Str(Get(s, ns+".headers.Host")); h != "" {
			cfg = append(cfg, "host="+h)
		}
	case "grpc":
		cfg = append(cfg, "transport=grpc")
		if sn := Str(Get(s, ns+".serviceName")); sn != "" {
			cfg = append(cfg, "grpc-service-name="+sn)
		}
	case "h2":
		cfg = append(cfg, "transport=h2")
		if p := Str(Get(s, ns+".path")); p != "" {
			cfg = append(cfg, "path="+p)
		}
		if h := Get(s, ns+".host"); Truthy(h) {
			cfg = append(cfg, "host="+loonFirst(h))
		}
	case "httpupgrade":
		cfg = append(cfg, "transport=httpupgrade")
		if p := Str(Get(s, ns+".path")); p != "" {
			cfg = append(cfg, "path="+p)
		}
		h := Get(s, ns+".host")
		if h == nil {
			h = hostDefault
		}
		if hs := Str(h); hs != "" {
			cfg = append(cfg, "host="+hs)
		}
	}
	return cfg
}

func buildLoonTrojan(pw string, s Map) string {
	cfg := []string{Str(s["name"]) + "=trojan", Str(s["host"]), Str(s["port"]), pw}
	mode := Int(Get(s, "protocol_settings.tls"))
	if Get(s, "protocol_settings.tls") == nil {
		mode = 1
	}
	if mode == 2 {
		if sn := Str(Get(s, "protocol_settings.reality_settings.server_name")); sn != "" {
			cfg = append(cfg, "tls-name="+sn)
		}
		if pk := Str(Get(s, "protocol_settings.reality_settings.public_key")); pk != "" {
			cfg = append(cfg, "public-key="+pk)
		}
		if sid := Str(Get(s, "protocol_settings.reality_settings.short_id")); sid != "" {
			cfg = append(cfg, "short-id="+sid)
		}
		cfg = append(cfg, "skip-cert-verify="+boolStr(Truthy(Get(s, "protocol_settings.reality_settings.allow_insecure"))))
	} else {
		if sn := Str(Get(s, "protocol_settings.tls_settings.server_name")); sn != "" {
			cfg = append(cfg, "tls-name="+sn)
		}
		cfg = append(cfg, "skip-cert-verify="+boolStr(Truthy(Get(s, "protocol_settings.tls_settings.allow_insecure"))))
	}
	net := Str(Get(s, "protocol_settings.network"))
	if net == "" {
		net = "tcp" // data_get 的默认值；tcp 在 trojan 中不输出 transport 行
	}
	cfg = loonTransportCommon(s, cfg, net, Str(s["host"]))
	return joinLine(cfg)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func buildLoonVless(uuid string, s Map) string {
	cfg := []string{Str(s["name"]) + "=VLESS", Str(s["host"]), Str(s["port"]), uuid, "alterId=0", "udp=true"}
	if flow := Str(Get(s, "protocol_settings.flow")); flow != "" {
		cfg = append(cfg, "flow="+flow)
	}
	switch Int(Get(s, "protocol_settings.tls")) {
	case 1:
		cfg = append(cfg, "over-tls=true",
			"skip-cert-verify="+boolStr(Truthy(Get(s, "protocol_settings.tls_settings.allow_insecure"))))
		if sn := Str(Get(s, "protocol_settings.tls_settings.server_name")); sn != "" {
			cfg = append(cfg, "sni="+sn)
		}
	case 2:
		cfg = append(cfg, "over-tls=true",
			"skip-cert-verify="+boolStr(Truthy(Get(s, "protocol_settings.reality_settings.allow_insecure"))))
		if sn := Str(Get(s, "protocol_settings.reality_settings.server_name")); sn != "" {
			cfg = append(cfg, "sni="+sn)
		}
		if pk := Str(Get(s, "protocol_settings.reality_settings.public_key")); pk != "" {
			cfg = append(cfg, "public-key="+pk)
		}
		if sid := Str(Get(s, "protocol_settings.reality_settings.short_id")); sid != "" {
			cfg = append(cfg, "short-id="+sid)
		}
	default:
		cfg = append(cfg, "over-tls=false")
	}
	switch net := Str(Get(s, "protocol_settings.network")); net {
	case "ws", "grpc", "h2", "httpupgrade":
		cfg = loonTransportCommon(s, cfg, net, Str(s["host"]))
	default:
		cfg = append(cfg, "transport=tcp")
	}
	return joinLine(cfg)
}

// buildLoonHysteria 仅输出 v2；其它版本返回空串。
func buildLoonHysteria(pw string, s Map) string {
	if Str(Get(s, "protocol_settings.version")) != "2" {
		return ""
	}
	cfg := []string{Str(s["name"]) + "=Hysteria2", Str(s["host"]), Str(s["port"]), pw}
	if sn := Str(Get(s, "protocol_settings.tls.server_name")); sn != "" {
		cfg = append(cfg, "sni="+sn)
	}
	if Truthy(Get(s, "protocol_settings.tls.allow_insecure")) {
		cfg = append(cfg, "skip-cert-verify=true")
	}
	if down := Str(Get(s, "protocol_settings.bandwidth.down")); down != "" && down != "0" {
		cfg = append(cfg, "download-bandwidth="+down)
	}
	cfg = append(cfg, "udp=true")
	return joinLine(cfg)
}

func buildLoonAnyTLS(pw string, s Map) string {
	cfg := []string{Str(s["name"]) + "=anytls", Str(s["host"]), Str(s["port"]), pw, "udp=true"}
	if sn := Str(Get(s, "protocol_settings.tls.server_name")); sn != "" {
		cfg = append(cfg, "sni="+sn)
	}
	if Truthy(Get(s, "protocol_settings.tls.allow_insecure")) {
		cfg = append(cfg, "skip-cert-verify=true")
	}
	return joinLine(cfg)
}

// BuildLoon 生成 Loon 订阅（明文）。
func BuildLoon(ctx *Ctx) (string, string, map[string]string) {
	var sb strings.Builder
	for _, s := range filterServers(ctx, loonAllowed, loonRequirements) {
		pw := Str(s["password"])
		switch Str(s["type"]) {
		case "shadowsocks":
			sb.WriteString(buildLoonSS(pw, s))
		case "vmess":
			sb.WriteString(buildLoonVmess(pw, s))
		case "trojan":
			sb.WriteString(buildLoonTrojan(pw, s))
		case "hysteria":
			sb.WriteString(buildLoonHysteria(pw, s))
		case "vless":
			sb.WriteString(buildLoonVless(pw, s))
		case "anytls":
			sb.WriteString(buildLoonAnyTLS(pw, s))
		}
	}
	return sb.String(), "text/plain", map[string]string{
		"Subscription-Userinfo": userinfoHeader(ctx),
	}
}
