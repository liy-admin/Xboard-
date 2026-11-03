package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"xboard/internal/model"
	"xboard/internal/setting"
	"xboard/internal/sub"
)

func TestParseQuickNodeItemsLegacyDefaults(t *testing.T) {
	items, err := parseQuickNodeItems(map[string]any{})
	if err != nil || len(items) != 1 || items[0].preset.key != "vless_reality" || items[0].port != 443 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestParseQuickNodeItemsRejects(t *testing.T) {
	cases := []map[string]any{
		{"protocols": []any{map[string]any{"protocol": "nope"}}},
		{"protocols": []any{map[string]any{"protocol": "tuic"}, map[string]any{"protocol": "tuic"}}},
		{"protocols": []any{map[string]any{"protocol": "tuic", "port": json.Number("70000")}}},
		{"port": json.Number("0")},
	}
	for i, m := range cases {
		if _, err := parseQuickNodeItems(m); err == nil {
			t.Errorf("case %d: expected error", i)
		}
	}
}

func TestAssignQuickNodePorts(t *testing.T) {
	hy, _ := quickNodePresetByKey("hysteria2")
	vl, _ := quickNodePresetByKey("vless_reality")
	items := []quickNodeItem{{preset: vl}, {preset: hy, port: 8444}}
	used := map[int64]string{443: "节点 #1（a）"}
	if err := assignQuickNodePorts(items, used); err != nil {
		t.Fatal(err)
	}
	if items[0].port != 444 || items[1].port != 8444 {
		t.Fatalf("ports %d %d", items[0].port, items[1].port)
	}

	// 指定的端口被占用时报错，不会悄悄换端口
	if err := assignQuickNodePorts([]quickNodeItem{{preset: vl, port: 443}}, map[int64]string{443: "x"}); err == nil {
		t.Fatal("expected conflict")
	}
	// 自动分配要避开同一批里明确指定的端口
	items = []quickNodeItem{{preset: vl}, {preset: hy, port: 443}}
	if err := assignQuickNodePorts(items, map[int64]string{}); err != nil {
		t.Fatal(err)
	}
	if items[0].port != 444 {
		t.Fatalf("auto port %d", items[0].port)
	}
}

// 规范化不能丢掉节点端和客户端需要的字段，否则节点起不来或链接缺参数。
func TestQuickNodePresetsSurviveNormalize(t *testing.T) {
	for _, p := range quickNodePresets {
		settings, cert, err := p.build("www.apple.com", "node.example.com")
		if err != nil {
			t.Fatalf("%s: %v", p.key, err)
		}
		got := sub.NormalizeProtocolSettings(p.nodeType, settings)
		switch p.key {
		case "vless_reality", "trojan_reality":
			if rs, _ := got["reality_settings"].(map[string]any); rs["private_key"] == "" || rs["public_key"] == "" || fmt.Sprint(got["tls"]) != "2" {
				t.Errorf("%s: %v", p.key, got)
			}
		case "hysteria2", "tuic", "anytls":
			tls, _ := got["tls"].(map[string]any)
			if tls["allow_insecure"] != true || tls["server_name"] != "www.apple.com" || cert["cert_mode"] != "self" {
				t.Errorf("%s: %v cert=%v", p.key, got, cert)
			}
		case "shadowsocks":
			if got["cipher"] != "aes-128-gcm" {
				t.Errorf("%s: %v", p.key, got)
			}
		case "vmess_ws":
			ns, _ := got["network_settings"].(map[string]any)
			if got["network"] != "ws" || ns["path"] == "" {
				t.Errorf("%s: %v", p.key, got)
			}
		}
	}
}

// hysteria / tuic / anytls 的 tls 是对象，只能作为 tls_settings 下发：节点端的 tls 字段是整数，
// 透传对象会让节点解码失败、起不来。cert_config 必须下发，否则这些协议没有证书。
func TestNodeConfigMapObjectTLS(t *testing.T) {
	a := &App{Set: setting.NewFromMap(map[string]string{})}
	settings, cert, _ := buildHysteria2("www.apple.com", "h")
	ps, _ := json.Marshal(settings)
	certJSON, _ := json.Marshal(cert)
	got := a.nodeConfigMap(&model.Server{Type: "hysteria", Host: "h", ServerPort: 1, ProtocolSettings: ps, CertConfig: certJSON})
	if _, has := got["tls"]; has {
		t.Fatalf("tls must not be passed through: %v", got["tls"])
	}
	if ts, _ := got["tls_settings"].(map[string]any); ts["server_name"] != "www.apple.com" {
		t.Fatalf("tls_settings = %v", got["tls_settings"])
	}
	if c, _ := got["cert_config"].(map[string]any); c["cert_mode"] != "self" {
		t.Fatalf("cert_config = %v", got["cert_config"])
	}
}
