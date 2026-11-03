package api

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
)

func (a *App) login(c *gin.Context) {
	var in struct {
		Email      string `json:"email" form:"email" binding:"required"`
		Password   string `json:"password" form:"password" binding:"required"`
		DeviceID   string `json:"device_id" form:"device_id"`
		DeviceName string `json:"device_name" form:"device_name"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	email := lower(in.Email)

	// 密码错误次数限制：沿用 PASSWORD_ERROR_LIMIT 的语义（Redis 计数）。
	limitKey := "PASSWORD_ERROR_LIMIT_" + email
	if a.Set.Int("password_limit_enable", 1) != 0 {
		n, _ := a.S.Redis.Get(c, limitKey).Int()
		if n >= a.Set.Int("password_limit_count", 5) {
			resp.Fail(c, 429, "There are too many password errors, please try again later.")
			return
		}
	}

	var u model.User
	if err := a.S.DB.Where("email = ?", email).First(&u).Error; err != nil || !auth.VerifyPassword(&u, in.Password) {
		if a.Set.Int("password_limit_enable", 1) != 0 {
			a.S.Redis.Incr(c, limitKey)
			a.S.Redis.Expire(c, limitKey, time.Duration(a.Set.Int("password_limit_expire", 60))*time.Minute)
		}
		resp.Fail(c, 400, "Incorrect email or password")
		return
	}
	if u.Banned {
		resp.Fail(c, 400, "Your account has been suspended")
		return
	}
	// 网页登录开关：无 device_id 视为网页端，非管理员拦截。
	if a.Set.Int("user_web_login_enable", 1) == 0 && !u.IsAdmin && in.DeviceID == "" {
		resp.Fail(c, 403, "Web login is not allowed for this account, please sign in from the app")
		return
	}

	name := fmt.Sprintf(`{"device_name":%q,"device_id":%q}`, in.DeviceName, in.DeviceID)
	if in.DeviceName == "" && in.DeviceID == "" {
		name = auth.RandHex(10)
	}
	if in.DeviceID != "" {
		a.S.DB.Exec(`DELETE FROM personal_access_tokens WHERE tokenable_id = ? AND name LIKE ?`,
			u.ID, `%"device_id":"`+in.DeviceID+`"%`)
	}
	bearer, err := auth.Issue(a.S.DB, u.ID, name)
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	a.S.DB.Model(&u).Update("last_login_at", now())
	resp.OK(c, gin.H{
		"token": u.Token, "auth_data": bearer, "is_admin": u.IsAdmin, "enc_key": a.encKey(&u),
	})
}

func (a *App) encKey(u *model.User) string {
	if u.EncKey != nil && *u.EncKey != "" {
		return *u.EncKey
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	k := base64.StdEncoding.EncodeToString(b)
	if a.S.DB.Model(u).Update("enc_key", k).Error != nil {
		return ""
	}
	return k
}

func (a *App) checkLogin(c *gin.Context) {
	u := cur(c)
	resp.OK(c, gin.H{"is_login": true, "is_admin": u.IsAdmin})
}

func (a *App) userInfo(c *gin.Context) {
	u := cur(c)
	// 字段与 PHP UserController::info 的 select 一致；remind_* 为布尔（模型 casts）。
	boolOf := func(p *bool) bool { return p != nil && *p }
	resp.OK(c, gin.H{
		"email": u.Email, "transfer_enable": u.TransferEnab, "device_limit": u.DeviceLimit,
		"last_login_at": u.LastLoginAt, "created_at": u.CreatedAt, "banned": u.Banned,
		"remind_expire": boolOf(u.RemindExpire), "remind_traffic": boolOf(u.RemindTraffic),
		"expired_at": u.ExpiredAt, "balance": u.Balance, "commission_balance": u.CommissionBal,
		"plan_id": u.PlanID, "discount": u.Discount, "commission_rate": u.CommissionRate,
		"telegram_id": u.TelegramID, "uuid": u.UUID, "avatar_url": gravatar(u.Email),
	})
}

func gravatar(email string) string {
	return "https://cdn.v2ex.com/gravatar/" + md5hex(lower(email)) + "?s=64&d=identicon"
}

func (a *App) changePassword(c *gin.Context) {
	var in struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	u := cur(c)
	if !auth.VerifyPassword(u, in.OldPassword) {
		resp.Fail(c, 400, "The old password is wrong")
		return
	}
	h, err := auth.HashPassword(in.NewPassword)
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	a.S.DB.Model(u).Updates(map[string]any{"password": h, "password_algo": nil, "password_salt": nil})
	resp.OK(c, true)
}
