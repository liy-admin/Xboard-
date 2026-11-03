package api

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/notify"
	"xboard/internal/resp"
)

// 注册前的通讯接口：邮箱验证码发送、邀请码访问计数（对应 V1/Passport/CommController）。

// redisCodeStore 把 Redis 适配为 notify.CodeStore。
type redisCodeStore struct{ c *redis.Client }

func (s redisCodeStore) Get(ctx context.Context, key string) (string, bool, error) {
	v, err := s.c.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	return v, err == nil, err
}

func (s redisCodeStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.c.Set(ctx, key, value, ttl).Err()
}

func (s redisCodeStore) Incr(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	n, err := s.c.Incr(ctx, key).Result()
	if err == nil && n == 1 {
		_ = s.c.Expire(ctx, key, ttl).Err()
	}
	return n, err
}

func (s redisCodeStore) Del(ctx context.Context, keys ...string) error {
	return s.c.Del(ctx, keys...).Err()
}

// captchaConfig 从后台设置读取验证码配置。
func (a *App) captchaConfig() notify.CaptchaConfig {
	return notify.CaptchaConfig{
		Enable:         a.Set.Bool("captcha_enable", false),
		Type:           a.Set.Get("captcha_type", "recaptcha"),
		TurnstileKey:   a.Set.Get("turnstile_secret_key", ""),
		RecaptchaKey:   a.Set.Get("recaptcha_key", ""),
		RecaptchaV3Key: a.Set.Get("recaptcha_v3_secret_key", ""),
		V3Threshold:    notify.ParseThreshold(a.Set.Get("recaptcha_v3_score_threshold", "0.5")),
	}
}

// emailWhitelistAllowed 对应 email_whitelist_enable：已注册邮箱不受限制，新邮箱后缀需在白名单内。
func (a *App) emailWhitelistAllowed(email string) bool {
	if !a.Set.Bool("email_whitelist_enable", false) {
		return true
	}
	var n int64
	a.S.DB.Model(&model.User{}).Where("email = ?", notify.Normalize(email)).Count(&n)
	if n > 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	suffix := strings.ToLower(email[at+1:])
	for _, s := range strings.Split(a.Set.Get("email_whitelist_suffix", ""), ",") {
		if strings.TrimSpace(strings.ToLower(s)) == suffix && suffix != "" {
			return true
		}
	}
	return false
}

// sendEmailVerify 对应 CommController::sendEmailVerify。
func (a *App) sendEmailVerify(c *gin.Context) {
	var in struct {
		Email          string `json:"email" form:"email" binding:"required,email"`
		TurnstileToken string `json:"turnstile_token" form:"turnstile_token"`
		RecaptchaV3    string `json:"recaptcha_v3_token" form:"recaptcha_v3_token"`
		RecaptchaData  string `json:"recaptcha_data" form:"recaptcha_data"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	if err := notify.NewCaptcha().Verify(a.captchaConfig(), notify.CaptchaRequest{
		TurnstileToken: in.TurnstileToken, RecaptchaV3: in.RecaptchaV3, RecaptchaData: in.RecaptchaData, IP: c.ClientIP(),
	}); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	if !a.emailWhitelistAllowed(in.Email) {
		resp.Fail(c, 400, "Email suffix is not in whitelist")
		return
	}
	email := notify.Normalize(in.Email)
	flag := "LAST_SEND_EMAIL_VERIFY_TIMESTAMP_" + email
	if n, _ := a.S.Redis.Exists(bgCtx(), flag).Result(); n > 0 {
		resp.Fail(c, 400, "Email verification code has been sent, please request again later")
		return
	}
	ec := &notify.EmailCode{Store: redisCodeStore{a.S.Redis}}
	code, err := ec.Issue(bgCtx(), email)
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	app := a.Set.Get("app_name", "Lets VPN")
	subject := app + "邮箱验证码"
	values := map[string]any{"name": app, "code": code, "url": a.Set.Get("app_url", "")}
	a.mailQ().Enqueue(jobs.MailTask{To: email, Subject: subject, Template: "verify", Values: values})
	a.S.Redis.Set(bgCtx(), flag, now(), 60*time.Second)
	resp.OK(c, true)
}

// invitePV 对应 CommController::pv：邀请码访问次数 +1。
func (a *App) invitePV(c *gin.Context) {
	code, _ := requestParams(c)["invite_code"].(string)
	a.S.DB.Model(&model.InviteCode{}).Where("code = ?", code).UpdateColumn("pv", gorm.Expr("pv + 1"))
	resp.OK(c, true)
}
