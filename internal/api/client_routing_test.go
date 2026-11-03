package api

import (
	"strings"
	"testing"
)

func TestParseClientRoutingRules(t *testing.T) {
	rules, err := ParseClientRoutingRules(`[
		{"name":"google cn","outbound":"Proxy","domain_suffix":[" googleapis.cn ","googleapis.cn",""]},
		{"outbound":"direct","ip_cidr":["223.5.5.5/32"],"rule_set":["https://example.com/geosite-private.srs"]},
		{"domain_keyword":["openai"]}
	]`, DefaultRuleSetURLTemplate)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 {
		t.Fatalf("got %d rules", len(rules))
	}
	if rules[0].Outbound != "proxy" || len(rules[0].DomainSuffix) != 1 || rules[0].DomainSuffix[0] != "googleapis.cn" {
		t.Errorf("rule 0 not normalized: %+v", rules[0])
	}
	if rules[2].Outbound != "proxy" {
		t.Errorf("empty outbound should default to proxy, got %q", rules[2].Outbound)
	}

	if r, err := ParseClientRoutingRules("  ", ""); err != nil || len(r) != 0 {
		t.Errorf("empty input: %v %v", r, err)
	}

	bad := map[string]string{
		`{"outbound":"proxy"}`:                          "JSON 数组",
		`[{"outbound":"warp","domain":["a.com"]}]`:      "outbound",
		`[{"outbound":"proxy"}]`:                        "匹配条件",
		`[{"outbound":"proxy","network":"icmp"}]`:       "network",
		`[{"outbound":"proxy","rule_set":["cn"]}]`:      "rule_set",
		`[{"outbound":"proxy","domains":["typo.com"]}]`: "JSON",
	}
	for in, want := range bad {
		if _, err := ParseClientRoutingRules(in, DefaultRuleSetURLTemplate); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want containing %q", in, err, want)
		}
	}
}

func TestRuleSetShortNames(t *testing.T) {
	in := `[{"outbound":"direct","rule_set":["geosite-private","geoip-cn"]}]`
	if _, err := ParseClientRoutingRules(in, DefaultRuleSetURLTemplate); err != nil {
		t.Errorf("short names with template: %v", err)
	}
	if _, err := ParseClientRoutingRules(in, ""); err == nil || !strings.Contains(err.Error(), "模板") {
		t.Errorf("short names without template: err = %v", err)
	}
	for tpl, ok := range map[string]bool{
		"":                                  true,
		DefaultRuleSetURLTemplate:           true,
		"ftp://x/{name}.srs":                false,
		"https://x/rule-set/geosite-cn.srs": false,
	} {
		if err := ValidateRuleSetURLTemplate(tpl); (err == nil) != ok {
			t.Errorf("validate(%q) err = %v", tpl, err)
		}
	}
}
