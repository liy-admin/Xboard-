package sub

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"xboard/internal/model"
)

func clashFixture() []Map {
	var vm, ss Map
	_ = yamlJSON(`{"tls":1,"network":"ws","tls_settings":{"server_name":"v.example","allow_insecure":true},
	 "network_settings":{"path":"/p","headers":{"Host":"v.example"}}}`, &vm)
	_ = yamlJSON(`{"cipher":"aes-256-gcm","plugin":"v2ray-plugin","plugin_opts":"mode=websocket;tls;host=h.example"}`, &ss)
	return []Map{
		{"id": 1, "type": "vmess", "name": "HK-01", "host": "1.1.1.1", "port": 443, "password": "u1", "protocol_settings": vm},
		{"id": 2, "type": "shadowsocks", "name": "JP-01", "host": "2.2.2.2", "port": 8388, "password": "p", "protocol_settings": ss},
	}
}

func yamlJSON(s string, out any) error { return yaml.Unmarshal([]byte(s), out) }

func TestClashOutputs(t *testing.T) {
	for _, kind := range []ClashKind{ClashKindClash, ClashKindMeta, ClashKindStash} {
		ctx := &Ctx{User: &model.User{}, Servers: clashFixture(), AppName: "XB", AppURL: "https://x", Host: "sub.example"}
		out, err := BuildClash(ctx, kind)
		if err != nil {
			t.Fatalf("kind %d: %v", kind, err)
		}
		s := string(out.Body)
		if !strings.Contains(s, "HK-01") || !strings.Contains(s, "DOMAIN,sub.example,DIRECT") {
			t.Errorf("kind %d missing proxy or host rule:\n%s", kind, s)
		}
		if strings.Contains(s, "$app_name") {
			t.Errorf("kind %d: $app_name not substituted", kind)
		}
		var parsed map[string]any
		if err := yaml.Unmarshal(out.Body, &parsed); err != nil {
			t.Errorf("kind %d: output is not valid yaml: %v", kind, err)
		}
	}
}

func TestClashSSCipherFilter(t *testing.T) {
	var ss Map
	_ = yamlJSON(`{"cipher":"2022-blake3-aes-256-gcm"}`, &ss)
	ctx := &Ctx{User: &model.User{}, Servers: []Map{{"id": 3, "type": "shadowsocks", "name": "X", "host": "h", "port": 1, "password": "p", "protocol_settings": ss}}}
	out, _ := BuildClash(ctx, ClashKindClash)
	if strings.Contains(string(out.Body), "name: X") {
		t.Error("clash must drop non-classic ss ciphers")
	}
}

func TestFillGroupRegex(t *testing.T) {
	g := Map{"name": "SELECT", "proxies": []any{"DIRECT", "/^HK/i", "自动"}}
	fillGroup(g, []string{"HK-01", "JP-01", "hk-02"})
	got := toStrings(g["proxies"])
	want := []string{"DIRECT", "自动", "HK-01", "hk-02"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %v want %v", got, want)
	}
	plain := Map{"proxies": []any{"A"}}
	fillGroup(plain, []string{"N1", "N2"})
	if len(toStrings(plain["proxies"])) != 3 {
		t.Error("non-regex group should append all nodes")
	}
}
