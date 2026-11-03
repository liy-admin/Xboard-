package auth

import (
	"testing"
	"time"
)

func TestParseDBTimeAcceptsDriverAndPlainFormats(t *testing.T) {
	// 驱动开启 parseTime 后，把 DATETIME 扫描进 string 得到的是 RFC3339。
	if got, ok := ParseDBTime("2027-10-16T15:31:09+08:00"); !ok || got.UTC().Hour() != 7 {
		t.Fatalf("rfc3339: %v %v", got, ok)
	}
	if _, ok := ParseDBTime("2027-10-16 15:31:09"); !ok {
		t.Fatal("plain datetime must parse")
	}
	if _, ok := ParseDBTime("not a time"); ok {
		t.Fatal("garbage must not parse")
	}
}

func TestParseDBTimeExpiryComparison(t *testing.T) {
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	if e, ok := ParseDBTime(past); !ok || !e.Before(time.Now()) {
		t.Fatal("past RFC3339 time must be recognised as expired")
	}
	if e, ok := ParseDBTime(future); !ok || e.Before(time.Now()) {
		t.Fatal("future time must not be expired")
	}
}
