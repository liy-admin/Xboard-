package notify

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrCaptcha 为统一的校验失败文案（对应 __('Invalid code is incorrect')）。
var ErrCaptcha = errors.New("Invalid code is incorrect")

// CaptchaConfig 来自后台设置。
type CaptchaConfig struct {
	Enable         bool
	Type           string // turnstile | recaptcha-v3 | recaptcha
	TurnstileKey   string
	RecaptchaKey   string
	RecaptchaV3Key string
	V3Threshold    float64
}

// CaptchaRequest 为请求中携带的令牌与客户端 IP。
type CaptchaRequest struct {
	TurnstileToken string
	RecaptchaV3    string
	RecaptchaData  string
	IP             string
}

// Captcha 校验验证码。地址字段可替换，便于测试。
type Captcha struct {
	HTTP         *http.Client
	TurnstileURL string
	RecaptchaURL string
}

// NewCaptcha 返回指向各服务商正式地址的校验器。
func NewCaptcha() *Captcha {
	return &Captcha{
		HTTP:         &http.Client{Timeout: 15 * time.Second},
		TurnstileURL: "https://challenges.cloudflare.com/turnstile/v0/siteverify",
		RecaptchaURL: "https://www.google.com/recaptcha/api/siteverify",
	}
}

// Verify 对应 CaptchaService::verify。验证码关闭时直接通过。
func (c *Captcha) Verify(cfg CaptchaConfig, req CaptchaRequest) error {
	if !cfg.Enable {
		return nil
	}
	switch cfg.Type {
	case "turnstile":
		if req.TurnstileToken == "" {
			return ErrCaptcha
		}
		ok, _, err := c.siteverify(c.TurnstileURL, url.Values{
			"secret": {cfg.TurnstileKey}, "response": {req.TurnstileToken}, "remoteip": {req.IP},
		})
		if err != nil || !ok {
			return ErrCaptcha
		}
		return nil
	case "recaptcha-v3":
		if req.RecaptchaV3 == "" {
			return ErrCaptcha
		}
		ok, score, err := c.siteverify(c.RecaptchaURL, url.Values{
			"secret": {cfg.RecaptchaV3Key}, "response": {req.RecaptchaV3}, "remoteip": {req.IP},
		})
		if err != nil || !ok || score < cfg.V3Threshold {
			return ErrCaptcha
		}
		return nil
	case "recaptcha":
		if req.RecaptchaData == "" {
			return ErrCaptcha
		}
		ok, _, err := c.siteverify(c.RecaptchaURL, url.Values{
			"secret": {cfg.RecaptchaKey}, "response": {req.RecaptchaData}, "remoteip": {req.IP},
		})
		if err != nil || !ok {
			return ErrCaptcha
		}
		return nil
	}
	return errors.New("Invalid captcha type")
}

// siteverify 以 POST 表单调用服务商接口，解析 success 与 score。
func (c *Captcha) siteverify(endpoint string, form url.Values) (bool, float64, error) {
	resp, err := c.HTTP.PostForm(endpoint, form)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r struct {
		Success bool    `json:"success"`
		Score   float64 `json:"score"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return false, 0, err
	}
	return r.Success, r.Score, nil
}

// ParseThreshold 解析 recaptcha_v3_score_threshold，缺省 0.5。
func ParseThreshold(s string) float64 {
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return f
	}
	return 0.5
}
