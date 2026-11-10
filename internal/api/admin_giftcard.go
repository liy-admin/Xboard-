package api

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
)

// 管理端礼品卡：模板 CRUD、兑换码生成与管理、使用记录、类型表。
// 对应 V2/Admin/GiftCardController。未包含：exportCodes（CSV 下载）、statistics 的图表聚合。

var giftCodePrefixRe = regexp.MustCompile(`^[A-Z0-9]*$`)
var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// jsonOrNull 把 map/array 转为存库的 JSON；nil 保持 NULL。
func jsonOrNull(v any) any {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func (a *App) giftTemplateView(t model.GiftCardTemplate) gin.H {
	var codes, used int64
	a.S.DB.Model(&model.GiftCardCode{}).Where("template_id = ?", t.ID).Count(&codes)
	a.S.DB.Model(&model.GiftCardUsage{}).Where("template_id = ?", t.ID).Count(&used)
	return gin.H{
		"id": t.ID, "name": t.Name, "description": t.Description, "type": t.Type, "type_name": giftTypeName(t.Type),
		"status": t.Status, "conditions": rawMap(t.Conditions), "rewards": rawMap(t.Rewards),
		"limits": rawMap(t.Limits), "special_config": rawMap(t.SpecialConfig), "icon": t.Icon,
		"background_image": t.BackgroundImage, "theme_color": t.ThemeColor, "sort": t.Sort, "admin_id": t.AdminID,
		"created_at": t.CreatedAt, "updated_at": t.UpdatedAt, "codes_count": codes, "used_count": used,
	}
}

func (a *App) giftTemplates(c *gin.Context) {
	cur, size := pageOf(c)
	q := a.S.DB.Model(&model.GiftCardTemplate{})
	if v := c.Query("type"); v != "" {
		q = q.Where("type = ?", v)
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	var total int64
	q.Count(&total)
	var rows []model.GiftCardTemplate
	q.Order("sort ASC").Order("created_at DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, t := range rows {
		out = append(out, a.giftTemplateView(t))
	}
	paginateJSON(c, total, cur, size, out)
}

// giftTemplateInput 解析模板字段（创建与更新共用）。
func giftTemplateInput(m map[string]any) map[string]any {
	out := map[string]any{}
	if v, ok := mStr(m, "name"); ok {
		out["name"] = v
	}
	if v, ok := mStr(m, "description"); ok {
		out["description"] = v
	}
	if v, ok := mInt(m, "type"); ok {
		out["type"] = v
	}
	if v, ok := mBool(m, "status"); ok {
		out["status"] = v
	}
	for _, k := range []string{"conditions", "rewards", "limits", "special_config"} {
		if v, ok := m[k]; ok {
			out[k] = jsonOrNull(v)
		}
	}
	if v, ok := mStr(m, "icon"); ok {
		out["icon"] = v
	}
	if v, ok := mStr(m, "background_image"); ok {
		out["background_image"] = v
	}
	if v, ok := mStr(m, "theme_color"); ok {
		out["theme_color"] = v
	}
	if v, ok := mInt(m, "sort"); ok {
		out["sort"] = v
	}
	return out
}

func (a *App) giftTemplateCreate(c *gin.Context) {
	m := bodyMap(c)
	name, _ := mStr(m, "name")
	if name == "" {
		resp.Fail(c, 422, "礼品卡名称不能为空")
		return
	}
	typ, ok := mInt(m, "type")
	if !ok || giftTypeNames[int(typ)] == "" {
		resp.Fail(c, 422, "无效的礼品卡类型")
		return
	}
	if m["rewards"] == nil {
		resp.Fail(c, 422, "奖励配置不能为空")
		return
	}
	in := giftTemplateInput(m)
	if tc, _ := mStr(m, "theme_color"); tc != "" && !hexColorRe.MatchString(tc) {
		resp.Fail(c, 422, "主题色格式不正确")
		return
	}
	if _, ok := in["status"]; !ok {
		in["status"] = true
	}
	if _, ok := in["theme_color"]; !ok {
		in["theme_color"] = "#1890ff"
	}
	if _, ok := in["sort"]; !ok {
		in["sort"] = 0
	}
	ts := time.Now().Unix()
	in["admin_id"] = cur(c).ID
	in["created_at"], in["updated_at"] = ts, ts
	if err := a.S.DB.Model(&model.GiftCardTemplate{}).Create(in).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) giftTemplateUpdate(c *gin.Context) {
	m := bodyMap(c)
	id, ok := mInt(m, "id")
	if !ok || id == 0 {
		resp.Fail(c, 422, "模板ID不能为空")
		return
	}
	var t model.GiftCardTemplate
	if err := a.S.DB.First(&t, id).Error; err != nil {
		resp.Fail(c, 404, "模板不存在")
		return
	}
	in := giftTemplateInput(m)
	in["updated_at"] = time.Now().Unix()
	if err := a.S.DB.Model(&t).Updates(in).Error; err != nil {
		resp.Fail(c, 500, "更新失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) giftTemplateDelete(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var t model.GiftCardTemplate
	if err := a.S.DB.First(&t, id).Error; err != nil {
		resp.Fail(c, 404, "模板不存在")
		return
	}
	var n int64
	a.S.DB.Model(&model.GiftCardCode{}).Where("template_id = ?", id).Count(&n)
	if n > 0 {
		resp.Fail(c, 400, "该模板下存在兑换码，无法删除")
		return
	}
	if err := a.S.DB.Delete(&t).Error; err != nil {
		resp.Fail(c, 500, "删除失败")
		return
	}
	resp.OK(c, true)
}

// giftGenerateCodes 对应 generateCodes（不含 CSV 下载）：批量生成兑换码，返回批次号。
func (a *App) giftGenerateCodes(c *gin.Context) {
	m := bodyMap(c)
	tid, ok := mInt(m, "template_id")
	if !ok || tid == 0 {
		resp.Fail(c, 422, "请选择礼品卡模板")
		return
	}
	count, ok := mInt(m, "count")
	if !ok || count < 1 || count > 10000 {
		resp.Fail(c, 422, "单次最多生成10000个兑换码")
		return
	}
	prefix, _ := mStr(m, "prefix")
	if prefix == "" {
		prefix = "GC"
	}
	if !giftCodePrefixRe.MatchString(prefix) || len(prefix) > 10 {
		resp.Fail(c, 422, "前缀只能包含大写字母和数字")
		return
	}
	var t model.GiftCardTemplate
	if err := a.S.DB.First(&t, tid).Error; err != nil {
		resp.Fail(c, 422, "请选择礼品卡模板")
		return
	}
	if !t.Status {
		resp.Fail(c, 400, "模板已被禁用")
		return
	}
	maxUsage := int64(1)
	if v, ok := mInt(m, "max_usage"); ok && v >= 1 {
		maxUsage = v
	}
	var expires *int64
	if h, ok := mInt(m, "expires_hours"); ok && h >= 1 {
		e := time.Now().Unix() + h*3600
		expires = &e
	}
	batch := "batch_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	ts := time.Now().Unix()
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		for i := int64(0); i < count; i++ {
			row := map[string]any{
				"template_id": tid, "code": prefix + strings.ToUpper(auth.RandHex(4)), "batch_id": batch,
				"status": giftCodeUnused, "expires_at": expires, "max_usage": maxUsage, "usage_count": 0,
				"created_at": ts, "updated_at": ts,
			}
			if err := tx.Model(&model.GiftCardCode{}).Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		resp.Fail(c, 500, "生成失败")
		return
	}
	resp.OK(c, gin.H{"batch_id": batch, "count": count, "message": "生成成功"})
}

// giftCodeView 对应 codes 列表项：用户邮箱做脱敏。
func (a *App) giftCodeView(code model.GiftCardCode, tplName string) gin.H {
	var email any
	if code.UserID != nil {
		var u model.User
		if a.S.DB.Select("email").First(&u, *code.UserID).Error == nil {
			email = maskEmail(u.Email)
		}
	}
	return gin.H{
		"id": code.ID, "template_id": code.TemplateID, "template_name": tplName, "code": code.Code,
		"batch_id": code.BatchID, "status": code.Status, "status_name": giftStatusName(code.Status),
		"user_id": code.UserID, "user_email": email, "used_at": code.UsedAt, "expires_at": code.ExpiresAt,
		"usage_count": code.UsageCount, "max_usage": code.MaxUsage, "created_at": code.CreatedAt,
	}
}

// maskEmail 对应 substr(email,0,3).'***@***'。
func maskEmail(e string) string {
	if len(e) < 3 {
		return e + "***@***"
	}
	return e[:3] + "***@***"
}

// giftCodeSort / giftUsageSort 兑换码与使用记录列表可排序的列（列键 → SQL 表达式）。
var giftCodeSort = map[string]string{
	"id": "v2_gift_card_code.id", "code": "v2_gift_card_code.code", "status": "v2_gift_card_code.status",
	"expires_at": "v2_gift_card_code.expires_at", "usage_count": "v2_gift_card_code.usage_count",
	"max_usage": "v2_gift_card_code.max_usage", "created_at": "v2_gift_card_code.created_at",
	"template_name": "(SELECT t.name FROM v2_gift_card_template t WHERE t.id = v2_gift_card_code.template_id)",
}

var giftCodeFilter = map[string]bool{"code": true, "status": true, "template_id": true, "batch_id": true}

var giftUsageSort = map[string]string{
	"id": "v2_gift_card_usage.id", "multiplier_applied": "v2_gift_card_usage.multiplier_applied",
	"ip_address": "v2_gift_card_usage.ip_address", "created_at": "v2_gift_card_usage.created_at",
	"rewards_given": "CAST(v2_gift_card_usage.rewards_given AS CHAR)", "invite_rewards": "CAST(v2_gift_card_usage.invite_rewards AS CHAR)",
	"code":          "(SELECT gc.code FROM v2_gift_card_code gc WHERE gc.id = v2_gift_card_usage.code_id)",
	"template_name": "(SELECT t.name FROM v2_gift_card_template t WHERE t.id = v2_gift_card_usage.template_id)",
	"user_email":    "(SELECT u.email FROM v2_user u WHERE u.id = v2_gift_card_usage.user_id)",
}

func (a *App) giftCodes(c *gin.Context) {
	cur, size := pageOf(c)
	q := a.S.DB.Model(&model.GiftCardCode{})
	if v := c.Query("template_id"); v != "" {
		q = q.Where("template_id = ?", v)
	}
	if v := c.Query("batch_id"); v != "" {
		q = q.Where("batch_id = ?", v)
	}
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	// 页面把兑换码搜索、状态、模板放在 filter 数组里提交。
	q = adminFilters(c, giftCodeFilter, q)
	var total int64
	q.Count(&total)
	var rows []model.GiftCardCode
	adminSortExprs(c, giftCodeSort, q).Order("v2_gift_card_code.created_at DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, code := range rows {
		var t model.GiftCardTemplate
		a.S.DB.Select("name").First(&t, code.TemplateID)
		out = append(out, a.giftCodeView(code, t.Name))
	}
	paginateJSON(c, total, cur, size, out)
}

// giftCodeToggle 对应 toggleCode：disable 置为已禁用；enable 仅对已禁用的码恢复为未使用。
func (a *App) giftCodeToggle(c *gin.Context) {
	m := bodyMap(c)
	id, _ := mInt(m, "id")
	action, _ := mStr(m, "action")
	var code model.GiftCardCode
	if err := a.S.DB.First(&code, id).Error; err != nil {
		resp.Fail(c, 404, "兑换码不存在")
		return
	}
	switch action {
	case "disable":
		a.S.DB.Model(&code).Update("status", giftCodeDisabled)
		resp.OK(c, gin.H{"message": "已禁用"})
	case "enable":
		if code.Status == giftCodeDisabled {
			a.S.DB.Model(&code).Update("status", giftCodeUnused)
		}
		resp.OK(c, gin.H{"message": "已启用"})
	default:
		resp.Fail(c, 422, "无效的操作")
	}
}

// giftCodeUpdate 对应 updateCode：允许修改有效期与最大使用次数。
func (a *App) giftCodeUpdate(c *gin.Context) {
	m := bodyMap(c)
	id, _ := mInt(m, "id")
	var code model.GiftCardCode
	if err := a.S.DB.First(&code, id).Error; err != nil {
		resp.Fail(c, 404, "兑换码不存在")
		return
	}
	upd := map[string]any{"updated_at": time.Now().Unix()}
	if v, ok := mInt(m, "max_usage"); ok && v >= 1 {
		upd["max_usage"] = v
	}
	if v, ok := mInt(m, "expires_at"); ok {
		if v == 0 {
			upd["expires_at"] = nil
		} else {
			upd["expires_at"] = v
		}
	}
	if err := a.S.DB.Model(&code).Updates(upd).Error; err != nil {
		resp.Fail(c, 500, "更新失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) giftCodeDelete(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	res := a.S.DB.Delete(&model.GiftCardCode{}, id)
	if res.RowsAffected == 0 {
		resp.Fail(c, 404, "兑换码不存在")
		return
	}
	resp.OK(c, true)
}

// giftUsages 对应 usages：使用记录分页。
func (a *App) giftUsages(c *gin.Context) {
	cur, size := pageOf(c)
	q := a.S.DB.Model(&model.GiftCardUsage{})
	if v := c.Query("template_id"); v != "" {
		q = q.Where("template_id = ?", v)
	}
	if v := c.Query("user_id"); v != "" {
		q = q.Where("user_id = ?", v)
	}
	// 页面按用户邮箱搜索：filter [{id: "user_email", value}]。
	for _, f := range requestFilters(c) {
		if f.ID != "user_email" {
			continue
		}
		if v, ok := f.Value.(string); ok && strings.TrimSpace(v) != "" {
			q = q.Where("user_id IN (SELECT id FROM v2_user WHERE email LIKE ?)", "%"+escapeLike(strings.TrimSpace(v))+"%")
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.GiftCardUsage
	adminSortExprs(c, giftUsageSort, q).Order("v2_gift_card_usage.created_at DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var t model.GiftCardTemplate
		a.S.DB.Select("name", "type").First(&t, r.TemplateID)
		// 使用记录页显示兑换码、用户邮箱与 IP，这里一并带上。
		var code model.GiftCardCode
		a.S.DB.Select("code").First(&code, r.CodeID)
		var u model.User
		a.S.DB.Select("email").First(&u, r.UserID)
		out = append(out, gin.H{"id": r.ID, "code_id": r.CodeID, "code": code.Code, "template_id": r.TemplateID, "template_name": t.Name,
			"user_id": r.UserID, "user_email": u.Email, "invite_user_id": r.InviteUserID, "rewards_given": r.RewardsGiven,
			"invite_rewards": r.InviteRewards, "multiplier_applied": r.MultiplierApplied, "ip_address": r.IPAddress,
			"created_at": r.CreatedAt})
	}
	paginateJSON(c, total, cur, size, out)
}

// giftTypes 对应 types：可选类型表。
func (a *App) adminGiftTypes(c *gin.Context) {
	resp.OK(c, gin.H{"types": giftTypeNames})
}
