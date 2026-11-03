package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"xboard/internal/setting"
)

// 后台「路由管理 → 客户端分流」通过 config/fetch?key=client_routing 读取这三个键；
// 分组丢失时页面会把规则集来源误读为空（App 内置），这里锁住整个读取链路。
func TestConfigFetchClientRoutingSection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fetch := func(stored map[string]string) map[string]any {
		a := &App{Set: setting.NewFromMap(stored)}
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/config/fetch?key=client_routing", nil)
		a.configFetch(c)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			Data map[string]map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		sec, ok := body.Data["client_routing"]
		if !ok {
			t.Fatalf("client_routing section missing: %s", w.Body.String())
		}
		return sec
	}

	cases := []struct {
		name   string
		stored map[string]string
		want   string
	}{
		{"never saved -> Loyalsoldier", map[string]string{}, DefaultRuleSetURLTemplate},
		{"saved Loyalsoldier", map[string]string{settingRuleSetURLTemplate: DefaultRuleSetURLTemplate}, DefaultRuleSetURLTemplate},
		{"saved builtin (empty)", map[string]string{settingRuleSetURLTemplate: ""}, ""},
		{"saved custom", map[string]string{settingRuleSetURLTemplate: "https://m.example/{type}/{name}.srs"}, "https://m.example/{type}/{name}.srs"},
	}
	for _, tc := range cases {
		sec := fetch(tc.stored)
		if got := sec["client_ruleset_url_template"]; got != tc.want {
			t.Errorf("%s: template = %#v, want %q", tc.name, got, tc.want)
		}
		for _, k := range []string{"client_routing_enable", "client_routing_rules"} {
			if _, ok := sec[k]; !ok {
				t.Errorf("%s: key %s missing", tc.name, k)
			}
		}
	}
}

// 直连 DNS：读取链路（config/fetch 与 App 接口）必须原样返回字符串，
// 尤其 223.5.5.5 这类值会经过「按 JSON 解码设置值」的逻辑，不能被当成数字截断。
func TestClientDirectDNSReadBack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	get := func(a *App, target string, h gin.HandlerFunc) map[string]any {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, target, nil)
		h(c)
		var body struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: %v %s", target, err, w.Body.String())
		}
		return body.Data
	}
	cases := []struct {
		name   string
		stored map[string]string
		want   string
	}{
		{"never saved -> 223.5.5.5", map[string]string{}, DefaultClientDirectDNS},
		{"saved 223.5.5.5", map[string]string{settingClientDirectDNS: "223.5.5.5"}, "223.5.5.5"},
		{"saved 119.29.29.29", map[string]string{settingClientDirectDNS: "119.29.29.29"}, "119.29.29.29"},
		{"saved DoH", map[string]string{settingClientDirectDNS: "https://dns.alidns.com/dns-query"}, "https://dns.alidns.com/dns-query"},
		{"saved empty (app setting)", map[string]string{settingClientDirectDNS: ""}, ""},
	}
	for _, tc := range cases {
		a := &App{Set: setting.NewFromMap(tc.stored)}
		sec, _ := get(a, "/config/fetch?key=client_routing", a.configFetch)["client_routing"].(map[string]any)
		if got := sec["client_direct_dns"]; got != tc.want {
			t.Errorf("%s: admin fetch = %#v, want %q", tc.name, got, tc.want)
		}
		if got := get(a, "/api/v1/guest/client/routing", a.guestClientRouting)["direct_dns"]; got != tc.want {
			t.Errorf("%s: app api = %#v, want %q", tc.name, got, tc.want)
		}
	}
}

func TestValidateClientDirectDNS(t *testing.T) {
	for v, ok := range map[string]bool{
		"": true, "local": true, "223.5.5.5": true, "223.5.5.5:53": true, "[2400:3200::1]:53": true, "2400:3200::1": true,
		"udp://223.5.5.5": true, "tcp://119.29.29.29:53": true, "tls://dns.alidns.com": true,
		"https://dns.alidns.com/dns-query": true, "h3://dns.alidns.com/dns-query": true, "quic://dns.alidns.com": true,
		"dns.alidns.com": false, "udp://dns.alidns.com": false, "ftp://1.1.1.1": false, "https://": false, "223.5.5.5:99999": false,
	} {
		if err := ValidateClientDirectDNS(v); (err == nil) != ok {
			t.Errorf("validate(%q) err = %v, want ok=%v", v, err, ok)
		}
	}
}
