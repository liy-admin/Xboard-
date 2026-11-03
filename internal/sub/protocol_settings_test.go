package sub

import (
	"encoding/json"
	"strings"
	"testing"
)

func norm(t *testing.T, typ, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatal(err)
		}
	}
	return NormalizeProtocolSettings(typ, m)
}

func TestNormalizeAnyTLSFillsDefaults(t *testing.T) {
	ps := norm(t, "anytls", `{"tls":{"server_name":"a.com"}}`)
	scheme, _ := ps["padding_scheme"].([]any)
	if len(scheme) != 9 || scheme[0] != "stop=8" {
		t.Fatalf("padding_scheme default: %v", ps["padding_scheme"])
	}
	tls, _ := ps["tls"].(map[string]any)
	if tls["server_name"] != "a.com" || tls["allow_insecure"] != false {
		t.Fatalf("tls: %v", tls)
	}
	// 对象字段缺失时与 PHP 一致返回 null（不会凭空生成子字段）。
	if v, present := tls["ech"]; !present || v != nil {
		t.Fatalf("missing ech object must be present as null: %#v present=%v", v, present)
	}
}

func TestNormalizeVlessRealityDefaultsAndDropsUnknown(t *testing.T) {
	ps := norm(t, "vless", `{"tls":"2","reality_settings":{"public_key":"PK"},"bogus":1,"flow":"xtls-rprx-vision"}`)
	if Int(ps["tls"]) != 2 {
		t.Fatalf("tls must be cast to int: %#v", ps["tls"])
	}
	if _, ok := ps["bogus"]; ok {
		t.Fatal("undeclared fields must be dropped, as PHP does")
	}
	rs, _ := ps["reality_settings"].(map[string]any)
	if rs["public_key"] != "PK" || rs["allow_insecure"] != false {
		t.Fatalf("reality: %v", rs)
	}
	if v, present := ps["utls"]; !present || v != nil {
		t.Fatalf("missing utls object must be present as null (PHP behaviour): %#v present=%v", v, present)
	}
	// 给了 utls 但没写指纹时，指纹默认 chrome。
	ps2 := norm(t, "vless", `{"utls":{"enabled":true}}`)
	if u, _ := ps2["utls"].(map[string]any); u["fingerprint"] != "chrome" || u["enabled"] != true {
		t.Fatalf("utls fingerprint default: %v", ps2["utls"])
	}
}

func TestNormalizeHysteriaAndTuicDefaults(t *testing.T) {
	h := norm(t, "hysteria", `{}`)
	if Int(h["version"]) != 2 {
		t.Fatalf("hysteria version default 2, got %#v", h["version"])
	}
	if v, present := h["obfs"]; !present || v != nil {
		t.Fatalf("missing obfs object must be null: %#v", v)
	}
	h2 := norm(t, "hysteria", `{"obfs":{"open":1}}`)
	if o, _ := h2["obfs"].(map[string]any); o["type"] != "salamander" || o["open"] != true {
		t.Fatalf("obfs sub-field defaults: %v", h2["obfs"])
	}
	tu := norm(t, "tuic", ``)
	if Int(tu["version"]) != 5 || tu["congestion_control"] != "cubic" || tu["udp_relay_mode"] != "native" {
		t.Fatalf("tuic defaults: %v", tu)
	}
	if a, _ := tu["alpn"].([]any); len(a) != 1 || a[0] != "h3" {
		t.Fatalf("alpn default: %v", tu["alpn"])
	}
}

func TestNormalizeUnknownTypePassesThrough(t *testing.T) {
	in := map[string]any{"x": 1}
	if out := NormalizeProtocolSettings("not-a-protocol", in); out["x"] != 1 {
		t.Fatalf("unknown type must not be altered: %v", out)
	}
}

// 管理端保存节点时请求体用 UseNumber 解析，数字是 json.Number。
// 之前 phpInt 不认 json.Number，编辑节点会把 tls 2 改成 0（Reality / TLS 被关掉）。
func TestNormalizeAcceptsJSONNumber(t *testing.T) {
	dec := json.NewDecoder(bytesReader(`{
		"tls": 2, "flow": "xtls-rprx-vision", "network": "tcp",
		"reality_settings": {"server_name": "dl.google.com", "server_port": 443, "allow_insecure": 0,
		                     "public_key": "pk", "private_key": "sk", "short_id": "ab"}
	}`))
	dec.UseNumber()
	var raw map[string]any
	if err := dec.Decode(&raw); err != nil {
		t.Fatal(err)
	}
	ps := NormalizeProtocolSettings("vless", raw)
	if ps["tls"] != int64(2) {
		t.Fatalf("tls must stay 2, got %#v", ps["tls"])
	}
	rs := ps["reality_settings"].(map[string]any)
	if rs["server_port"] != "443" || rs["allow_insecure"] != false {
		t.Fatalf("reality settings mis-cast: %#v", rs)
	}

	dec = json.NewDecoder(bytesReader(`{"version": 2, "obfs": {"open": 1}}`))
	dec.UseNumber()
	raw = nil
	if err := dec.Decode(&raw); err != nil {
		t.Fatal(err)
	}
	hy := NormalizeProtocolSettings("hysteria", raw)
	if hy["version"] != int64(2) || hy["obfs"].(map[string]any)["open"] != true {
		t.Fatalf("hysteria mis-cast: %#v", hy)
	}
}

func TestTruthyJSONNumber(t *testing.T) {
	for in, want := range map[string]bool{"0": false, "0.0": false, "1": true, "-2": true} {
		if got := Truthy(json.Number(in)); got != want {
			t.Errorf("Truthy(json.Number(%q)) = %v, want %v", in, got, want)
		}
	}
}

func bytesReader(s string) *strings.Reader { return strings.NewReader(s) }
