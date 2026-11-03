package sub

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"xboard/internal/model"
)

func protoCtx(servers ...Map) *Ctx {
	exp := int64(1900000000)
	return &Ctx{
		User:    &model.User{Token: "tok", U: 1 << 30, D: 2 << 30, TransferEnab: 10 << 30, ExpiredAt: &exp},
		Servers: servers,
		AppName: "XBoard",
		SubscribeURL: func(token, domain string) string {
			return "https://example.test/s/" + token
		},
	}
}

func ssServer() Map {
	return Map{
		"id": 7, "type": "shadowsocks", "name": "HK 1", "host": "1.2.3.4", "port": 8388,
		"password": "pw", "protocol_settings": Map{"cipher": "aes-256-gcm"},
	}
}

func TestSurgeShadowsocksLine(t *testing.T) {
	line := buildSurgeSS("pw", ssServer())
	want := "HK 1 = ss,1.2.3.4,8388,encrypt-method=aes-256-gcm,password=pw,tfo=true,udp-relay=true\r\n"
	if line != want {
		t.Fatalf("got %q want %q", line, want)
	}
}

func TestSurfboardShadowsocksUsesNoSpaceSeparator(t *testing.T) {
	line := buildSurfboardSS("pw", ssServer())
	if !strings.HasPrefix(line, "HK 1=ss,") {
		t.Fatalf("surfboard line format wrong: %q", line)
	}
}

func TestSurgeOutputContainsProxyGroupAndInfo(t *testing.T) {
	body, ct, h := BuildSurge(protoCtx(ssServer()))
	if ct != "application/octet-stream" || !strings.Contains(h["content-disposition"], "XBoard.conf") {
		t.Fatalf("headers wrong: %s %v", ct, h)
	}
	if !strings.Contains(body, "HK 1 = ss,1.2.3.4") {
		t.Fatal("proxy line missing")
	}
	if !strings.Contains(body, "https://example.test/s/tok") {
		t.Fatal("subscribe link not substituted")
	}
	if !strings.Contains(body, "上传流量：1GB") || !strings.Contains(body, "套餐流量：10GB") {
		t.Fatal("subscribe info not substituted")
	}
	if strings.Contains(body, "$proxies") || strings.Contains(body, "$proxy_group") {
		t.Fatal("placeholders left unreplaced")
	}
}

func TestLoonShadowsocksLine(t *testing.T) {
	line := buildLoonSS("pw", ssServer())
	want := "HK 1=Shadowsocks,1.2.3.4,8388,aes-256-gcm,pw,fast-open=false,udp=true\r\n"
	if line != want {
		t.Fatalf("got %q want %q", line, want)
	}
}

func TestLoonHysteriaWithoutSNIOmitsNullLiteral(t *testing.T) {
	s := Map{"name": "H", "host": "h", "port": 443, "type": "hysteria",
		"protocol_settings": Map{"version": 2, "tls": Map{}}}
	line := buildLoonHysteria("pw", s)
	if strings.Contains(line, "(null)") {
		t.Fatalf("PHP placeholder leaked: %q", line)
	}
}

func TestQuantumultXIsBase64Subscription(t *testing.T) {
	body, _, h := BuildQuantumultX(protoCtx(ssServer()))
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(raw), "shadowsocks=1.2.3.4:8388,method=aes-256-gcm,password=pw,fast-open=true,udp-relay=true,tag=HK 1") {
		t.Fatalf("quanx line: %q", raw)
	}
	if !strings.Contains(h["subscription-userinfo"], "upload=1073741824; download=2147483648; total=10737418240") {
		t.Fatalf("userinfo: %s", h["subscription-userinfo"])
	}
}

func TestShadowrocketStatusLineAndBase64(t *testing.T) {
	body, _, _ := BuildShadowrocket(protoCtx(ssServer()))
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		t.Fatal(err)
	}
	first := strings.SplitN(string(raw), "\r\n", 2)[0]
	if !strings.HasPrefix(first, "STATUS=🚀↑:1GB,↓:2GB,TOT:10GB💡Expires:") {
		t.Fatalf("status line: %q", first)
	}
	if !strings.Contains(string(raw), "ss://") {
		t.Fatal("ss uri missing")
	}
}

func TestSIP008Shape(t *testing.T) {
	body, ct := BuildSIP008(protoCtx(ssServer()))
	if ct != "application/json" {
		t.Fatalf("content type %s", ct)
	}
	var got struct {
		Servers []map[string]any `json:"servers"`
		Version int              `json:"version"`
		Used    int64            `json:"bytes_used"`
	}
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || len(got.Servers) != 1 || got.Used != 3<<30 {
		t.Fatalf("unexpected %+v", got)
	}
	if got.Servers[0]["method"] != "aes-256-gcm" || got.Servers[0]["server_port"].(float64) != 8388 {
		t.Fatalf("server fields: %v", got.Servers[0])
	}
	if strings.Index(body, `"servers"`) > strings.Index(body, `"version"`) {
		t.Fatal("key order should mirror PHP: servers first, version last")
	}
}

func TestSIP008SkipsUnsupportedCipher(t *testing.T) {
	bad := ssServer()
	bad["protocol_settings"] = Map{"cipher": "2022-blake3-aes-128-gcm"}
	body, _ := BuildSIP008(protoCtx(bad))
	if !strings.Contains(body, `"servers":[]`) {
		t.Fatalf("unsupported cipher should be dropped: %s", body)
	}
}

func TestPluginOptsParsing(t *testing.T) {
	got := pluginOpts("obfs=http;obfs-host=example.com; path = /x ;junk;")
	if got["obfs"] != "http" || got["obfs-host"] != "example.com" || got["path"] != "/x" {
		t.Fatalf("parsed %v", got)
	}
	if _, ok := got["junk"]; ok {
		t.Fatal("segment without '=' must be ignored")
	}
}

func TestOrderedQueryEncoding(t *testing.T) {
	q := newQuery().set("a", 1).set("b", "x y").set("c", nil).set("a", true)
	if got := q.encode(true); got != "a=1&b=x%20y" {
		t.Fatalf("rfc3986 encode: %q", got)
	}
	if got := q.encode(false); got != "a=1&b=x+y" {
		t.Fatalf("rfc1738 encode: %q", got)
	}
}
