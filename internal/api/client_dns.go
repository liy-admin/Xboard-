package api

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// 极速模式（分流）下 App 的直连 DNS：中国域名、直连规则命中的域名都用它解析。
// 默认用阿里 223.5.5.5（与 v2rayN 一致）：用 1.1.1.1 这类境外 DNS 解析国内域名，
// 会拿到境外 CDN 节点（如 qq.com 解析到香港），直连反而绕远。
// 空串表示不覆盖，沿用 App 本地设置；安全模式（全局）下 App 不使用此值。

const (
	settingClientDirectDNS = "client_direct_dns"
	DefaultClientDirectDNS = "223.5.5.5"
)

// ValidateClientDirectDNS 按 hiddify-core（getDNSServerOptions）支持的写法校验：
// local、IP[:端口]、udp:// / tcp:// + IP[:端口]、tls:// / quic:// / https:// / h3:// + 主机。
// 写错会让 App 内核启动失败，所以这里从严。
func ValidateClientDirectDNS(v string) error {
	v = strings.TrimSpace(v)
	if v == "" || v == "local" {
		return nil
	}
	if !strings.Contains(v, "://") {
		if !isIPHostPort(v) {
			return fmt.Errorf("直连 DNS 需填写 IP（如 223.5.5.5）或带协议的地址（如 https://dns.alidns.com/dns-query）")
		}
		return nil
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" {
		return fmt.Errorf("直连 DNS 地址格式不正确：%q", v)
	}
	switch u.Scheme {
	case "udp", "tcp":
		if !isIPHostPort(u.Host) {
			return fmt.Errorf("%s:// 后需填写 IP（如 udp://223.5.5.5）", u.Scheme)
		}
	case "tls", "quic", "https", "h3":
	default:
		return fmt.Errorf("直连 DNS 协议只支持 udp / tcp / tls / quic / https / h3，当前为 %q", u.Scheme)
	}
	return nil
}

// isIPHostPort 判断 "IP" 或 "IP:端口"（IPv6 带端口需写成 [::1]:53）。
func isIPHostPort(s string) bool {
	if net.ParseIP(s) != nil {
		return true
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil || net.ParseIP(host) == nil {
		return false
	}
	p, err := strconv.Atoi(port)
	return err == nil && p > 0 && p < 65536
}

// clientDirectDNS 取下发给 App 的直连 DNS：从未设置过时用默认值；显式清空则为空（App 用本地设置）。
func (a *App) clientDirectDNS() string {
	if !a.Set.Has(settingClientDirectDNS) {
		return DefaultClientDirectDNS
	}
	v := strings.TrimSpace(a.Set.Get(settingClientDirectDNS, ""))
	if ValidateClientDirectDNS(v) != nil {
		return DefaultClientDirectDNS
	}
	return v
}
