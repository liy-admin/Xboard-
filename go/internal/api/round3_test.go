package api

import (
	"strings"
	"testing"
)

func TestShellQuoteEscapesSingleQuotes(t *testing.T) {
	if got := shellQuote("a'b"); got != `'a'\''b'` {
		t.Fatalf("got %s", got)
	}
}

func TestMaskEmailKeepsThreeChars(t *testing.T) {
	if got := maskEmail("alice@example.com"); got != "ali***@***" {
		t.Fatalf("got %s", got)
	}
	if got := maskEmail("ab"); got != "ab***@***" {
		t.Fatalf("short email: %s", got)
	}
}

func TestGrowthRounding(t *testing.T) {
	if got := growth(150, 100); got != 50 {
		t.Fatalf("want 50 got %v", got)
	}
	if got := growth(10, 0); got != 0 {
		t.Fatalf("zero base must be 0, got %v", got)
	}
	if got := growth(1, 3); got != -66.7 {
		t.Fatalf("want -66.7 got %v", got)
	}
}

func TestFormatMoneyGrouping(t *testing.T) {
	cases := map[float64]string{0: "0.00", 1234.5: "1,234.50", 1234567.891: "1,234,567.89", -9999.1: "-9,999.10"}
	for in, want := range cases {
		if got := formatMoney(in); got != want {
			t.Errorf("%v: want %s got %s", in, want, got)
		}
	}
}

func TestValidateMailContentRequiredVars(t *testing.T) {
	if errs := validateMailContent("verify", "no code here"); len(errs) != 1 {
		t.Fatalf("missing {{code}} should be reported: %v", errs)
	}
	if errs := validateMailContent("verify", "your code {{code}}"); len(errs) != 0 {
		t.Fatalf("valid content rejected: %v", errs)
	}
}

func TestPEMBlockWrapsAt64(t *testing.T) {
	out := pemBlock("ECH KEYS", make([]byte, 100))
	lines := strings.Split(out, "\n")
	if lines[0] != "-----BEGIN ECH KEYS-----" || lines[len(lines)-1] != "-----END ECH KEYS-----" {
		t.Fatalf("framing wrong: %q", out)
	}
	for _, l := range lines[1 : len(lines)-1] {
		if len(l) > 64 {
			t.Fatalf("line longer than 64: %d", len(l))
		}
	}
}
