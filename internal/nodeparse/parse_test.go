package nodeparse

import (
	"encoding/base64"
	"testing"
)

func TestParseTextSkipsCommentsAndReportsUnknown(t *testing.T) {
	res := ParseText("# c\n\nfoo://bar\ntrojan://pw@example.com:443?sni=a.com#HK\n")
	if len(res.Servers) != 1 || len(res.Errors) != 1 {
		t.Fatalf("servers=%d errors=%d", len(res.Servers), len(res.Errors))
	}
	if res.Errors[0].Line != 3 {
		t.Fatalf("error line %d", res.Errors[0].Line)
	}
	s := res.Servers[0]
	if s.Type != "trojan" || s.Name != "HK" || s.Host != "example.com" || s.Port != 443 {
		t.Fatalf("unexpected %+v", s)
	}
}

func TestParseSSSIP002AndLegacy(t *testing.T) {
	userinfo := base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:pw"))
	s, ok := parseSS("ss://" + userinfo + "@1.2.3.4:8388#JP")
	if !ok || s.ProtocolSettings["cipher"] != "aes-256-gcm" || s.Host != "1.2.3.4" || s.Name != "JP" {
		t.Fatalf("sip002: %+v ok=%v", s, ok)
	}
	legacy := base64.StdEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:pw@5.6.7.8:9000"))
	s, ok = parseSS("ss://" + legacy)
	if !ok || s.Host != "5.6.7.8" || s.Port != 9000 {
		t.Fatalf("legacy: %+v ok=%v", s, ok)
	}
}

func TestParseVmessJSON(t *testing.T) {
	raw := `{"v":"2","ps":"US","add":"u.example.com","port":"8443","id":"0f0b0c0d-1111-2222-3333-444444444444","net":"ws","tls":"tls","host":"cdn.example.com","path":"/p"}`
	s, ok := parseLine("vmess://" + base64.StdEncoding.EncodeToString([]byte(raw)))
	if !ok || s.Name != "US" || s.Port != 8443 || s.Code == nil {
		t.Fatalf("vmess: %+v ok=%v", s, ok)
	}
	if s.ProtocolSettings["tls"] != 1 {
		t.Fatalf("tls flag: %v", s.ProtocolSettings["tls"])
	}
}

func TestSplitHostPortIPv6(t *testing.T) {
	h, p := splitHostPort("[2001:db8::1]:443")
	if h != "2001:db8::1" || p != "443" {
		t.Fatalf("%s %s", h, p)
	}
}

func TestTryDecodeBase64Subscription(t *testing.T) {
	body := base64.StdEncoding.EncodeToString([]byte("trojan://pw@h.example:443"))
	if got := TryDecodeBase64(body); got != "trojan://pw@h.example:443" {
		t.Fatalf("decode: %q", got)
	}
}

func TestParseAnyTLS(t *testing.T) {
	res := ParseText("anytls://0f0b0c0d-1111-2222-3333-444444444444@a.example.com:8443?sni=s.example.com&insecure=1#HK-01")
	if len(res.Servers) != 1 || len(res.Errors) != 0 {
		t.Fatalf("servers=%d errors=%+v", len(res.Servers), res.Errors)
	}
	s := res.Servers[0]
	tls, _ := s.ProtocolSettings["tls"].(map[string]any)
	if s.Type != "anytls" || s.Name != "HK-01" || s.Port != 8443 || s.Code == nil || tls["server_name"] != "s.example.com" || tls["allow_insecure"] != true {
		t.Fatalf("unexpected %+v tls=%v", s, tls)
	}
	// 非 UUID 的密码不应被当作账号 code
	s2, ok := parseLine("anytls://plain-password@b.example.com:443#X")
	if !ok || s2.Code != nil {
		t.Fatalf("plain password: %+v ok=%v", s2, ok)
	}
}

func TestParseVlessRealityStoresTLSMode2(t *testing.T) {
	s, ok := parseLine("vless://0f0b0c0d-1111-2222-3333-444444444444@r.example.com:443?security=reality&type=tcp&flow=xtls-rprx-vision&sni=www.example.com&pbk=PUB&sid=ab#R")
	if !ok || s.ProtocolSettings["tls"] != 2 || s.ProtocolSettings["reality_settings"] == nil {
		t.Fatalf("reality must be tls=2: %+v", s.ProtocolSettings)
	}
	s, ok = parseLine("vless://0f0b0c0d-1111-2222-3333-444444444444@t.example.com:443?security=tls&type=ws&sni=t.example.com#T")
	if !ok || s.ProtocolSettings["tls"] != 1 || s.ProtocolSettings["reality_settings"] != nil {
		t.Fatalf("plain tls must stay 1: %+v", s.ProtocolSettings)
	}
	s, ok = parseLine("vless://0f0b0c0d-1111-2222-3333-444444444444@n.example.com:80?security=none&type=tcp#N")
	if !ok || s.ProtocolSettings["tls"] != 0 {
		t.Fatalf("no security must be 0: %+v", s.ProtocolSettings)
	}
}

func TestParseUTLSFingerprint(t *testing.T) {
	s, _ := parseLine("vless://0f0b0c0d-1111-2222-3333-444444444444@r.example.com:443?security=reality&type=tcp&pbk=P&sni=a.com&fp=ios#R")
	u, _ := s.ProtocolSettings["utls"].(map[string]any)
	if u["enabled"] != true || u["fingerprint"] != "ios" {
		t.Fatalf("fp must be kept: %v", s.ProtocolSettings["utls"])
	}
	// Reality 没写 fp：sing-box 要求必须有 uTLS，默认 chrome
	s, _ = parseLine("vless://0f0b0c0d-1111-2222-3333-444444444444@r.example.com:443?security=reality&type=tcp&pbk=P&sni=a.com#R")
	u, _ = s.ProtocolSettings["utls"].(map[string]any)
	if u["fingerprint"] != "chrome" {
		t.Fatalf("reality default fingerprint: %v", s.ProtocolSettings["utls"])
	}
	// 普通 TLS 没写 fp：不强加 uTLS
	s, _ = parseLine("vless://0f0b0c0d-1111-2222-3333-444444444444@t.example.com:443?security=tls&type=ws&sni=t.com#T")
	if s.ProtocolSettings["utls"] != nil {
		t.Fatalf("plain tls must not get utls: %v", s.ProtocolSettings["utls"])
	}
	s, _ = parseLine("trojan://pw@t.example.com:443?sni=t.com&fp=firefox#T")
	if u, _ := s.ProtocolSettings["utls"].(map[string]any); u["fingerprint"] != "firefox" {
		t.Fatalf("trojan fp: %v", s.ProtocolSettings["utls"])
	}
}

func TestClashClientFingerprintBecomesUTLS(t *testing.T) {
	res := ParseClashYAML("proxies:\n  - {name: R, type: vless, server: r.example.com, port: 443, uuid: 0f0b0c0d-1111-2222-3333-444444444444, tls: true, network: tcp, servername: a.com, client-fingerprint: safari, reality-opts: {public-key: P, short-id: ab}}\n")
	if len(res.Servers) != 1 {
		t.Fatalf("%+v", res)
	}
	ps := res.Servers[0].ProtocolSettings
	if u, _ := ps["utls"].(map[string]any); u["fingerprint"] != "safari" || ps["tls"] != 2 {
		t.Fatalf("clash reality: %+v", ps)
	}
}
