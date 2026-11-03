package notify

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCaptchaDisabledPasses(t *testing.T) {
	if err := NewCaptcha().Verify(CaptchaConfig{Enable: false}, CaptchaRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestCaptchaTurnstileAndV3Threshold(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("response") == "good" {
			io.WriteString(w, `{"success":true,"score":0.9}`)
			return
		}
		io.WriteString(w, `{"success":true,"score":0.1}`)
	}))
	defer srv.Close()
	c := NewCaptcha()
	c.TurnstileURL = srv.URL
	c.RecaptchaURL = srv.URL

	if err := c.Verify(CaptchaConfig{Enable: true, Type: "turnstile"}, CaptchaRequest{TurnstileToken: "good"}); err != nil {
		t.Fatalf("turnstile good: %v", err)
	}
	if err := c.Verify(CaptchaConfig{Enable: true, Type: "turnstile"}, CaptchaRequest{}); err != ErrCaptcha {
		t.Fatalf("missing token: %v", err)
	}
	cfg := CaptchaConfig{Enable: true, Type: "recaptcha-v3", V3Threshold: 0.5}
	if err := c.Verify(cfg, CaptchaRequest{RecaptchaV3: "good"}); err != nil {
		t.Fatalf("v3 high score: %v", err)
	}
	if err := c.Verify(cfg, CaptchaRequest{RecaptchaV3: "bad"}); err != ErrCaptcha {
		t.Fatalf("v3 low score should fail: %v", err)
	}
	if err := c.Verify(CaptchaConfig{Enable: true, Type: "nope"}, CaptchaRequest{}); err == nil {
		t.Fatal("unknown type should error")
	}
}

func TestParseThreshold(t *testing.T) {
	if ParseThreshold("") != 0.5 || ParseThreshold("0.7") != 0.7 {
		t.Fatal("threshold parse")
	}
}
