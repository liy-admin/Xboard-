package sub

import (
	"net/url"
	"strings"
	"testing"

	"xboard/internal/model"
)

func TestShareLinkVlessReality(t *testing.T) {
	n := model.Server{
		ID: 7, Type: "vless", Name: "US 1", Host: "node.example.com", Port: "443", ServerPort: 443,
		ProtocolSettings: []byte(`{"tls":2,"flow":"xtls-rprx-vision","network":"tcp",
			"reality_settings":{"server_name":"www.apple.com","public_key":"PUB","short_id":"abcd","private_key":"PRIV"}}`),
	}
	link := ShareLink(n, n, &model.User{UUID: "u-1"})
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Scheme != "vless" || u.User.Username() != "u-1" || u.Host != "node.example.com:443" {
		t.Fatalf("unexpected link %s", link)
	}
	for k, v := range map[string]string{"security": "reality", "sni": "www.apple.com", "pbk": "PUB", "sid": "abcd", "flow": "xtls-rprx-vision"} {
		if q.Get(k) != v {
			t.Errorf("%s = %q, want %q (%s)", k, q.Get(k), v, link)
		}
	}
	if q.Get("pbk") == "PRIV" || strings.Contains(link, "PRIV") {
		t.Fatalf("private key leaked into link: %s", link)
	}
}

func TestShareLinkUnknownType(t *testing.T) {
	if l := ShareLink(model.Server{Type: "mieru", Port: "1"}, model.Server{}, &model.User{UUID: "u"}); l != "" {
		t.Fatalf("expected empty link, got %q", l)
	}
}
