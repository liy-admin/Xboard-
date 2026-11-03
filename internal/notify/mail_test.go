package notify

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestBuildMessageHeadersAndBody(t *testing.T) {
	at := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	raw := string(buildMessage("noreply@example.com", "XBoard", "u@example.com", "Hi", "<p>hello</p>", at))
	for _, want := range []string{
		"From: XBoard <noreply@example.com>\r\n",
		"To: u@example.com\r\n",
		"Content-Type: text/html; charset=UTF-8\r\n",
		"Content-Transfer-Encoding: base64\r\n",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing header %q in:\n%s", want, raw)
		}
	}
	body := raw[strings.Index(raw, "\r\n\r\n")+4:]
	dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(strings.TrimSpace(body), "\r\n", ""))
	if err != nil || string(dec) != "<p>hello</p>" {
		t.Fatalf("body round trip failed: %v %q", err, dec)
	}
}

func TestSMTPRequiresConfig(t *testing.T) {
	if err := (SMTP{}).Send("a@b.c", "s", "b"); err == nil {
		t.Fatal("unconfigured mailer must error")
	}
}
