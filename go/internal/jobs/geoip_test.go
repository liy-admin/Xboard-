package jobs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostToIPv4Literal(t *testing.T) {
	if ip, ok := hostToIPv4("1.2.3.4"); !ok || ip != "1.2.3.4" {
		t.Fatalf("ipv4 literal: got %q %v", ip, ok)
	}
	if _, ok := hostToIPv4("2001:db8::1"); ok {
		t.Fatalf("ipv6 literal must be rejected like gethostbyname")
	}
}

func TestLookupCountriesViaStubAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req []map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		out := []map[string]string{}
		for _, q := range req {
			if q["query"] == "8.8.8.8" {
				out = append(out, map[string]string{"status": "success", "countryCode": "us", "query": "8.8.8.8"})
			} else {
				out = append(out, map[string]string{"status": "fail", "query": q["query"]})
			}
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer srv.Close()
	old := ipAPIBatchURL
	ipAPIBatchURL = srv.URL
	defer func() { ipAPIBatchURL = old }()

	got := LookupCountries(context.Background(), []string{"8.8.8.8", "1.1.1.1", ""})
	if got["8.8.8.8"] != "US" {
		t.Fatalf("8.8.8.8 => %q, want US", got["8.8.8.8"])
	}
	if got["1.1.1.1"] != "" {
		t.Fatalf("failed lookup must stay empty, got %q", got["1.1.1.1"])
	}
}

func TestBalalaTruthy(t *testing.T) {
	cases := map[string]struct {
		v    any
		want bool
	}{
		"nil":      {nil, false},
		"true":     {true, true},
		"zero str": {"0", false},
		"empty":    {"", false},
		"one str":  {"1", true},
		"float0":   {float64(0), false},
	}
	for name, c := range cases {
		if got := balalaTruthy(c.v); got != c.want {
			t.Errorf("%s: got %v want %v", name, got, c.want)
		}
	}
}

func TestBalalaScriptPathOverride(t *testing.T) {
	t.Setenv("XBOARD_BALALA_SCRIPT", "/tmp/x.py")
	if p := balalaScriptPath(); p != "/tmp/x.py" {
		t.Fatalf("override ignored: %s", p)
	}
}
