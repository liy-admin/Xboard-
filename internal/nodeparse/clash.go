package nodeparse

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Clash/mihomo YAML 订阅解析：把 proxies 里的每个节点转换成等价的分享链接，再交给现有解析函数，
// 这样两种订阅格式导入后的 protocol_settings 结构完全一致。

type clashDoc struct {
	Proxies []map[string]any `yaml:"proxies"`
}

var (
	clashMarker = regexp.MustCompile(`(?m)^proxies:\s*$`)
	// 行首是分享链接（vmess:// 等）说明是链接列表；YAML 里的 DNS 配置也含 https:// 但不在行首作为代理链接。
	shareLinkLine = regexp.MustCompile(`(?m)^\s*(vmess|vless|trojan|ss|ssr|hysteria2?|hy2|tuic|anytls)://`)
)

// LooksLikeClashYAML 判断订阅内容是否为 Clash YAML（顶层存在 proxies: 且没有分享链接行）。
func LooksLikeClashYAML(body string) bool {
	return clashMarker.MatchString(body) && !shareLinkLine.MatchString(body)
}

// ParseClashYAML 解析 Clash YAML 订阅里的 proxies。不支持的类型记录到 Errors。
func ParseClashYAML(body string) Result {
	var res Result
	var doc clashDoc
	if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
		res.Errors = append(res.Errors, ParseError{Error: "Clash YAML 解析失败: " + err.Error()})
		return res
	}
	for i, p := range doc.Proxies {
		link := proxyToLink(p)
		if link == "" {
			res.Errors = append(res.Errors, ParseError{Line: i + 1, Content: cstr(p["name"]), Error: "不支持的节点类型: " + cstr(p["type"])})
			continue
		}
		s, ok := parseLine(link)
		if !ok {
			res.Errors = append(res.Errors, ParseError{Line: i + 1, Content: cstr(p["name"]), Error: "无法识别的格式"})
			continue
		}
		res.Servers = append(res.Servers, s)
	}
	return res
}

// cstr 把 YAML 标量转成字符串（数字端口等）。
func cstr(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

func ctruthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	case int:
		return t != 0
	}
	return false
}

func cmap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func firstString(v any) string {
	if arr, ok := v.([]any); ok {
		if len(arr) > 0 {
			return cstr(arr[0])
		}
		return ""
	}
	return cstr(v)
}

func joinAny(v any) string {
	arr, ok := v.([]any)
	if !ok {
		return cstr(v)
	}
	parts := make([]string, 0, len(arr))
	for _, x := range arr {
		parts = append(parts, cstr(x))
	}
	return strings.Join(parts, ",")
}

var digits = regexp.MustCompile(`\d+`)

// mbps 取带宽字符串里的数字（如 "100 Mbps" -> "100"）。
func mbps(v any) string { return digits.FindString(cstr(v)) }

// networkParams 把 ws-opts / grpc-opts / h2-opts / http-opts 展平成分享链接的 type/path/host/serviceName 参数。
func networkParams(q url.Values, p map[string]any) {
	net := cstr(p["network"])
	if net == "" {
		net = "tcp"
	}
	q.Set("type", net)
	switch net {
	case "ws":
		o := cmap(p["ws-opts"])
		if v := cstr(o["path"]); v != "" {
			q.Set("path", v)
		}
		if h := cmap(o["headers"]); h != nil {
			if v := cstr(h["Host"]); v != "" {
				q.Set("host", v)
			}
		}
	case "grpc":
		if v := cstr(cmap(p["grpc-opts"])["grpc-service-name"]); v != "" {
			q.Set("serviceName", v)
		}
	case "h2":
		o := cmap(p["h2-opts"])
		if v := cstr(o["path"]); v != "" {
			q.Set("path", v)
		}
		if v := firstString(o["host"]); v != "" {
			q.Set("host", v)
		}
	}
}

func buildLink(scheme, user, host, port string, q url.Values, name string, withUser bool) string {
	u := &url.URL{Scheme: scheme, Host: net.JoinHostPort(host, port), RawQuery: q.Encode(), Fragment: name}
	if withUser {
		u.User = url.User(user)
	}
	return u.String()
}

// proxyToLink 把 Clash 节点转换为分享链接；不支持的类型返回空串。
func proxyToLink(p map[string]any) string {
	typ := strings.ToLower(cstr(p["type"]))
	server, port, name := cstr(p["server"]), cstr(p["port"]), cstr(p["name"])
	if server == "" || port == "" {
		return ""
	}
	sni := cstr(p["sni"])
	if sni == "" {
		sni = cstr(p["servername"])
	}
	q := url.Values{}
	insecure := ctruthy(p["skip-cert-verify"])

	switch typ {
	case "vless":
		security := "none"
		if ro := cmap(p["reality-opts"]); ro != nil {
			security = "reality"
			q.Set("pbk", cstr(ro["public-key"]))
			q.Set("sid", cstr(ro["short-id"]))
		} else if ctruthy(p["tls"]) {
			security = "tls"
		}
		q.Set("security", security)
		if v := cstr(p["client-fingerprint"]); v != "" {
			q.Set("fp", v)
		}
		networkParams(q, p)
		if v := cstr(p["flow"]); v != "" {
			q.Set("flow", v)
		}
		if sni != "" {
			q.Set("sni", sni)
		}
		if insecure {
			q.Set("allowInsecure", "1")
		}
		return buildLink("vless", cstr(p["uuid"]), server, port, q, name, true)

	case "trojan":
		if v := cstr(p["client-fingerprint"]); v != "" {
			q.Set("fp", v)
		}
		networkParams(q, p)
		if sni != "" {
			q.Set("sni", sni)
		}
		if insecure {
			q.Set("allowInsecure", "1")
		}
		return buildLink("trojan", cstr(p["password"]), server, port, q, name, true)

	case "anytls":
		if sni != "" {
			q.Set("sni", sni)
		}
		if insecure {
			q.Set("insecure", "1")
		}
		return buildLink("anytls", cstr(p["password"]), server, port, q, name, true)

	case "hysteria2", "hy2":
		if sni != "" {
			q.Set("sni", sni)
		}
		if insecure {
			q.Set("insecure", "1")
		}
		if v := cstr(p["obfs"]); v != "" {
			q.Set("obfs", v)
			q.Set("obfs-password", cstr(p["obfs-password"]))
		}
		if v := cstr(p["ports"]); v != "" {
			q.Set("mport", v)
		}
		return buildLink("hysteria2", cstr(p["password"]), server, port, q, name, true)

	case "hysteria":
		if v := mbps(p["up"]); v != "" {
			q.Set("upmbps", v)
		}
		if v := mbps(p["down"]); v != "" {
			q.Set("downmbps", v)
		}
		if v := cstr(p["obfs"]); v != "" {
			q.Set("obfs", v)
		}
		if v := joinAny(p["alpn"]); v != "" {
			q.Set("alpn", v)
		}
		if sni != "" {
			q.Set("peer", sni)
		}
		if insecure {
			q.Set("insecure", "1")
		}
		return buildLink("hysteria", "", server, port, q, name, false)

	case "tuic":
		if sni != "" {
			q.Set("sni", sni)
		}
		if v := joinAny(p["alpn"]); v != "" {
			q.Set("alpn", v)
		}
		cc := cstr(p["congestion-controller"])
		if cc == "" {
			cc = cstr(p["congestion-control"])
		}
		if cc != "" {
			q.Set("congestion_control", cc)
		}
		if v := cstr(p["udp-relay-mode"]); v != "" {
			q.Set("udp_relay_mode", v)
		}
		if insecure {
			q.Set("allow_insecure", "1")
		}
		u := &url.URL{Scheme: "tuic", Host: net.JoinHostPort(server, port), RawQuery: q.Encode(), Fragment: name,
			User: url.UserPassword(cstr(p["uuid"]), cstr(p["password"]))}
		return u.String()

	case "ss":
		cred := base64.RawURLEncoding.EncodeToString([]byte(cstr(p["cipher"]) + ":" + cstr(p["password"])))
		l := "ss://" + cred + "@" + net.JoinHostPort(server, port)
		if name != "" {
			l += "#" + url.PathEscape(name)
		}
		return l

	case "vmess":
		d := map[string]any{"v": "2", "ps": name, "add": server, "port": port, "id": cstr(p["uuid"]), "net": cstr(p["network"])}
		if d["net"] == "" {
			d["net"] = "tcp"
		}
		if ctruthy(p["tls"]) {
			d["tls"] = "tls"
		}
		if sni != "" {
			d["sni"] = sni
		}
		if v := cstr(p["client-fingerprint"]); v != "" {
			d["fp"] = v
		}
		switch cstr(p["network"]) {
		case "ws":
			o := cmap(p["ws-opts"])
			d["path"] = cstr(o["path"])
			if h := cmap(o["headers"]); h != nil {
				d["host"] = cstr(h["Host"])
			}
		case "grpc":
			d["path"] = cstr(cmap(p["grpc-opts"])["grpc-service-name"])
		case "h2":
			o := cmap(p["h2-opts"])
			d["path"] = cstr(o["path"])
			d["host"] = firstString(o["host"])
		}
		b, _ := json.Marshal(d)
		return "vmess://" + base64.StdEncoding.EncodeToString(b)
	}
	return ""
}
