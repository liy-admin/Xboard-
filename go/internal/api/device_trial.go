package api

import (
	"strings"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
)

// 设备体验账号：同一台设备只能领取一次试用，之后凭设备码重新登录同一个影子账号。
// 对应 V1/Passport/DeviceTrialController。

// deviceDisplayCode 对应 FNV-1a 哈希映射到 9 位数字（与客户端一致）。
func deviceDisplayCode(deviceID string) string {
	var hash uint32 = 0x811c9dc5
	const prime uint32 = 0x01000193
	for i := 0; i < len(deviceID); i++ {
		hash ^= uint32(deviceID[i])
		hash *= prime
	}
	return itoa64(100000000 + int64(hash%900000000))
}

func deviceEmail(deviceID string) string { return deviceDisplayCode(deviceID) + "@device.local" }

func isTrialEmail(email string) bool { return strings.HasSuffix(email, "@device.local") }

// bindingRow 读取 v2_device_binding 中的 user_id 与 claimed_trial（模型未映射 claimed_trial）。
type bindingRow struct {
	UserID       *int64
	ClaimedTrial bool
}

func (a *App) findBinding(deviceID string) (bindingRow, bool) {
	var b bindingRow
	var n int64
	a.S.DB.Model(&model.DeviceBinding{}).Where("device_id = ?", deviceID).Count(&n)
	if n == 0 {
		return b, false
	}
	a.S.DB.Raw("SELECT user_id, claimed_trial FROM v2_device_binding WHERE device_id = ?", deviceID).Row().Scan(&b.UserID, &b.ClaimedTrial)
	return b, true
}

func (a *App) setBindingUser(deviceID string, uid *int64) {
	a.S.DB.Exec("UPDATE v2_device_binding SET user_id = ?, updated_at = ? WHERE device_id = ?", uid, now(), deviceID)
}

func (a *App) deviceTrialClaim(c *gin.Context) {
	p := requestParams(c)
	deviceID, _ := p["device_id"].(string)
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		resp.Fail(c, 400, "Invalid device")
		return
	}
	deviceName, _ := p["device_name"].(string)
	b, found := a.findBinding(deviceID)
	if found && b.UserID != nil {
		var bound model.User
		if a.S.DB.First(&bound, *b.UserID).Error == nil {
			if isTrialEmail(bound.Email) {
				a.reauthTrialUser(c, &bound, deviceID, deviceName)
				return
			}
			var trial model.User
			if a.S.DB.Where("device_id = ? AND email LIKE ?", deviceID, "%@device.local").First(&trial).Error == nil {
				a.setBindingUser(deviceID, &trial.ID)
				a.reauthTrialUser(c, &trial, deviceID, deviceName)
				return
			}
		}
		a.setBindingUser(deviceID, nil)
	}
	if !found || !b.ClaimedTrial {
		tid := int64(a.Set.Int("try_out_plan_id", 0))
		var pl model.Plan
		if tid == 0 || a.S.DB.First(&pl, tid).Error != nil {
			resp.Fail(c, 403, "Trial is not available")
			return
		}
	}
	a.createTrialUser(c, deviceID, deviceName, found)
}

func (a *App) reauthTrialUser(c *gin.Context, u *model.User, deviceID, deviceName string) {
	if u.Banned {
		resp.Fail(c, 400, "Your account has been suspended")
		return
	}
	data, err := a.issueAuth(u, deviceName, deviceID)
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	data["is_device_trial"] = true
	resp.OK(c, data)
}

func (a *App) createTrialUser(c *gin.Context, deviceID, deviceName string, hasBinding bool) {
	var user model.User
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		pw, err := auth.HashPassword(auth.RandHex(16))
		if err != nil {
			return err
		}
		ts := now()
		user = model.User{Email: deviceEmail(deviceID), Password: pw, UUID: uuid.NewString(), Token: auth.RandHex(16),
			CreatedAt: ts, UpdatedAt: ts}
		did := deviceID
		user.DeviceID = &did
		remind := a.Set.Bool("default_remind_expire", true)
		remindT := a.Set.Bool("default_remind_traffic", true)
		user.RemindExpire, user.RemindTraffic = &remind, &remindT
		if dl := int64(a.Set.Int("default_device_limit", 4)); dl > 0 {
			user.DeviceLimit = &dl
		}
		granted := a.tryOutPlanFor(tx, &user, deviceID)
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if granted {
			logTrialGrant(tx, deviceID, &user, trialSourceDeviceTrial, c.ClientIP())
		}
		tx.Exec("UPDATE v2_user SET remarks = ? WHERE id = ?", "[设备体验] "+deviceDisplayCode(deviceID), user.ID)
		if hasBinding {
			tx.Exec("UPDATE v2_device_binding SET user_id = ?, claimed_trial = 1, updated_at = ? WHERE device_id = ?", user.ID, now(), deviceID)
		} else {
			tx.Exec("INSERT INTO v2_device_binding (device_id, user_id, claimed_trial, created_at, updated_at, type) VALUES (?, ?, 1, ?, ?, 'trial')",
				deviceID, user.ID, now(), now())
		}
		return nil
	})
	if err != nil {
		// 并发重复领取：回退为重新登录已绑定的账号（与 PHP 的 QueryException 分支一致）。
		if b, ok := a.findBinding(deviceID); ok && b.UserID != nil {
			var u model.User
			if a.S.DB.First(&u, *b.UserID).Error == nil {
				a.reauthTrialUser(c, &u, deviceID, deviceName)
				return
			}
		}
		resp.Fail(c, 500, "Failed to create trial account")
		return
	}
	a.reauthTrialUser(c, &user, deviceID, deviceName)
}
