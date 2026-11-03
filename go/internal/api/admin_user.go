package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 管理端用户：完整列表过滤、更新、删除、生成、在线用户、用户详情。
// 对应 V2/Admin/UserController。未包含：sendMail、dumpCSV、setInviteUser（待后续补充）。

// adminUserFilter 为列表允许的过滤字段（与 PHP ALLOWED_FILTER_COLUMNS 一致，total_used 特殊处理）。
//
// PHP 的白名单没有 online_count / uuid / token / remarks / invite_user_id / is_staff，但管理端筛选界面提供了这些入口，
// 在 PHP 里它们从来不生效。这里补上，让界面上能选的筛选项都真正起作用。
var adminUserFilter = map[string]bool{"id": true, "email": true, "plan_id": true, "group_id": true,
	"transfer_enable": true, "expired_at": true, "created_at": true, "balance": true, "commission_balance": true,
	"is_admin": true, "banned": true, "invite_by_id": true, "last_login_at": true, "total_used": true,
	"online_count": true, "uuid": true, "token": true, "remarks": true, "invite_user_id": true, "is_staff": true,
	"device_limit": true}

var adminUserSort = map[string]bool{"id": true, "email": true, "created_at": true, "expired_at": true, "balance": true,
	"commission_balance": true, "plan_id": true, "transfer_enable": true, "u": true, "d": true, "last_login_at": true,
	"is_admin": true, "banned": true}

// adminUserList 返回分页列表（支持 filter/sort 参数）。
func (a *App) adminUserList(c *gin.Context) {
	if err := checkFilters(c, adminUserFilter); err != nil {
		resp.Fail(c, 422, err.Error())
		return
	}
	cur, size := pageOf(c)
	q := a.S.DB.Model(&model.User{})
	q = adminFilters(c, adminUserFilter, q)
	var total int64
	q.Count(&total)
	var rows []model.User
	adminSorts(c, adminUserSort, q).Order("id DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	paginateJSON(c, total, cur, size, a.userAdminViews(rows))
}

// userAdminView 对应 transformUserData：金额换算为元，附加订阅地址与关联信息。
func (a *App) userAdminView(u *model.User) gin.H {
	return a.userAdminViews([]model.User{*u})[0]
}

// userExtra 是 model.User 未映射、但管理端页面会用到的列。
type userExtra struct {
	ID             int64
	Remarks        *string
	T              *int64
	OnlineCount    *int64
	LastOnlineAt   *string
	LastLoginIP    *int64
	NodeRevision   *string
	NodeRevisionAt *int64
}

// userAdminViews 批量生成用户列表项。对应 UserController::fetch：
// 预加载 plan(id,name)、invite_user(id,email)、group(id,name)、codes，并附加 total_used = u + d。
func (a *App) userAdminViews(rows []model.User) []gin.H {
	ids := make([]int64, 0, len(rows))
	planIDs, groupIDs, inviterIDs := map[int64]bool{}, map[int64]bool{}, map[int64]bool{}
	for _, u := range rows {
		ids = append(ids, u.ID)
		if u.PlanID != nil {
			planIDs[*u.PlanID] = true
		}
		if u.GroupID != nil {
			groupIDs[*u.GroupID] = true
		}
		if u.InviteUserID != nil {
			inviterIDs[*u.InviteUserID] = true
		}
	}
	keys := func(m map[int64]bool) []int64 {
		out := make([]int64, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out
	}
	plans := map[int64]gin.H{}
	if len(planIDs) > 0 {
		var ps []model.Plan
		a.S.DB.Select("id", "name").Where("id IN ?", keys(planIDs)).Find(&ps)
		for _, p := range ps {
			plans[p.ID] = gin.H{"id": p.ID, "name": p.Name}
		}
	}
	groups := map[int64]gin.H{}
	if len(groupIDs) > 0 {
		var gs []model.ServerGroup
		a.S.DB.Select("id", "name").Where("id IN ?", keys(groupIDs)).Find(&gs)
		for _, g := range gs {
			groups[g.ID] = gin.H{"id": g.ID, "name": g.Name}
		}
	}
	inviters := map[int64]gin.H{}
	if len(inviterIDs) > 0 {
		var us []model.User
		a.S.DB.Select("id", "email").Where("id IN ?", keys(inviterIDs)).Find(&us)
		for _, u := range us {
			inviters[u.ID] = gin.H{"id": u.ID, "email": u.Email}
		}
	}
	codes := map[int64][]gin.H{}
	extras := map[int64]userExtra{}
	if len(ids) > 0 {
		var cs []model.InviteCode
		a.S.DB.Select("id", "user_id", "code", "status").Where("user_id IN ?", ids).Find(&cs)
		for _, c := range cs {
			codes[c.UserID] = append(codes[c.UserID], gin.H{"id": c.ID, "user_id": c.UserID, "code": c.Code, "status": c.Status})
		}
		var ex []userExtra
		a.S.DB.Table("v2_user").
			Select("id, remarks, t, online_count, last_online_at, last_login_ip, node_revision, node_revision_at").
			Where("id IN ?", ids).Scan(&ex)
		for _, e := range ex {
			extras[e.ID] = e
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, u := range rows {
		e := extras[u.ID]
		var plan, group, inviter any
		if u.PlanID != nil {
			if v, ok := plans[*u.PlanID]; ok {
				plan = v
			}
		}
		if u.GroupID != nil {
			if v, ok := groups[*u.GroupID]; ok {
				group = v
			}
		}
		if u.InviteUserID != nil {
			if v, ok := inviters[*u.InviteUserID]; ok {
				inviter = v
			}
		}
		cs := codes[u.ID]
		if cs == nil {
			cs = []gin.H{}
		}
		t := int64(0)
		if e.T != nil {
			t = *e.T
		}
		out = append(out, gin.H{
			"id": u.ID, "invite_user_id": u.InviteUserID, "telegram_id": u.TelegramID, "email": u.Email,
			"balance": float64(u.Balance) / 100, "commission_balance": float64(u.CommissionBal) / 100,
			"u": u.U, "d": u.D, "t": t, "total_used": u.U + u.D, "transfer_enable": u.TransferEnab,
			"banned": u.Banned, "is_admin": u.IsAdmin, "is_staff": u.IsStaff, "last_login_at": u.LastLoginAt,
			"last_login_ip": e.LastLoginIP, "uuid": u.UUID, "group_id": u.GroupID, "plan_id": u.PlanID,
			"speed_limit": u.SpeedLimit, "device_limit": u.DeviceLimit, "token": u.Token,
			"expired_at": u.ExpiredAt, "device_id": u.DeviceID, "created_at": u.CreatedAt, "updated_at": u.UpdatedAt,
			"commission_type": u.CommissionType, "commission_rate": u.CommissionRate, "discount": u.Discount,
			"next_reset_at": u.NextResetAt, "last_reset_at": u.LastResetAt, "reset_count": u.ResetCount,
			"remarks": e.Remarks, "online_count": e.OnlineCount, "last_online_at": e.LastOnlineAt,
			"remind_expire": u.RemindExpire, "remind_traffic": u.RemindTraffic,
			"node_revision": e.NodeRevision, "node_revision_at": e.NodeRevisionAt,
			"referral_code": u.ReferralCode,
			"plan":          plan, "group": group, "invite_user": inviter, "codes": cs,
			"subscribe_url": a.subscribeURL(u.Token, ""),
		})
	}
	return out
}

// adminUserUpdateCols 为允许直接写入的列（金额与流量按 PHP 原值写入）。
var adminUserUpdateCols = []string{"balance", "commission_balance", "u", "d", "transfer_enable", "expired_at",
	"speed_limit", "device_limit", "is_admin", "is_staff", "commission_type", "commission_rate", "discount",
	"remarks", "uuid", "token", "telegram_id", "group_id", "plan_id", "banned", "remind_expire", "remind_traffic"}

// adminUserUpdate 对应 UserController::update。
func (a *App) adminUserUpdate(c *gin.Context) {
	m := bodyMap(c)
	id, ok := mInt(m, "id")
	if !ok || id == 0 {
		resp.Fail(c, 422, "用户ID不能为空")
		return
	}
	var u model.User
	if err := a.S.DB.First(&u, id).Error; err != nil {
		resp.Fail(c, 400, "用户不存在")
		return
	}
	upd := map[string]any{}
	for _, col := range adminUserUpdateCols {
		if v, ok := m[col]; ok {
			upd[col] = v
		}
	}
	if email, ok := mStr(m, "email"); ok && email != "" && email != u.Email {
		var n int64
		a.S.DB.Model(&model.User{}).Where("email = ?", email).Count(&n)
		if n > 0 {
			resp.Fail(c, 400, "邮箱已被使用")
			return
		}
		upd["email"] = email
	}
	if pw, ok := mStr(m, "password"); ok && pw != "" {
		h, err := auth.HashPassword(pw)
		if err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		upd["password"] = h
		upd["password_algo"] = nil
	}
	if pid, ok := mInt(m, "plan_id"); ok && pid > 0 {
		var p model.Plan
		if err := a.S.DB.First(&p, pid).Error; err != nil {
			resp.Fail(c, 400, "订阅计划不存在")
			return
		}
		upd["group_id"] = p.GroupID
	}
	if inviteEmail, _ := mStr(m, "invite_user_email"); inviteEmail != "" {
		var inv model.User
		if a.S.DB.Where("email = ?", strings.ToLower(inviteEmail)).First(&inv).Error == nil {
			upd["invite_user_id"] = inv.ID
		} else {
			upd["invite_user_id"] = nil
		}
	} else {
		upd["invite_user_id"] = nil
	}
	if banned, _ := mBool(m, "banned"); banned {
		a.S.DB.Where("tokenable_type = ? AND tokenable_id = ?", "App\\Models\\User", u.ID).Delete(&model.PersonalAccessToken{})
	}
	if len(upd) == 0 {
		resp.OK(c, true)
		return
	}
	if err := a.S.DB.Model(&u).Updates(upd).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

// adminUserDestroy 对应 destroy：删除用户及其订单、礼品卡使用记录、流量统计、工单。
func (a *App) adminUserDestroy(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if id == 0 {
		resp.Fail(c, 422, "用户ID不能为空")
		return
	}
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		tx.Model(&model.User{}).Where("id = ?", id).Count(&n)
		if n == 0 {
			return fmt.Errorf("用户不存在")
		}
		for _, q := range []string{
			"DELETE FROM v2_order WHERE user_id = ?",
			"DELETE FROM v2_gift_card_usage WHERE user_id = ?",
			"DELETE FROM v2_stat_user WHERE user_id = ?",
			"DELETE FROM v2_ticket_message WHERE user_id = ?",
			"DELETE FROM v2_ticket WHERE user_id = ?",
			"DELETE FROM v2_user WHERE id = ?",
		} {
			if err := tx.Exec(q, id).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		resp.Fail(c, 500, "删除失败")
		return
	}
	resp.OK(c, true)
}

// adminUserInfo 对应 getUserInfoById：返回用户详情及邀请人。
func (a *App) adminUserInfo(c *gin.Context) {
	id, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil || id == 0 {
		resp.Fail(c, 422, "用户ID不能为空")
		return
	}
	var u model.User
	if err := a.S.DB.First(&u, id).Error; err != nil {
		resp.Fail(c, 500, "用户不存在")
		return
	}
	out := a.userAdminView(&u)
	var inv any
	if u.InviteUserID != nil {
		var iu model.User
		if a.S.DB.Select("id", "email").First(&iu, *u.InviteUserID).Error == nil {
			inv = gin.H{"id": iu.ID, "email": iu.Email}
		}
	}
	out["invite_user"] = inv
	resp.OK(c, out)
}

// adminUserOnline 对应 getOnlineUsers：扫描 Redis 设备状态，统计每个用户的在线 IP 数。
func (a *App) adminUserOnline(c *gin.Context) {
	ctx := context.Background()
	nowTs := now()
	type entry struct {
		uid   int64
		count int
	}
	var list []entry
	var cursor uint64
	for {
		keys, next, err := a.S.Redis.Scan(ctx, cursor, devicePrefix+"*", 500).Result()
		if err != nil {
			break
		}
		for _, k := range keys {
			uid, err := strconv.ParseInt(strings.TrimPrefix(k, devicePrefix), 10, 64)
			if err != nil {
				continue
			}
			data, err := a.S.Redis.HGetAll(ctx, k).Result()
			if err != nil {
				continue
			}
			ips := map[string]struct{}{}
			for field, ts := range data {
				t, err := strconv.ParseInt(ts, 10, 64)
				if err != nil || nowTs-t > deviceTTL {
					continue
				}
				if _, ip, ok := strings.Cut(field, ":"); ok {
					ips[ip] = struct{}{}
				}
			}
			if len(ips) > 0 {
				list = append(list, entry{uid, len(ips)})
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	if len(list) == 0 {
		resp.OK(c, gin.H{"total": 0, "users": []gin.H{}})
		return
	}
	ids := make([]int64, 0, len(list))
	for _, e := range list {
		ids = append(ids, e.uid)
	}
	var users []model.User
	a.S.DB.Select("id", "email", "device_limit", "plan_id", "expired_at").Where("id IN ?", ids).Find(&users)
	byID := map[int64]model.User{}
	for _, u := range users {
		byID[u.ID] = u
	}
	out := make([]gin.H, 0, len(list))
	for _, e := range list {
		u, ok := byID[e.uid]
		email := "—"
		var limit int64
		if ok {
			email = u.Email
			limit = derefInt64(u.DeviceLimit)
		}
		out = append(out, gin.H{"user_id": e.uid, "email": email, "device_count": e.count,
			"device_limit": limit, "expired_at": u.ExpiredAt})
	}
	// 按在线设备数降序，与 PHP usort 一致。
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j]["device_count"].(int) > out[j-1]["device_count"].(int); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	resp.OK(c, gin.H{"total": len(out), "users": out})
}

// adminUserGenerate 对应 UserController::generate / multiGenerate / multiGenerateWithPrefix。
//   - email_prefix + generate_count：prefix_1@suffix … prefix_N@suffix
//   - email_prefix 单个：prefix@suffix
//   - 仅 generate_count：随机 6 位@suffix
//
// 密码缺省为邮箱本身；单个返回 true，批量返回 {code, message, data}（无 status 外壳）或 CSV。
func (a *App) adminUserGenerate(c *gin.Context) {
	m := bodyMap(c)
	suffix, _ := mStr(m, "email_suffix")
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		resp.Fail(c, 422, "email_suffix 不能为空")
		return
	}
	count, _ := mInt(m, "generate_count")
	if count > 500 {
		resp.Fail(c, 422, "生成数量最大为500个")
		return
	}
	prefix, _ := mStr(m, "email_prefix")
	prefix = strings.TrimSpace(prefix)
	password, hasPassword := mStr(m, "password")
	hasPassword = hasPassword && password != ""

	var emails []string
	single := false
	switch {
	case prefix != "" && count > 0:
		for i := int64(1); i <= count; i++ {
			emails = append(emails, fmt.Sprintf("%s_%d@%s", prefix, i, suffix))
		}
	case prefix != "":
		emails = []string{prefix + "@" + suffix}
		single = true
	case count > 0:
		for i := int64(0); i < count; i++ {
			emails = append(emails, randomChars(6)+"@"+suffix)
		}
	default:
		resp.Fail(c, 422, "请填写邮箱前缀或生成数量")
		return
	}
	for _, e := range emails {
		var n int64
		a.S.DB.Model(&model.User{}).Where("email = ?", e).Count(&n)
		if n > 0 {
			if single {
				resp.Fail(c, 400, "邮箱已存在于系统中")
			} else {
				resp.Fail(c, 400, "邮箱 "+e+" 已存在于系统中")
			}
			return
		}
	}

	var plan *model.Plan
	if pid, ok := mInt(m, "plan_id"); ok && pid > 0 {
		var p model.Plan
		if a.S.DB.First(&p, pid).Error != nil {
			resp.Fail(c, 500, "订阅计划不存在")
			return
		}
		plan = &p
	}
	expiredAt, hasExpire := mInt(m, "expired_at")
	remind := a.Set.Bool("default_remind_expire", true)
	remindT := a.Set.Bool("default_remind_traffic", true)
	devLimit := int64(a.Set.Int("default_device_limit", 4))

	created := make([]model.User, 0, len(emails))
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		for _, e := range emails {
			pw := password
			if !hasPassword {
				pw = e
			}
			hash, err := auth.HashPassword(pw)
			if err != nil {
				return err
			}
			ts := now()
			u := model.User{Email: e, Password: hash, UUID: uuid.NewString(), Token: auth.RandHex(16), CreatedAt: ts, UpdatedAt: ts}
			u.RemindExpire, u.RemindTraffic = &remind, &remindT
			if devLimit > 0 {
				dl := devLimit
				u.DeviceLimit = &dl
			}
			if plan != nil {
				pid := plan.ID
				u.PlanID = &pid
				u.GroupID = plan.GroupID
				u.TransferEnab = shop.GbToTransfer(derefInt64(plan.TransferEnable))
				u.SpeedLimit = plan.SpeedLimit
				if plan.DeviceLimit != nil {
					u.DeviceLimit = plan.DeviceLimit
				}
			}
			if hasExpire && expiredAt > 0 {
				ex := expiredAt
				u.ExpiredAt = &ex
			}
			if err := tx.Create(&u).Error; err != nil {
				return err
			}
			created = append(created, u)
		}
		return nil
	})
	if err != nil {
		resp.Fail(c, 500, "生成失败")
		return
	}
	if single {
		resp.OK(c, true)
		return
	}

	pwOf := func(u model.User) string {
		if hasPassword {
			return password
		}
		return u.Email
	}
	expireText := func(u model.User) string {
		if u.ExpiredAt == nil {
			return "长期有效"
		}
		return time.Unix(*u.ExpiredAt, 0).Format("2006-01-02 15:04:05")
	}
	if dl, _ := m["download_csv"].(bool); dl || m["download_csv"] == "1" || m["download_csv"] == float64(1) {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		_ = w.Write([]string{"账号", "密码", "过期时间", "UUID", "创建时间", "订阅地址"})
		for _, u := range created {
			_ = w.Write([]string{u.Email, pwOf(u), expireText(u), u.UUID,
				time.Unix(u.CreatedAt, 0).Format("2006-01-02 15:04:05"), a.subscribeURL(u.Token, "")})
		}
		w.Flush()
		c.Header("Content-Disposition", `attachment; filename="users.csv"`)
		c.Data(http.StatusOK, "text/csv", buf.Bytes())
		return
	}
	data := make([]gin.H, 0, len(created))
	for _, u := range created {
		data = append(data, gin.H{"email": u.Email, "password": pwOf(u), "expired_at": expireText(u), "uuid": u.UUID,
			"created_at": time.Unix(u.CreatedAt, 0).Format("2006-01-02 15:04:05"), "subscribe_url": a.subscribeURL(u.Token, "")})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "批量生成成功", "data": data})
}

func randomDigits(n int) string {
	out := make([]byte, n)
	for i := range out {
		v, _ := rand.Int(rand.Reader, big.NewInt(10))
		out[i] = byte('0' + v.Int64())
	}
	return string(out)
}
