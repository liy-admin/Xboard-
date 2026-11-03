package api

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 礼品卡类型（v2_gift_card_template.type）与状态（v2_gift_card_code.status）。
const (
	giftTypeGeneral = 1
	giftTypePlan    = 2
	giftTypeMystery = 3

	giftCodeUnused   = 0
	giftCodeUsed     = 1
	giftCodeExpired  = 2
	giftCodeDisabled = 3

	giftSourceCard = "gift_card"
	orderCompleted = 3
)

var giftTypeNames = map[int]string{giftTypeGeneral: "通用礼品卡", giftTypePlan: "套餐礼品卡", giftTypeMystery: "盲盒礼品卡"}

var giftStatusNames = map[int]string{
	giftCodeUnused: "未使用", giftCodeUsed: "已使用", giftCodeExpired: "已过期", giftCodeDisabled: "已禁用",
}

// bizError 表示可直接返回给用户的业务错误（对应 PHP ApiException）。
type bizError string

func (e bizError) Error() string { return string(e) }

func giftTypeName(t int) string {
	if n, ok := giftTypeNames[t]; ok {
		return n
	}
	return "未知类型"
}

func giftStatusName(s int) string {
	if n, ok := giftStatusNames[s]; ok {
		return n
	}
	return "未知状态"
}

// rawMap 解析 JSON 配置列为 map，空或非法时返回空 map。
func rawMap(raw json.RawMessage) map[string]any {
	m := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m
}

// isset 等价于 PHP isset：键存在且值非 null。
func isset(m map[string]any, k string) bool {
	v, ok := m[k]
	return ok && v != nil
}

// toNum 把 JSON 解析出的数值统一为 float64。
func toNum(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		return f, err == nil
	}
	return 0, false
}

func numOr(m map[string]any, k string, def float64) float64 {
	if v, ok := m[k]; ok {
		if f, ok := toNum(v); ok {
			return f
		}
	}
	return def
}

// userIsActive 对应 User::isActive。
func userIsActive(u *model.User, nowTs int64) bool {
	return !u.Banned && (u.ExpiredAt == nil || *u.ExpiredAt > nowTs) && u.PlanID != nil
}

// templateUserConditions 对应 GiftCardTemplate::checkUserConditions。
func templateUserConditions(t *model.GiftCardTemplate, u *model.User, nowTs int64, paidOrder bool) bool {
	switch t.Type {
	case giftTypeGeneral:
		r := rawMap(t.Rewards)
		if isset(r, "transfer_enable") || isset(r, "expire_days") || isset(r, "reset_package") {
			if u.PlanID == nil {
				return false
			}
		}
	case giftTypePlan:
		if userIsActive(u, nowTs) {
			return false
		}
	}
	c := rawMap(t.Conditions)
	if b, _ := c["new_user_only"].(bool); b {
		maxDays := numOr(c, "new_user_max_days", 7)
		if float64(u.CreatedAt) < float64(nowTs)-maxDays*86400 {
			return false
		}
	}
	if b, _ := c["paid_user_only"].(bool); b && !paidOrder {
		return false
	}
	if allowed, ok := c["allowed_plans"].([]any); ok && u.PlanID != nil {
		found := false
		for _, p := range allowed {
			if f, ok := toNum(p); ok && int64(f) == *u.PlanID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if b, _ := c["require_invite"].(bool); b && (u.InviteUserID == nil || *u.InviteUserID == 0) {
		return false
	}
	return true
}

// templateUsageAllowed 对应 GiftCardTemplate::checkUsageLimit。used 为该用户历史使用次数，
// lastUsedAt 为最近一次使用时间（无记录时传 0）。
func templateUsageAllowed(t *model.GiftCardTemplate, used int64, lastUsedAt, nowTs int64) bool {
	l := rawMap(t.Limits)
	if isset(l, "max_use_per_user") && used >= int64(numOr(l, "max_use_per_user", 0)) {
		return false
	}
	if isset(l, "cooldown_hours") && lastUsedAt > 0 {
		until := lastUsedAt + int64(numOr(l, "cooldown_hours", 0)*3600)
		if nowTs < until {
			return false
		}
	}
	return true
}

// festivalWindow 返回节日配置是否在生效期，以及倍率（缺省 1.0）。
func festivalWindow(sc map[string]any, nowTs int64) (bool, float64) {
	if !isset(sc, "start_time") || !isset(sc, "end_time") {
		return false, 1
	}
	start, end := numOr(sc, "start_time", 0), numOr(sc, "end_time", 0)
	if float64(nowTs) < start || float64(nowTs) > end {
		return false, 1
	}
	return true, numOr(sc, "festival_bonus", 1)
}

// calcActualRewards 对应 GiftCardTemplate::calculateActualRewards。
// 盲盒按权重抽取 random_rewards 中的一项合并进奖励；节日倍率作用于数值奖励，但跳过 plan_id（修正 PHP 的缺陷）。
func calcActualRewards(t *model.GiftCardTemplate, nowTs int64, pick func(n int) int) map[string]any {
	base := rawMap(t.Rewards)
	actual := make(map[string]any, len(base))
	for k, v := range base {
		actual[k] = v
	}
	if t.Type == giftTypeMystery {
		if list, ok := base["random_rewards"].([]any); ok && len(list) > 0 {
			total := 0.0
			for _, it := range list {
				if m, ok := it.(map[string]any); ok {
					total += numOr(m, "weight", 0)
				}
			}
			if total > 0 {
				r := float64(pick(int(total))) + 1
				cum := 0.0
				for _, it := range list {
					m, ok := it.(map[string]any)
					if !ok {
						continue
					}
					cum += numOr(m, "weight", 0)
					if r <= cum {
						for k, v := range m {
							if k != "weight" {
								actual[k] = v
							}
						}
						break
					}
				}
			}
		}
	}
	sc := rawMap(t.SpecialConfig)
	if isset(sc, "festival_bonus") {
		if in, bonus := festivalWindow(sc, nowTs); in && bonus > 1 {
			for k, v := range actual {
				if k == "plan_id" {
					continue
				}
				if f, ok := toNum(v); ok {
					actual[k] = int64(f * bonus)
				}
			}
		}
	}
	return actual
}

// giftCodeAvailable 对应 GiftCardCode::isAvailable。
func giftCodeAvailable(c *model.GiftCardCode, nowTs int64) bool {
	if c.Status == giftCodeExpired || c.Status == giftCodeDisabled {
		return false
	}
	if c.ExpiresAt != nil && *c.ExpiresAt < nowTs {
		return false
	}
	return c.UsageCount < c.MaxUsage
}

// giftValidateActive 对应 validateIsActive。
func (a *App) giftValidateActive(code *model.GiftCardCode, tpl *model.GiftCardTemplate, nowTs int64) error {
	if !tpl.Status {
		return bizError("该礼品卡类型已停用")
	}
	if !giftCodeAvailable(code, nowTs) {
		return bizError("兑换码不可用：" + giftStatusName(code.Status))
	}
	return nil
}

// giftEligibility 对应 GiftCardService::checkUserEligibility（需要数据库查询的部分由调用方给出）。
func giftEligibility(tpl *model.GiftCardTemplate, u *model.User, nowTs int64, paid bool, used int64, last int64) (bool, string) {
	if !templateUserConditions(tpl, u, nowTs, paid) {
		return false, "您不满足此礼品卡的使用条件"
	}
	if !templateUsageAllowed(tpl, used, last, nowTs) {
		return false, "您已达到此礼品卡的使用限制"
	}
	return true, ""
}

func (a *App) giftLoad(code string) (*model.GiftCardCode, *model.GiftCardTemplate, error) {
	var c model.GiftCardCode
	if err := a.S.DB.Where("code = ?", code).First(&c).Error; err != nil {
		return nil, nil, bizError("兑换码不存在")
	}
	var t model.GiftCardTemplate
	if err := a.S.DB.First(&t, c.TemplateID).Error; err != nil {
		return nil, nil, bizError("兑换码不存在")
	}
	return &c, &t, nil
}

// giftEligibilityFor 查询用户在该模板下的付费状态与使用记录后判定资格。
func (a *App) giftEligibilityFor(tpl *model.GiftCardTemplate, u *model.User, nowTs int64) (bool, string) {
	var paidN, used, last int64
	a.S.DB.Model(&model.Order{}).Where("user_id = ? AND status = ?", u.ID, orderCompleted).Count(&paidN)
	var usages []model.GiftCardUsage
	a.S.DB.Where("template_id = ? AND user_id = ?", tpl.ID, u.ID).Order("created_at DESC").Find(&usages)
	used = int64(len(usages))
	if len(usages) > 0 {
		last = usages[0].CreatedAt
	}
	return giftEligibility(tpl, u, nowTs, paidN > 0, used, last)
}

func (a *App) giftCheck(c *gin.Context) {
	u := cur(c)
	var in struct {
		Code string `json:"code" form:"code" binding:"required"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	nowTs := now()
	code, tpl, err := a.giftLoad(in.Code)
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	if err := a.giftValidateActive(code, tpl, nowTs); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	can, reason := a.giftEligibilityFor(tpl, u, nowTs)
	info := gin.H{
		"code": code.Code,
		"template": gin.H{
			"name": tpl.Name, "description": tpl.Description, "type": tpl.Type, "type_name": giftTypeName(tpl.Type),
			"icon": tpl.Icon, "background_image": tpl.BackgroundImage, "theme_color": tpl.ThemeColor,
		},
		"status": code.Status, "status_name": giftStatusName(code.Status), "expires_at": code.ExpiresAt,
		"usage_count": code.UsageCount, "max_usage": code.MaxUsage,
	}
	if tpl.Type == giftTypePlan {
		if pid, ok := rawMap(tpl.Rewards)["plan_id"]; ok {
			if f, ok := toNum(pid); ok {
				var p model.Plan
				if a.S.DB.First(&p, int64(f)).Error == nil {
					info["plan_info"] = planResource(&p, a.systemResetMethod())
				}
			}
		}
	}
	var reason0 any
	if reason != "" {
		reason0 = reason
	}
	resp.OK(c, gin.H{
		"code_info":      info,
		"reward_preview": calcActualRewards(tpl, nowTs, randIndex),
		"can_redeem":     can, "reason": reason0,
	})
}

func (a *App) giftRedeem(c *gin.Context) {
	u := cur(c)
	var in struct {
		Code string `json:"code" form:"code" binding:"required"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	nowTs := now()
	code, tpl, err := a.giftLoad(in.Code)
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	if err := a.giftValidateActive(code, tpl, nowTs); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	if ok, reason := a.giftEligibilityFor(tpl, u, nowTs); !ok {
		resp.Fail(c, 400, reason)
		return
	}
	rewards, inv, err := a.giftApply(code, tpl, u.ID, c.Request.UserAgent(), nowTs)
	if err != nil {
		if be, ok := err.(bizError); ok {
			resp.Fail(c, 400, be.Error())
			return
		}
		resp.Fail(c, 500, "兑换失败，请稍后重试")
		return
	}
	resp.OK(c, gin.H{
		"message": "兑换成功！", "rewards": rewards, "invite_rewards": inv, "template_name": tpl.Name,
	})
}

// giftApply 在事务内完成发放、邀请人返利、码标记与使用记录（对应 GiftCardService::redeem）。
func (a *App) giftApply(code *model.GiftCardCode, tpl *model.GiftCardTemplate, uid int64, ua string, nowTs int64) (map[string]any, map[string]any, error) {
	var rewards, inviteRewards map[string]any
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, uid).Error; err != nil {
			return bizError("用户不存在")
		}
		rewards = calcActualRewards(tpl, nowTs, randIndex)
		if tpl.Type == giftTypeMystery {
			b, _ := json.Marshal(rewards)
			if err := tx.Model(code).Update("actual_rewards", json.RawMessage(b)).Error; err != nil {
				return err
			}
		}
		if err := giveGiftRewards(tx, a, &user, rewards, nowTs); err != nil {
			return err
		}
		if user.InviteUserID != nil && isset(rewards, "invite_reward_rate") {
			inviteRewards = giveInviteGiftRewards(tx, *user.InviteUserID, rewards)
		}
		if err := tx.Model(code).Updates(map[string]any{
			"status": giftCodeUsed, "user_id": uid, "used_at": nowTs,
			"usage_count": gorm.Expr("usage_count + 1"), "updated_at": nowTs,
		}).Error; err != nil {
			return err
		}
		var planSort *int64
		if user.PlanID != nil {
			var p model.Plan
			if tx.First(&p, *user.PlanID).Error == nil && p.Sort != nil {
				s := int64(*p.Sort)
				planSort = &s
			}
		}
		rb, _ := json.Marshal(rewards)
		ib, _ := json.Marshal(inviteRewards)

		return tx.Create(&model.GiftCardUsage{
			CodeID: code.ID, TemplateID: code.TemplateID, UserID: uid, InviteUserID: user.InviteUserID,
			RewardsGiven: rb, InviteRewards: ib, UserLevelAtUse: planSort, PlanIDAtUse: user.PlanID,
			MultiplierApplied: festivalMultiplier(tpl, nowTs), UserAgent: &ua, CreatedAt: nowTs,
		}).Error
	})
	return rewards, inviteRewards, err
}

// festivalMultiplier 对应 GiftCardService::calculateMultiplier。
func festivalMultiplier(tpl *model.GiftCardTemplate, nowTs int64) float64 {
	_, bonus := festivalWindow(rawMap(tpl.SpecialConfig), nowTs)
	return bonus
}

// giveGiftRewards 对应 GiftCardService::giveRewards。余额与流量等为累加，使用原子表达式以免覆盖并发写。
func giveGiftRewards(tx *gorm.DB, a *App, user *model.User, r map[string]any, nowTs int64) error {
	if b := numOr(r, "balance", 0); b > 0 {
		if err := tx.Model(&model.User{}).Where("id = ?", user.ID).
			UpdateColumn("balance", gorm.Expr("balance + ?", int64(b))).Error; err != nil {
			return err
		}
	}
	if t := numOr(r, "transfer_enable", 0); t > 0 {
		user.TransferEnab += int64(t)
	}
	if d := numOr(r, "device_limit", 0); d > 0 {
		v := derefInt64(user.DeviceLimit) + int64(d)
		user.DeviceLimit = &v
	}
	if b, _ := r["reset_package"].(bool); b && user.PlanID != nil {
		if err := (&shop.Service{DB: tx, Set: a.Set}).PerformReset(tx, user, giftSourceCard); err != nil {
			return err
		}
	}
	if isset(r, "plan_id") {
		if f, ok := toNum(r["plan_id"]); ok {
			var p model.Plan
			if tx.First(&p, int64(f)).Error == nil {
				pid := p.ID
				user.PlanID = &pid
				user.GroupID = p.GroupID
				user.TransferEnab = shop.GbToTransfer(derefInt64(p.TransferEnable))
				user.SpeedLimit = p.SpeedLimit
				user.DeviceLimit = p.DeviceLimit
				if days := numOr(r, "plan_validity_days", 0); days > 0 {
					exp := extendExpiry(user.ExpiredAt, nowTs, int64(days)*86400)
					user.ExpiredAt = &exp
				}
			}
		}
	} else if days := numOr(r, "expire_days", 0); days > 0 {
		permanent := user.PlanID != nil && user.ExpiredAt == nil
		if !permanent {
			exp := extendExpiry(user.ExpiredAt, nowTs, int64(days)*86400)
			user.ExpiredAt = &exp
		}
	}
	return tx.Omit("balance").Save(user).Error
}

// giveInviteGiftRewards 对应 GiftCardService::giveInviteRewards，返回邀请人实际获得的奖励。
func giveInviteGiftRewards(tx *gorm.DB, inviteID int64, r map[string]any) map[string]any {
	var inv model.User
	if tx.First(&inv, inviteID).Error != nil {
		return nil
	}
	rate := numOr(r, "invite_reward_rate", 0.2)
	out := map[string]any{}
	if b := numOr(r, "balance", 0); b > 0 {
		if ib := int64(b * rate); ib > 0 {
			tx.Model(&model.User{}).Where("id = ?", inv.ID).UpdateColumn("balance", gorm.Expr("balance + ?", ib))
			out["balance"] = ib
		}
	}
	if t := numOr(r, "transfer_enable", 0); t > 0 {
		if it := int64(t * rate); it > 0 {
			tx.Model(&model.User{}).Where("id = ?", inv.ID).UpdateColumn("transfer_enable", gorm.Expr("transfer_enable + ?", it))
			out["transfer_enable"] = it
		}
	}
	return out
}

func maskGiftCode(code string) string {
	if code == "" {
		return ""
	}
	if len(code) > 8 {
		return code[:8] + "****"
	}
	return code + "****"
}

func (a *App) giftHistory(c *gin.Context) {
	u := cur(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	per, _ := strconv.Atoi(c.DefaultQuery("per_page", "15"))
	if per < 1 || per > 100 {
		per = 15
	}
	q := a.S.DB.Model(&model.GiftCardUsage{}).Where("user_id = ?", u.ID)
	var total int64
	q.Count(&total)
	var rows []model.GiftCardUsage
	q.Order("created_at DESC").Offset((page - 1) * per).Limit(per).Find(&rows)

	data := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var code model.GiftCardCode
		var tpl model.GiftCardTemplate
		a.S.DB.First(&code, r.CodeID)
		a.S.DB.First(&tpl, r.TemplateID)
		data = append(data, gin.H{
			"id": r.ID, "code": maskGiftCode(code.Code), "template_name": tpl.Name,
			"template_type": tpl.Type, "template_type_name": giftTypeName(tpl.Type),
			"rewards_given": r.RewardsGiven, "invite_rewards": r.InviteRewards,
			"multiplier_applied": r.MultiplierApplied, "created_at": r.CreatedAt,
		})
	}
	last := int((total + int64(per) - 1) / int64(per))
	if last < 1 {
		last = 1
	}
	c.JSON(200, gin.H{
		"data":       data,
		"pagination": gin.H{"current_page": page, "last_page": last, "per_page": per, "total": total},
	})
}

func (a *App) giftDetail(c *gin.Context) {
	u := cur(c)
	id, err := strconv.ParseInt(c.Query("id"), 10, 64)
	if err != nil || id < 1 {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	var r model.GiftCardUsage
	if err := a.S.DB.Where("user_id = ? AND id = ?", u.ID, id).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resp.Fail(c, 404, "记录不存在")
			return
		}
		resp.Fail(c, 500, "Internal error")
		return
	}
	var code model.GiftCardCode
	var tpl model.GiftCardTemplate
	a.S.DB.First(&code, r.CodeID)
	a.S.DB.First(&tpl, r.TemplateID)
	var invite any
	if r.InviteUserID != nil {
		var iu model.User
		if a.S.DB.First(&iu, *r.InviteUserID).Error == nil {
			email := ""
			if len(iu.Email) >= 3 {
				email = iu.Email[:3] + "***@***"
			}
			invite = gin.H{"id": iu.ID, "email": email}
		}
	}
	desc := ""
	if tpl.Description != nil {
		desc = *tpl.Description
	}
	resp.OK(c, gin.H{
		"id": r.ID, "code": code.Code,
		"template": gin.H{
			"name": tpl.Name, "description": desc, "type": tpl.Type, "type_name": giftTypeName(tpl.Type),
			"icon": deref(tpl.Icon), "theme_color": tpl.ThemeColor,
		},
		"rewards_given": r.RewardsGiven, "invite_rewards": r.InviteRewards, "invite_user": invite,
		"user_level_at_use": r.UserLevelAtUse, "plan_id_at_use": r.PlanIDAtUse,
		"multiplier_applied": r.MultiplierApplied, "notes": r.Notes, "created_at": r.CreatedAt,
	})
}

func (a *App) giftTypes(c *gin.Context) {
	resp.OK(c, gin.H{"types": giftTypeNames})
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
