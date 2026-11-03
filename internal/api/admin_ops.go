package api

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 管理端：服务器分组、路由规则、邀请返利、邮件模板、流量重置。
// 对应 V2/Admin/Server/GroupController、RouteController、ReferralController、MailTemplateController、TrafficResetController。

// ---- 服务器分组 ----

func (a *App) groupFetch(c *gin.Context) {
	var groups []model.ServerGroup
	a.S.DB.Order("id DESC").Find(&groups)
	out := make([]gin.H, 0, len(groups))
	for _, g := range groups {
		var servers, users int64
		a.S.DB.Model(&model.Server{}).Where("JSON_CONTAINS(group_ids, ?)", strconv.FormatInt(g.ID, 10)).Count(&servers)
		a.S.DB.Model(&model.User{}).Where("group_id = ?", g.ID).Count(&users)
		out = append(out, gin.H{"id": g.ID, "name": g.Name, "created_at": g.CreatedAt, "updated_at": g.UpdatedAt,
			"server_count": servers, "users_count": users})
	}
	resp.OK(c, out)
}

func (a *App) groupSave(c *gin.Context) {
	m := bodyMap(c)
	name, _ := mStr(m, "name")
	if name == "" {
		resp.Fail(c, 422, "组名不能为空")
		return
	}
	ts := now()
	if id, ok := mInt(m, "id"); ok && id > 0 {
		if err := a.S.DB.Model(&model.ServerGroup{}).Where("id = ?", id).
			Updates(map[string]any{"name": name, "updated_at": ts}).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
	} else if err := a.S.DB.Create(&model.ServerGroup{Name: name, CreatedAt: ts, UpdatedAt: ts}).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) groupDrop(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var g model.ServerGroup
	if err := a.S.DB.First(&g, id).Error; err != nil {
		resp.Fail(c, 400, "组不存在")
		return
	}
	var n int64
	a.S.DB.Model(&model.Server{}).Where("JSON_CONTAINS(group_ids, ?)", strconv.FormatInt(id, 10)).Count(&n)
	if n > 0 {
		resp.Fail(c, 400, "该组已被节点所使用，无法删除")
		return
	}
	a.S.DB.Model(&model.Plan{}).Where("group_id = ?", id).Count(&n)
	if n > 0 {
		resp.Fail(c, 400, "该组已被订阅所使用，无法删除")
		return
	}
	a.S.DB.Model(&model.User{}).Where("group_id = ?", id).Count(&n)
	if n > 0 {
		resp.Fail(c, 400, "该组已被用户所使用，无法删除")
		return
	}
	a.S.DB.Delete(&g)
	resp.OK(c, true)
}

// ---- 路由规则 ----

var routeActions = map[string]bool{"block": true, "direct": true, "dns": true, "proxy": true}

func (a *App) routeFetch(c *gin.Context) {
	var rows []model.ServerRoute
	a.S.DB.Find(&rows)
	c.JSON(200, gin.H{"data": rows})
}

func (a *App) routeSave(c *gin.Context) {
	m := bodyMap(c)
	remarks, _ := mStr(m, "remarks")
	if remarks == "" {
		resp.Fail(c, 422, "备注不能为空")
		return
	}
	matchRaw, ok := m["match"].([]any)
	if !ok || len(matchRaw) == 0 {
		resp.Fail(c, 422, "匹配值不能为空")
		return
	}
	action, _ := mStr(m, "action")
	if action == "" {
		resp.Fail(c, 422, "动作类型不能为空")
		return
	}
	if !routeActions[action] {
		resp.Fail(c, 422, "动作类型参数有误")
		return
	}
	// 与 PHP array_filter 一致：去掉空字符串等假值。
	var match []any
	for _, v := range matchRaw {
		if s := fmt.Sprint(v); s != "" && s != "0" && s != "<nil>" {
			match = append(match, v)
		}
	}
	mb, _ := json.Marshal(match)
	upd := map[string]any{"remarks": remarks, "match": string(mb), "action": action, "updated_at": now()}
	if v, ok := m["action_value"]; ok {
		if s, ok := v.(string); ok {
			upd["action_value"] = s
		} else {
			upd["action_value"] = nil
		}
	}
	if id, ok := mInt(m, "id"); ok && id > 0 {
		if err := a.S.DB.Model(&model.ServerRoute{}).Where("id = ?", id).Updates(upd).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		resp.OK(c, true)
		return
	}
	upd["created_at"] = now()
	if err := a.S.DB.Model(&model.ServerRoute{}).Create(upd).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) routeDrop(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if res := a.S.DB.Delete(&model.ServerRoute{}, id); res.RowsAffected == 0 {
		resp.Fail(c, 400, "路由不存在")
		return
	}
	resp.OK(c, true)
}

// ---- 邀请返利 ----

// referralSummary 对应 ReferralController::summary：按邀请人汇总，最多 200 条。
func (a *App) referralSummary(c *gin.Context) {
	type row struct {
		InviterID    int64
		InviteCount  int64
		TotalSeconds int64
		LastAt       int64
	}
	var rows []row
	a.S.DB.Model(&model.ReferralDurationLog{}).
		Select("inviter_id, COUNT(DISTINCT invitee_id) AS invite_count, SUM(seconds_credited) AS total_seconds, MAX(created_at) AS last_at").
		Group("inviter_id").Order("invite_count DESC").Limit(200).Scan(&rows)
	emails := a.emailsByID(func(add func(int64)) {
		for _, r := range rows {
			add(r.InviterID)
		}
	})
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"inviter_id": r.InviterID, "inviter_email": emailOrNil(emails, r.InviterID),
			"invite_count": r.InviteCount, "total_hours": roundOne(float64(r.TotalSeconds) / 3600), "last_at": r.LastAt,
		})
	}
	resp.OK(c, out)
}

// referralRecords 对应 ReferralController::records：分页明细，page_size 最大 100。
func (a *App) referralRecords(c *gin.Context) {
	q := a.S.DB.Model(&model.ReferralDurationLog{})
	if inv, _ := strconv.ParseInt(c.Query("inviter_id"), 10, 64); inv > 0 {
		q = q.Where("inviter_id = ?", inv)
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size > 100 {
		size = 100
	}
	if size < 1 {
		size = 20
	}
	cur, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	if cur < 1 {
		cur = 1
	}
	var total int64
	q.Count(&total)
	var logs []model.ReferralDurationLog
	q.Order("id DESC").Offset((cur - 1) * size).Limit(size).Find(&logs)
	emails := a.emailsByID(func(add func(int64)) {
		for _, l := range logs {
			add(l.InviterID)
			add(l.InviteeID)
		}
	})
	data := make([]gin.H, 0, len(logs))
	for _, l := range logs {
		data = append(data, gin.H{
			"id": l.ID, "inviter_id": l.InviterID, "inviter_email": emailOrNil(emails, l.InviterID),
			"invitee_id": l.InviteeID, "invitee_email": emailOrNil(emails, l.InviteeID),
			"trade_no": l.TradeNo, "hours_credited": roundOne(float64(l.SecondsCredited) / 3600), "created_at": l.CreatedAt,
		})
	}
	last := int((total + int64(size) - 1) / int64(size))
	if last < 1 {
		last = 1
	}
	resp.OK(c, gin.H{"data": data, "total": total, "current_page": cur, "last_page": last})
}

// emailsByID 收集一组用户 ID 的邮箱映射。
func (a *App) emailsByID(collect func(add func(int64))) map[int64]string {
	ids := map[int64]bool{}
	collect(func(id int64) { ids[id] = true })
	out := map[int64]string{}
	if len(ids) == 0 {
		return out
	}
	list := make([]int64, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	var users []model.User
	a.S.DB.Select("id", "email").Where("id IN ?", list).Find(&users)
	for _, u := range users {
		out[u.ID] = u.Email
	}
	return out
}

func emailOrNil(m map[int64]string, id int64) any {
	if e, ok := m[id]; ok {
		return e
	}
	return nil
}

// ---- 邮件模板 ----

// mailTemplateMeta 对应 MailTemplate::TEMPLATES。
var mailTemplateMeta = map[string]struct {
	Label        string
	RequiredVars []string
	OptionalVars []string
}{
	"verify":        {"邮箱验证码", []string{"code"}, []string{"name", "url"}},
	"notify":        {"站点通知", []string{"content"}, []string{"name", "url"}},
	"remindExpire":  {"到期提醒", []string{}, []string{"name", "url"}},
	"remindTraffic": {"流量提醒", []string{}, []string{"name", "url"}},
	"mailLogin":     {"邮件登录", []string{"link"}, []string{"name", "url"}},
}

func (a *App) mailTemplateList(c *gin.Context) {
	var rows []model.MailTemplate
	a.S.DB.Find(&rows)
	db := map[string]model.MailTemplate{}
	for _, r := range rows {
		db[r.Name] = r
	}
	out := []gin.H{}
	for _, name := range []string{"verify", "notify", "remindExpire", "remindTraffic", "mailLogin"} {
		meta := mailTemplateMeta[name]
		item := gin.H{"name": name, "label": meta.Label, "customized": false, "subject": nil, "updated_at": nil}
		if r, ok := db[name]; ok {
			item["customized"] = true
			item["subject"] = r.Subject
			item["updated_at"] = r.UpdatedAt.Unix()
		}
		out = append(out, item)
	}
	resp.OK(c, out)
}

func (a *App) mailTemplateGet(c *gin.Context) {
	name := c.Query("name")
	meta, ok := mailTemplateMeta[name]
	if !ok {
		resp.Fail(c, 404, "模板不存在")
		return
	}
	var r model.MailTemplate
	customized := a.S.DB.Where("name = ?", name).First(&r).Error == nil
	// 未自定义时给出默认主题与默认正文（Blade 视图转占位符），编辑器才有可改的起点。
	subject, content := r.Subject, r.Content
	if !customized {
		subject = defaultMailSubject(name, a.Set.Get("app_name", "XBoard"))
		content = defaultMailContent(name)
	}
	resp.OK(c, gin.H{
		"name": name, "label": meta.Label, "required_vars": meta.RequiredVars, "optional_vars": meta.OptionalVars,
		"customized": customized, "subject": subject, "content": content,
	})
}

// validateMailContent 对应 MailTemplate::validateContent：必填变量必须出现在内容中。
func validateMailContent(name, content string) []string {
	var errs []string
	for _, v := range mailTemplateMeta[name].RequiredVars {
		if !strings.Contains(content, "{{"+v) {
			errs = append(errs, "缺少必填变量 {{"+v+"}}")
		}
	}
	return errs
}

func (a *App) mailTemplateSave(c *gin.Context) {
	m := bodyMap(c)
	name, _ := mStr(m, "name")
	subject, _ := mStr(m, "subject")
	content, _ := mStr(m, "content")
	if name == "" || subject == "" || content == "" {
		resp.Fail(c, 422, "参数有误")
		return
	}
	if _, ok := mailTemplateMeta[name]; !ok {
		resp.Fail(c, 404, "模板不存在")
		return
	}
	if errs := validateMailContent(name, content); len(errs) > 0 {
		resp.Fail(c, 422, strings.Join(errs, "; "))
		return
	}
	ts := time.Now()
	var r model.MailTemplate
	if a.S.DB.Where("name = ?", name).First(&r).Error == nil {
		a.S.DB.Model(&r).Updates(map[string]any{"subject": subject, "content": content, "updated_at": ts})
	} else {
		a.S.DB.Create(&model.MailTemplate{Name: name, Subject: subject, Content: content, CreatedAt: ts, UpdatedAt: ts})
	}
	resp.OK(c, true)
}

func (a *App) mailTemplateReset(c *gin.Context) {
	name, _ := mStr(bodyMap(c), "name")
	if _, ok := mailTemplateMeta[name]; !ok {
		resp.Fail(c, 404, "模板不存在")
		return
	}
	a.S.DB.Where("name = ?", name).Delete(&model.MailTemplate{})
	resp.OK(c, true)
}

// mailTemplateTest 对应 MailTemplateController::test：使用测试变量发送一封邮件。
func (a *App) mailTemplateTest(c *gin.Context) {
	m := bodyMap(c)
	name, _ := mStr(m, "name")
	if _, ok := mailTemplateMeta[name]; !ok {
		resp.Fail(c, 404, "模板不存在")
		return
	}
	email, _ := mStr(m, "email")
	if email == "" {
		email = cur(c).Email
	}
	app := a.Set.Get("app_name", "Lets VPN")
	vars := map[string]any{"name": app, "url": a.Set.Get("app_url", "https://example.com"), "code": "123456",
		"content": "这是一封测试邮件", "link": "https://example.com/test"}
	subject := app + " - 邮件测试"
	if err := a.jobsEnv().SendTemplateMail(email, subject, name, vars); err != nil {
		resp.Fail(c, 500, "发送失败: "+err.Error())
		return
	}
	resp.OK(c, true)
}

// ---- 流量重置 ----

// trafficFmt 对应 TrafficResetLog::formatTraffic：round(值, 2) 后拼单位，不补多余的 .00（"6 MB"、"1.5 GB"）。
func trafficFmt(b int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	if b < 0 {
		b = 0
	}
	pow := 0
	if b > 0 {
		pow = int(math.Floor(math.Log(float64(b)) / math.Log(1024)))
	}
	if pow > len(units)-1 {
		pow = len(units) - 1
	}
	v := float64(b) / math.Pow(1024, float64(pow))
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64) + " " + units[pow]
}

var resetTypeNames = map[string]string{
	shop.TypeMonthly: "按月重置", shop.TypeFirstDayMon: "每月1号重置", shop.TypeYearly: "按年重置",
	shop.TypeFirstDayYear: "每年1月1日重置", shop.TypeManual: "手动重置", "purchase": "购买重置包",
}

var sourceNames = map[string]string{
	shop.SourceAuto: "自动触发", shop.SourceManual: "手动触发", "api": "API调用", shop.SourceCron: "定时任务",
	shop.SourceUserAccess: "用户访问", shop.SourceOrder: "订单触发",
}

func trafficResetView(l model.TrafficResetLog, email string) gin.H {
	return gin.H{
		"id": l.ID, "user_id": l.UserID, "user_email": email,
		"reset_type": l.ResetType, "reset_type_name": resetTypeNames[l.ResetType], "reset_time": l.ResetTime.UTC().Format("2006-01-02T15:04:05.000000Z"),
		"old_traffic":    gin.H{"upload": l.OldUpload, "download": l.OldDownload, "total": l.OldTotal, "formatted": trafficFmt(l.OldTotal)},
		"new_traffic":    gin.H{"upload": l.NewUpload, "download": l.NewDownload, "total": l.NewTotal, "formatted": trafficFmt(l.NewTotal)},
		"trigger_source": l.TriggerSource, "trigger_source_name": sourceNames[l.TriggerSource],
		"metadata": l.Metadata, "created_at": l.CreatedAt,
	}
}

func (a *App) trafficResetLogs(c *gin.Context) {
	q := a.S.DB.Model(&model.TrafficResetLog{})
	if v, _ := strconv.ParseInt(c.Query("user_id"), 10, 64); v > 0 {
		q = q.Where("user_id = ?", v)
	}
	if e := c.Query("user_email"); e != "" {
		q = q.Where("user_id IN (?)", a.S.DB.Model(&model.User{}).Select("id").Where("email LIKE ?", "%"+e+"%"))
	}
	if v := c.Query("reset_type"); v != "" {
		q = q.Where("reset_type = ?", v)
	}
	if v := c.Query("trigger_source"); v != "" {
		q = q.Where("trigger_source = ?", v)
	}
	if v := c.Query("start_date"); v != "" {
		q = q.Where("reset_time >= ?", v)
	}
	if v := c.Query("end_date"); v != "" {
		q = q.Where("reset_time <= ?", v+" 23:59:59")
	}
	size, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if size < 1 || size > 10000 {
		size = 20
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	var total int64
	q.Count(&total)
	var logs []model.TrafficResetLog
	q.Order("reset_time DESC").Offset((page - 1) * size).Limit(size).Find(&logs)
	emails := a.emailsByID(func(add func(int64)) {
		for _, l := range logs {
			add(l.UserID)
		}
	})
	data := make([]gin.H, 0, len(logs))
	for _, l := range logs {
		e, _ := emailOrNil(emails, l.UserID).(string)
		if e == "" {
			e = "N/A"
		}
		data = append(data, trafficResetView(l, e))
	}
	last := int((total + int64(size) - 1) / int64(size))
	if last < 1 {
		last = 1
	}
	c.JSON(200, gin.H{"data": data, "pagination": gin.H{"current_page": page, "last_page": last, "per_page": size, "total": total}})
}

func (a *App) trafficResetStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days < 1 || days > 365 {
		days = 30
	}
	y, m, d := time.Now().Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, time.Local).AddDate(0, 0, -days)
	count := func(src string) int64 {
		var n int64
		q := a.S.DB.Model(&model.TrafficResetLog{}).Where("reset_time >= ?", start)
		if src != "" {
			q = q.Where("trigger_source = ?", src)
		}
		q.Count(&n)
		return n
	}
	resp.OK(c, gin.H{"total_resets": count(""), "auto_resets": count(shop.SourceAuto),
		"manual_resets": count(shop.SourceManual), "cron_resets": count(shop.SourceCron)})
}

// trafficResetUser 对应 resetUser：仅活跃且有套餐的用户可重置。
func (a *App) trafficResetUser(c *gin.Context) {
	m := bodyMap(c)
	uid, ok := mInt(m, "user_id")
	if !ok || uid == 0 {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	var u model.User
	if err := a.S.DB.First(&u, uid).Error; err != nil {
		resp.Fail(c, 422, "用户不存在")
		return
	}
	s := a.shop()
	var plan *model.Plan
	if u.PlanID != nil {
		var p model.Plan
		if a.S.DB.First(&p, *u.PlanID).Error == nil {
			plan = &p
		}
	}
	if !userIsActive(&u, now()) || plan == nil {
		resp.Fail(c, 400, "该用户当前无法重置流量")
		return
	}
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		return s.PerformReset(tx, &u, shop.SourceManual)
	})
	if err != nil {
		resp.Fail(c, 500, "流量重置失败")
		return
	}
	reason, _ := mStr(m, "reason")
	if reason != "" {
		meta, _ := json.Marshal(map[string]any{"reason": reason, "admin_id": cur(c).ID})
		a.S.DB.Model(&model.TrafficResetLog{}).Where("user_id = ?", u.ID).Order("id DESC").Limit(1).
			Update("metadata", json.RawMessage(meta))
	}
	var fresh model.User
	a.S.DB.First(&fresh, u.ID)
	// PHP 此处返回顶层 message/data，不包 status 外壳。
	c.JSON(200, gin.H{"message": "流量重置成功", "data": gin.H{
		"user_id": u.ID, "email": u.Email, "reset_time": now(), "next_reset_at": fresh.NextResetAt,
	}})
}

// trafficResetHistory 对应 userHistory。
func (a *App) trafficResetHistory(c *gin.Context) {
	uid, _ := strconv.ParseInt(c.Param("uid"), 10, 64)
	var u model.User
	if err := a.S.DB.First(&u, uid).Error; err != nil {
		resp.Fail(c, 404, "用户不存在")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}
	var logs []model.TrafficResetLog
	a.S.DB.Where("user_id = ?", u.ID).Order("reset_time DESC").Limit(limit).Find(&logs)
	hist := make([]gin.H, 0, len(logs))
	for _, l := range logs {
		hist = append(hist, trafficResetView(l, u.Email))
	}
	c.JSON(200, gin.H{"data": gin.H{
		"user":    gin.H{"id": u.ID, "email": u.Email, "reset_count": u.ResetCount, "last_reset_at": u.LastResetAt, "next_reset_at": u.NextResetAt},
		"history": hist,
	}})
}
