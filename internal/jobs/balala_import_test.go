package jobs

import (
	"reflect"
	"testing"
)

func TestIsInfoNode(t *testing.T) {
	cases := []struct {
		name, extra string
		want        bool
	}{
		{"香港 01", "", false},
		{"剩余流量：100GB", "", true},
		{"距离到期还有 10 天", "", true},
		{"https://example.com", "", true},
		{"example.com 官网", "", true},
		{"Tokyo Fast", "", false},
		{"Tokyo Promo", "promo", true},
		{"", "", false},
	}
	for _, c := range cases {
		if got := isInfoNode(c.name, c.extra); got != c.want {
			t.Errorf("isInfoNode(%q,%q)=%v want %v", c.name, c.extra, got, c.want)
		}
	}
}

func TestParseGroupIDs(t *testing.T) {
	if got := parseGroupIDs("1, 2 3,,x"); !reflect.DeepEqual(got, []int64{1, 2, 3}) {
		t.Fatalf("string: %v", got)
	}
	if got := parseGroupIDs([]any{float64(4), "5"}); !reflect.DeepEqual(got, []int64{4, 5}) {
		t.Fatalf("array: %v", got)
	}
	if got := parseGroupIDs(""); len(got) != 0 {
		t.Fatalf("empty: %v", got)
	}
}

func TestPickBalalaURL(t *testing.T) {
	r := BalalaResult{SubscribeURL: "main", Subscriptions: []map[string]any{
		{"client": "Clash", "url": "clash-url"}, {"client": "V2RayN", "url": "v2-url"},
	}}
	if got := pickBalalaURL(r, "v2rayn"); got != "v2-url" {
		t.Fatalf("got %s", got)
	}
	if got := pickBalalaURL(r, "surge"); got != "main" {
		t.Fatalf("fallback got %s", got)
	}
}
