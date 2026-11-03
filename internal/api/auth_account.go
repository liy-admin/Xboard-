package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/notify"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 注册、找回密码、快捷登录，以及用户账户相关接口（转移佣金、会话、注销、提醒设置）。
// 对应 V1/Passport/AuthController、V1/User/UserController、Services/Auth/RegisterService、LoginService。

// tokenTTL 对应 generateQuickLoginUrl / TEMP_TOKEN 的 60 秒有效期。
const tempTokenTTL = 60 * time.Second

// issueAuth 对应 AuthService::generateAuthData。
func (a *App) issueAuth(u *model.User, deviceName, deviceID string) (gin.H, error) {
	name := auth.RandHex(10)
	if deviceName != "" || deviceID != "" {
		name = fmt.Sprintf(`{"device_name":%q,"device_id":%q}`, deviceName, deviceID)
	}
	bearer, err := auth.Issue(a.S.DB, u.ID, name)
	if err != nil {
		return nil, err
	}
	return gin.H{"token": u.Token, "auth_data": bearer, "is_admin": u.IsAdmin, "enc_key": a.encKey(u)}, nil
}

// webLoginBlocked 对应 isWebLoginBlocked：关闭网页登录时，非管理员不发放网页会话。
func (a *App) webLoginBlocked(u *model.User) bool {
	return a.Set.Int("user_web_login_enable", 1) == 0 && !u.IsAdmin
}

// registerWebBlocked 对应 PHP isWebLoginBlocked 在注册上的语义：新注册账号必不是管理员，
// 带 device_id 视为 App 端恒放行，只有关闭网页登录时的网页端（无 device_id）注册才拦截。
func (a *App) registerWebBlocked(deviceID string) bool {
	return a.Set.Int("user_web_login_enable", 1) == 0 && strings.TrimSpace(deviceID) == ""
}

// emailSuffixAllowed 对应 Helper::emailSuffixVerify，白名单为逗号分隔的后缀列表。
func (a *App) emailSuffixAllowed(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	suffix := strings.ToLower(email[at+1:])
	for _, s := range strings.Split(a.Set.Get("email_whitelist_suffix", ""), ",") {
		if strings.ToLower(strings.TrimSpace(s)) == suffix && suffix != "" {
			return true
		}
	}
	return false
}

// tryOutPlanFor 对应 UserService::setTryOutPlan：设备未领取过试用时，按 try_out_plan_id 发放试用套餐。
// 返回是否真的发放了试用；调用方负责在同一事务里把该设备标记为已领取（见 markTrialClaimed）。
// 读 claimed_trial 时加行锁，同一设备并发注册时后到的请求会等前一个事务提交后读到已领取。
func (a *App) tryOutPlanFor(tx *gorm.DB, u *model.User, deviceID string) bool {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false
	}
	var claimed bool
	tx.Raw("SELECT claimed_trial FROM v2_device_binding WHERE device_id = ? LIMIT 1 FOR UPDATE", deviceID).Scan(&claimed)
	if claimed {
		return false
	}
	planID := int64(a.Set.Int("try_out_plan_id", 0))
	if planID == 0 {
		return false
	}
	var p model.Plan
	if tx.First(&p, planID).Error != nil {
		return false
	}
	u.TransferEnab = shop.GbToTransfer(derefInt64(p.TransferEnable))
	pid := p.ID
	u.PlanID = &pid
	u.GroupID = p.GroupID
	exp := now() + int64(a.Set.Int("try_out_hour", 1))*3600
	u.ExpiredAt = &exp
	u.SpeedLimit = p.SpeedLimit
	u.DeviceLimit = p.DeviceLimit
	return true
}

// markTrialClaimed 把设备标记为已领取试用（claimed_trial 一旦为 1 永不重置），并绑定到 uid。
// device_id 有唯一索引，用 upsert 同时覆盖"已有绑定记录"和"首次出现"两种情况。
func markTrialClaimed(tx *gorm.DB, deviceID string, uid int64) *gorm.DB {
	ts := now()
	return tx.Exec(`INSERT INTO v2_device_binding (device_id, user_id, claimed_trial, created_at, updated_at, type)
VALUES (?, ?, 1, ?, ?, 'trial')
ON DUPLICATE KEY UPDATE claimed_trial = 1, user_id = ?, updated_at = ?`,
		strings.TrimSpace(deviceID), uid, ts, ts, uid, ts)
}

// 试用领取记录的来源（v2_device_trial_log.source）。
const (
	trialSourceRegister    = "register"
	trialSourceDeviceTrial = "device_trial"
)

// logTrialGrant 记录一次试用发放，供后台「设备管理」查看领取历史。
// 邮箱存快照：账号注销后 v2_user.email 会被匿名化。
// 只是辅助记录，写入失败（例如关闭了自动迁移、表还没建）不影响注册/领取本身：
// MySQL 单条语句失败只回滚该语句，事务其余部分照常提交。
func logTrialGrant(tx *gorm.DB, deviceID string, u *model.User, source, ip string) {
	rec := model.DeviceTrialLog{
		DeviceID: strings.TrimSpace(deviceID), UserID: u.ID, Email: u.Email, Source: source,
		PlanID: u.PlanID, ExpiredAt: u.ExpiredAt, CreatedAt: now(),
	}
	if ip != "" {
		rec.IP = &ip
	}
	if err := tx.Create(&rec).Error; err != nil {
		log.Printf("logTrialGrant: device=%s user=%d: %v", rec.DeviceID, u.ID, err)
	}
}

// register 对应 AuthController::register（含 RegisterService 的校验与建号逻辑）。
func (a *App) register(c *gin.Context) {
	var in struct {
		Email         string `json:"email" form:"email" binding:"required,email"`
		Password      string `json:"password" form:"password" binding:"required,min=8"`
		InviteCode    string `json:"invite_code" form:"invite_code"`
		ReferralCode  string `json:"referral_code" form:"referral_code"`
		EmailCode     string `json:"email_code" form:"email_code"`
		DeviceID      string `json:"device_id" form:"device_id"`
		DeviceName    string `json:"device_name" form:"device_name"`
		TurnstileTok  string `json:"turnstile_token" form:"turnstile_token"`
		RecaptchaV3   string `json:"recaptcha_v3_token" form:"recaptcha_v3_token"`
		RecaptchaData string `json:"recaptcha_data" form:"recaptcha_data"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, registerBindMessage(err))
		return
	}
	email := notify.Normalize(in.Email)
	ip := c.ClientIP()
	ipKey := "REGISTER_IP_RATE_LIMIT_" + ip

	if a.Set.Bool("register_limit_by_ip_enable", false) {
		n, _ := a.S.Redis.Get(bgCtx(), ipKey).Int64()
		if n >= int64(a.Set.Int("register_limit_count", 3)) {
			resp.Fail(c, 429, fmt.Sprintf("注册过于频繁，请 %d 分钟后再试", a.Set.Int("register_limit_expire", 60)))
			return
		}
	}
	if err := notify.NewCaptcha().Verify(a.captchaConfig(), notify.CaptchaRequest{
		TurnstileToken: in.TurnstileTok, RecaptchaV3: in.RecaptchaV3, RecaptchaData: in.RecaptchaData, IP: ip,
	}); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	if a.Set.Bool("email_whitelist_enable", false) && !a.emailSuffixAllowed(email) {
		resp.Fail(c, 400, "邮箱后缀不在白名单中")
		return
	}
	if a.Set.Bool("email_gmail_limit_enable", false) {
		prefix := strings.SplitN(email, "@", 2)[0]
		if strings.ContainsAny(prefix, ".+") {
			resp.Fail(c, 400, "不支持 Gmail 别名邮箱")
			return
		}
	}
	if a.Set.Bool("stop_register", false) {
		resp.Fail(c, 400, "注册已关闭")
		return
	}
	if a.Set.Bool("invite_force", false) && in.InviteCode == "" {
		resp.Fail(c, 422, "必须使用邀请码注册")
		return
	}
	if a.Set.Bool("email_verify", false) {
		if len(in.EmailCode) != 6 {
			resp.Fail(c, 422, "邮箱验证码不能为空")
			return
		}
		ok, _ := (&notify.EmailCode{Store: redisCodeStore{a.S.Redis}}).Verify(bgCtx(), email, in.EmailCode)
		if !ok {
			resp.Fail(c, 400, "邮箱验证码错误")
			return
		}
	}
	var exist int64
	a.S.DB.Model(&model.User{}).Where("email = ?", email).Count(&exist)
	if exist > 0 {
		resp.Fail(c, 400, "邮箱已存在")
		return
	}
	// 必须在建号之前拦截：否则账号已写库却返回失败，用户重试只会得到"邮箱已存在"。
	if a.registerWebBlocked(in.DeviceID) {
		resp.Fail(c, 403, "Web login is not allowed for this account, please sign in from the app")
		return
	}

	var inviteUserID *int64
	if in.InviteCode != "" {
		var ic model.InviteCode
		if err := a.S.DB.Where("code = ? AND status = 0", in.InviteCode).First(&ic).Error; err == nil {
			uid := ic.UserID
			inviteUserID = &uid
			if !a.Set.Bool("invite_never_expire", false) {
				a.S.DB.Model(&ic).Update("status", 1)
			}
		} else if a.Set.Bool("invite_force", false) {
			resp.Fail(c, 400, "邀请码无效")
			return
		}
	} else if in.ReferralCode != "" {
		var r model.User
		if a.S.DB.Where("referral_code = ?", in.ReferralCode).First(&r).Error == nil {
			uid := r.ID
			inviteUserID = &uid
		}
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		resp.Fail(c, 500, "Register failed")
		return
	}
	ts := now()
	u := model.User{
		Email: email, Password: hash, UUID: uuid.NewString(), Token: auth.RandHex(16),
		InviteUserID: inviteUserID, CreatedAt: ts, UpdatedAt: ts, LastLoginAt: &ts,
		Banned: false, CommissionType: 0,
	}
	if dl := int64(a.Set.Int("default_device_limit", 4)); dl > 0 {
		u.DeviceLimit = &dl
	}
	remindExpire := a.Set.Bool("default_remind_expire", true)
	remindTraffic := a.Set.Bool("default_remind_traffic", true)
	u.RemindExpire, u.RemindTraffic = &remindExpire, &remindTraffic
	if in.DeviceID != "" {
		did := in.DeviceID
		u.DeviceID = &did
	}
	err = a.S.DB.Transaction(func(tx *gorm.DB) error {
		granted := a.tryOutPlanFor(tx, &u, in.DeviceID)
		if err := tx.Create(&u).Error; err != nil {
			return err
		}
		// 注册发了试用就记到设备上，否则注销后重注册、或同设备换邮箱注册都能反复领。
		if !granted {
			return nil
		}
		if err := markTrialClaimed(tx, in.DeviceID, u.ID).Error; err != nil {
			return err
		}
		logTrialGrant(tx, in.DeviceID, &u, trialSourceRegister, ip)
		return nil
	})
	if err != nil {
		resp.Fail(c, 500, "Register failed")
		return
	}
	if a.Set.Bool("email_verify", false) {
		_ = (&notify.EmailCode{Store: redisCodeStore{a.S.Redis}}).Forget(bgCtx(), email)
	}
	if a.Set.Bool("register_limit_by_ip_enable", false) {
		a.S.Redis.Incr(bgCtx(), ipKey)
		a.S.Redis.Expire(bgCtx(), ipKey, time.Duration(a.Set.Int("register_limit_expire", 60))*time.Minute)
	}
	a.bindDevice(in.DeviceID, &u)
	data, err := a.issueAuth(&u, in.DeviceName, in.DeviceID)
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	resp.OK(c, data)
}

// forget 对应 AuthController::forget / LoginService::resetPassword。
func (a *App) forget(c *gin.Context) {
	var in struct {
		Email     string `json:"email" form:"email" binding:"required,email"`
		Password  string `json:"password" form:"password" binding:"required,min=8"`
		EmailCode string `json:"email_code" form:"email_code" binding:"required,len=6"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	email := notify.Normalize(in.Email)
	limitKey := "FORGET_REQUEST_LIMIT_" + email
	n, _ := a.S.Redis.Get(bgCtx(), limitKey).Int64()
	if n >= 3 {
		resp.Fail(c, 429, "重置失败，请稍后再试")
		return
	}
	ec := &notify.EmailCode{Store: redisCodeStore{a.S.Redis}}
	ok, _ := ec.Verify(bgCtx(), email, in.EmailCode)
	if !ok {
		a.S.Redis.Incr(bgCtx(), limitKey)
		a.S.Redis.Expire(bgCtx(), limitKey, 300*time.Second)
		resp.Fail(c, 400, "邮箱验证码错误")
		return
	}
	var u model.User
	if err := a.S.DB.Where("email = ?", email).First(&u).Error; err != nil {
		resp.Fail(c, 400, "该邮箱未在系统中注册")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		resp.Fail(c, 500, "Reset failed")
		return
	}
	if err := a.S.DB.Model(&u).Updates(map[string]any{"password": hash, "password_algo": nil, "password_salt": nil}).Error; err != nil {
		resp.Fail(c, 500, "Reset failed")
		return
	}
	_ = ec.Forget(bgCtx(), email)
	resp.OK(c, true)
}

// newTempToken 生成一次性快捷登录码，写入 TEMP_TOKEN_<code>，60 秒有效。
func (a *App) newTempToken(uid int64) string {
	code := auth.RandHex(16)
	a.S.Redis.Set(bgCtx(), "TEMP_TOKEN_"+code, uid, tempTokenTTL)
	return code
}

// quickLoginURL 对应 LoginService::generateQuickLoginUrl。
func (a *App) quickLoginURL(uid int64, redirect string) string {
	code := a.newTempToken(uid)
	if redirect == "" {
		redirect = "dashboard"
	}
	path := "/#/login?verify=" + code + "&redirect=" + url.QueryEscape(redirect)
	return strings.TrimRight(a.Set.Get("app_url", ""), "/") + path
}

// token2Login 对应 AuthController::token2Login：token 参数为跳转，verify 参数为一次性登录。
func (a *App) token2Login(c *gin.Context) {
	p := requestParams(c)
	if tok, _ := p["token"].(string); tok != "" {
		redirect, _ := p["redirect"].(string)
		if redirect == "" {
			redirect = "dashboard"
		}
		target := "/#/login?verify=" + url.QueryEscape(tok) + "&redirect=" + url.QueryEscape(redirect)
		c.Redirect(302, strings.TrimRight(a.Set.Get("app_url", ""), "/")+target)
		return
	}
	verify, _ := p["verify"].(string)
	if verify == "" {
		c.JSON(400, gin.H{"message": "Invalid request"})
		return
	}
	key := "TEMP_TOKEN_" + verify
	uid, err := a.S.Redis.Get(bgCtx(), key).Int64()
	if err != nil || uid == 0 {
		c.JSON(400, gin.H{"message": "Token error"})
		return
	}
	a.S.Redis.Del(bgCtx(), key)
	var u model.User
	if err := a.S.DB.First(&u, uid).Error; err != nil {
		c.JSON(400, gin.H{"message": "User not found"})
		return
	}
	if u.Banned {
		c.JSON(400, gin.H{"message": "Token error"})
		return
	}
	if a.webLoginBlocked(&u) {
		c.JSON(403, gin.H{"message": "Web login is not allowed for this account, please sign in from the app"})
		return
	}
	data, err := a.issueAuth(&u, "", "")
	if err != nil {
		c.JSON(500, gin.H{"message": "Internal error"})
		return
	}
	c.JSON(200, gin.H{"data": data})
}

// passportQuickLoginURL 对应 AuthController::getQuickLoginUrl：凭 auth_data 或 Authorization 头取用户。
func (a *App) passportQuickLoginURL(c *gin.Context) {
	p := requestParams(c)
	authz, _ := p["auth_data"].(string)
	if authz == "" {
		authz = c.GetHeader("Authorization")
	}
	if authz == "" {
		c.JSON(401, gin.H{"message": "Unauthorized"})
		return
	}
	u := auth.UserFromBearer(a.S.DB, authz)
	if u == nil {
		c.JSON(401, gin.H{"message": "Unauthenticated, expired"})
		return
	}
	if a.webLoginBlocked(u) {
		resp.Fail(c, 403, "Web login is not allowed for this account, please sign in from the app")
		return
	}
	redirect, _ := p["redirect"].(string)
	resp.OK(c, a.quickLoginURL(u.ID, redirect))
}

// ---- 用户账户接口（V1/User/UserController） ----

// userTransfer 对应 transfer：佣金余额转入账户余额。
func (a *App) userTransfer(c *gin.Context) {
	var in struct {
		Amount int64 `json:"transfer_amount" form:"transfer_amount" binding:"required"`
	}
	if err := c.ShouldBind(&in); err != nil || in.Amount <= 0 {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	uid := cur(c).ID
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		var u model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&u, uid).Error; err != nil {
			return fmt.Errorf("The user does not exist")
		}
		if in.Amount > u.CommissionBal {
			return fmt.Errorf("Insufficient commission balance")
		}
		return tx.Model(&model.User{}).Where("id = ?", uid).Updates(map[string]any{
			"commission_balance": gorm.Expr("commission_balance - ?", in.Amount),
			"balance":            gorm.Expr("balance + ?", in.Amount),
		}).Error
	})
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	resp.OK(c, true)
}

// sessionView 对应 personal_access_tokens（Sanctum 令牌模型的 JSON）：token 哈希不返回给客户端，
// abilities 为数组，时间为 UTC 的 ISO 8601（与 Laravel 序列化一致）。
type sessionView struct {
	ID            int64    `json:"id"`
	TokenableType string   `json:"tokenable_type"`
	TokenableID   int64    `json:"tokenable_id"`
	Name          string   `json:"name"`
	Abilities     []string `json:"abilities"`
	LastUsedAt    *string  `json:"last_used_at"`
	ExpiresAt     *string  `json:"expires_at"`
	CreatedAt     *string  `json:"created_at"`
	UpdatedAt     *string  `json:"updated_at"`
}

// isoUTC 把数据库里的本地时间字符串转成 Laravel 的 ISO 8601（UTC，带微秒），无法解析时原样返回。
func isoUTC(v *string) *string {
	if v == nil || *v == "" {
		return v
	}
	t, ok := auth.ParseDBTime(*v)
	if !ok {
		return v
	}
	s := t.UTC().Format("2006-01-02T15:04:05.000000Z")
	return &s
}

func (a *App) userSessions(c *gin.Context) {
	var rows []model.PersonalAccessToken
	a.S.DB.Where("tokenable_type = ? AND tokenable_id = ?", "App\\Models\\User", cur(c).ID).Find(&rows)
	out := make([]sessionView, 0, len(rows))
	for _, r := range rows {
		abilities := []string{}
		if r.Abilities != nil {
			_ = json.Unmarshal([]byte(*r.Abilities), &abilities)
		}
		out = append(out, sessionView{ID: r.ID, TokenableType: r.TokenableType, TokenableID: r.TokenableID,
			Name: r.Name, Abilities: abilities, LastUsedAt: isoUTC(r.LastUsedAt), ExpiresAt: isoUTC(r.ExpiresAt),
			CreatedAt: isoUTC(r.CreatedAt), UpdatedAt: isoUTC(r.UpdatedAt)})
	}
	resp.OK(c, out)
}

func (a *App) userRemoveSession(c *gin.Context) {
	sid := requestParams(c)["session_id"]
	id, _ := anyInt(sid)
	a.S.DB.Where("id = ? AND tokenable_type = ? AND tokenable_id = ?", id, "App\\Models\\User", cur(c).ID).
		Delete(&model.PersonalAccessToken{})
	resp.OK(c, true)
}

// userCancelAccount 对应 cancelAccount：匿名化账号并封禁，撤销全部会话。
func (a *App) userCancelAccount(c *gin.Context) {
	u := cur(c)
	if u.IsAdmin {
		resp.Fail(c, 400, "管理员账号无法自行注销，请联系其他管理员")
		return
	}
	pw, _ := requestParams(c)["password"].(string)
	if !auth.VerifyPassword(u, pw) {
		resp.Fail(c, 400, "密码错误")
		return
	}
	ts := now()
	upd := map[string]any{
		"email":         fmt.Sprintf("deleted_%d_%d@deleted.local", u.ID, ts),
		"password":      mustHash(newEncKey()),
		"password_algo": nil, "password_salt": nil, "banned": true,
		"telegram_id": nil, "uuid": uuid.NewString(), "token": auth.RandHex(16), "updated_at": ts,
	}
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ?", u.ID).Updates(upd).Error; err != nil {
			return err
		}
		return tx.Where("tokenable_type = ? AND tokenable_id = ?", "App\\Models\\User", u.ID).
			Delete(&model.PersonalAccessToken{}).Error
	})
	if err != nil {
		resp.Fail(c, 500, "注销失败")
		return
	}
	resp.OK(c, true)
}

// mustHash 为注销后的随机密码生成哈希；失败时返回空串（账号已封禁，不影响安全）。
func mustHash(s string) string {
	h, err := auth.HashPassword(s)
	if err != nil {
		return ""
	}
	return h
}

// userQuickLoginURL 对应 UserController::getQuickLoginUrl。
func (a *App) userQuickLoginURL(c *gin.Context) {
	u := cur(c)
	if a.webLoginBlocked(u) {
		resp.Fail(c, 403, "Web login is not allowed for this account, please sign in from the app")
		return
	}
	redirect, _ := requestParams(c)["redirect"].(string)
	resp.OK(c, a.quickLoginURL(u.ID, redirect))
}

// userUpdate 对应 update：只允许修改到期与流量提醒开关（0/1）。
func (a *App) userUpdate(c *gin.Context) {
	p := requestParams(c)
	upd := map[string]any{}
	for _, k := range []string{"remind_expire", "remind_traffic"} {
		if v, ok := p[k]; ok && v != nil {
			n, ok2 := anyInt(v)
			if !ok2 || (n != 0 && n != 1) {
				resp.Fail(c, 422, "Invalid parameters")
				return
			}
			upd[k] = n
		}
	}
	if len(upd) > 0 {
		if err := a.S.DB.Model(&model.User{}).Where("id = ?", cur(c).ID).Updates(upd).Error; err != nil {
			resp.Fail(c, 400, "Save failed")
			return
		}
	}
	resp.OK(c, true)
}

// registerBindMessage 把参数校验失败翻译为具体原因，对应 AuthRegister::messages。
// 笼统的 "Invalid parameters" 会让客户端无法判断是邮箱格式、密码长度还是请求体格式问题。
func registerBindMessage(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		f := ve[0]
		switch f.Field() {
		case "Email":
			if f.Tag() == "required" {
				return "邮箱不能为空"
			}
			return "邮箱格式不正确"
		case "Password":
			if f.Tag() == "required" {
				return "密码不能为空"
			}
			return "密码长度不能少于 8 位"
		}
		return "参数 " + f.Field() + " 不合法"
	}
	return "请求格式错误，请检查参数（email、password）"
}
