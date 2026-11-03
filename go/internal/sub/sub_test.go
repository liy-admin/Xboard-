package sub

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"xboard/internal/model"
)

func fixture() []Map {
	var vless, trojan, hy Map
	_ = json.Unmarshal([]byte(`{"network":"ws","tls":2,"flow":"xtls-rprx-vision",
	 "network_settings":{"path":"/ws","headers":{"Host":"h.example"}},
	 "reality_settings":{"server_name":"sni.example","public_key":"PK","short_id":"ab"}}`), &vless)
	_ = json.Unmarshal([]byte(`{"tls":1,"network":"tcp","tls_settings":{"server_name":"t.example","allow_insecure":false}}`), &trojan)
	_ = json.Unmarshal([]byte(`{"version":2,"tls":{"server_name":"hy.example"},"bandwidth":{"up":100,"down":200},"obfs":{"open":true,"type":"salamander","password":"p"}}`), &hy)
	return []Map{
		{"id": 1, "type": "vless", "name": "HK-1", "host": "1.2.3.4", "port": 443, "password": "uuid-1", "protocol_settings": vless},
		{"id": 2, "type": "trojan", "name": "JP-1", "host": "::1", "port": 8443, "password": "pw", "protocol_settings": trojan},
		{"id": 3, "type": "hysteria", "name": "US-1", "host": "h.example", "port": 1000, "ports": "1000-2000", "password": "hp", "protocol_settings": hy},
	}
}

func TestGeneral(t *testing.T) {
	ctx := &Ctx{User: &model.User{TransferEnab: 10}, Servers: fixture()}
	body, _, h := BuildGeneral(ctx)
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{
		"vless://uuid-1@1.2.3.4:443?", "security=reality", "pbk=PK", "type=ws", "path=%2Fws", "#HK-1",
		"trojan://pw@[::1]:8443?", "hysteria2://hp@h.example:1000?", "mport=1000-2000", "obfs=salamander",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in\n%s", want, s)
		}
	}
	if !strings.Contains(h["subscription-userinfo"], "total=10") {
		t.Error("userinfo header")
	}
}

func TestSingBox(t *testing.T) {
	ctx := &Ctx{User: &model.User{}, Servers: fixture(), ClientName: "sing-box", ClientVersion: "1.12.0",
		UserAgent: "sing-box 1.12.0", AppName: "X"}
	cfg, _, err := BuildSingBox(ctx)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(cfg)
	s := string(b)
	for _, want := range []string{`"tag":"HK-1"`, `"public_key":"PK"`, `"type":"hysteria2"`, `"server_ports":["1000:2000"]`} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s", want)
		}
	}
	// 1.12 < 1.13：block/dns 出站保留；>=1.12 DNS 保持新格式
	ctx.UserAgent, ctx.ClientVersion = "sing-box 1.9.0", "1.9.0"
	cfg, _, _ = BuildSingBox(ctx)
	b, _ = json.Marshal(cfg)
	if !strings.Contains(string(b), `"inet4_address"`) {
		t.Error("1.9 should use legacy tun address")
	}
}

func TestParseClient(t *testing.T) {
	ci := ParseClient("Hiddify/2.0.5 (android)")
	if ci.Name != "hiddify" || ci.Version != "2.0.5" {
		t.Errorf("%+v", ci)
	}
	if n, _ := MatchProtocol("sing-box 1.8"); n != "singbox" {
		t.Error("match")
	}
}
