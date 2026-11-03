package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestLuckyWinPercentClamps(t *testing.T) {
	if got := luckyWinPercent(500, 1000, 1, 50); got != 50 {
		t.Fatalf("want 50 (500/1000), got %v", got)
	}
	if got := luckyWinPercent(1, 1000000, 1, 50); got != 1 {
		t.Fatalf("want min clamp 1, got %v", got)
	}
	if got := luckyWinPercent(900, 1, 1, 50); got != 50 {
		t.Fatalf("want max clamp 50, got %v", got)
	}
	if got := luckyWinPercent(10, 0, 0, 100); got != 100 {
		t.Fatalf("zero users treated as 1, want 100 (10/1 clamped) got %v", got)
	}
}

func TestLuckyWonBoundary(t *testing.T) {
	// 10% == 100000 / 1000000，r 取 1..100000 中奖，100001 不中。
	if !luckyWon(10, 100000) || luckyWon(10, 100001) {
		t.Fatal("boundary at 10% wrong")
	}
	if luckyWon(0, 1) {
		t.Fatal("0% chance must never win")
	}
}

func TestExtendExpiry(t *testing.T) {
	now := int64(1000)
	past := int64(500)
	future := int64(5000)
	if got := extendExpiry(nil, now, 100); got != 1100 {
		t.Fatalf("nil expiry: %d", got)
	}
	if got := extendExpiry(&past, now, 100); got != 1100 {
		t.Fatalf("past expiry should restart from now: %d", got)
	}
	if got := extendExpiry(&future, now, 100); got != 5100 {
		t.Fatalf("future expiry should extend: %d", got)
	}
}

func TestRenderKnowledgeAccessAndPlaceholders(t *testing.T) {
	body := "A <!--access start-->secret<!--access end--> {{siteName}} {{subscribeUrl}}"
	out := renderKnowledge(body, false, "Site", "https://x/sub?token=t")
	if strings.Contains(out, "secret") || !strings.Contains(out, "v2board-no-access") {
		t.Fatalf("access block not hidden: %s", out)
	}
	if !strings.Contains(out, "Site https://x/sub?token=t") {
		t.Fatalf("placeholders not replaced: %s", out)
	}
	open := renderKnowledge(body, true, "Site", "u")
	if !strings.Contains(open, "secret") {
		t.Fatalf("available user should see content: %s", open)
	}
}

func TestSafeBase64Placeholder(t *testing.T) {
	out := renderKnowledge("{{safeBase64SubscribeUrl}}", true, "", "https://a/??>")
	want := strings.NewReplacer("+", "-", "/", "_", "=", "").Replace(base64.StdEncoding.EncodeToString([]byte("https://a/??>")))
	if out != want {
		t.Fatalf("want %s got %s", want, out)
	}
}

func TestShareLink(t *testing.T) {
	cases := []struct{ tpl, app, want string }{
		{"https://d.example/?c=:code", "", "https://d.example/?c=ABC"},
		{"https://d.example/landing", "", "https://d.example/landing"},
		{"ftp://bad", "https://app.example/", "https://app.example/#/register?referral_code=ABC"},
		{"", "", ""},
		{"", "https://app.example/", "https://app.example/#/register?referral_code=ABC"},
	}
	for _, c := range cases {
		if got := shareLink(c.tpl, c.app, "ABC"); got != c.want {
			t.Errorf("tpl=%q app=%q: want %q got %q", c.tpl, c.app, c.want, got)
		}
	}
}

func TestNewReferralCodeFormat(t *testing.T) {
	for i := 0; i < 200; i++ {
		c := newReferralCode()
		if len(c) != 6 {
			t.Fatalf("length: %q", c)
		}
		for _, r := range c {
			if r < '0' || r > '9' {
				t.Fatalf("non digit: %q", c)
			}
		}
	}
}

func TestRandomCharsFormat(t *testing.T) {
	s := randomChars(8)
	if len(s) != 8 {
		t.Fatalf("len %d", len(s))
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			t.Fatalf("bad char %q", r)
		}
	}
}

// 解密实现与 PHP 端 openssl_decrypt 等价，用于验证信封格式。
func openSupport(t *testing.T, encKey string, env supportEnvelope) string {
	t.Helper()
	nonce, _ := base64.StdEncoding.DecodeString(env.N)
	ct, _ := base64.StdEncoding.DecodeString(env.C)
	tag, _ := base64.StdEncoding.DecodeString(env.T)
	key := sha256.Sum256([]byte(encKey))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 12)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := gcm.Open(nil, nonce, append(ct, tag...), nil)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	return string(pt)
}

func TestSupportEnvelopeRoundTrip(t *testing.T) {
	const k = "c2VjcmV0LWtleS1mb3ItdGVzdGluZy1vbmx5MTIzNDU2"
	env, err := sealSupport(k, supportPayload{Enabled: true, BaseURL: "https://cw.example", WebsiteToken: "tok", Locale: "zh_CN"})
	if err != nil {
		t.Fatal(err)
	}
	if env.V != 1 || env.Alg != "aes-256-gcm" {
		t.Fatalf("bad envelope header: %+v", env)
	}
	raw := openSupport(t, k, env)
	var got supportPayload
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.WebsiteToken != "tok" || !got.Enabled || got.Locale != "zh_CN" {
		t.Fatalf("payload mismatch: %s", raw)
	}
	if !strings.HasPrefix(raw, `{"enabled":true,"base_url"`) {
		t.Fatalf("key order differs from PHP: %s", raw)
	}
}
