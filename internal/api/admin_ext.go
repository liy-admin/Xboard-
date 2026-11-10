package api

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 管理端：工单、订单、知识库。对应 V2/Admin 下的 TicketController、OrderController、KnowledgeController。

// ---- 工单 ----

var adminTicketFilter = map[string]bool{"id": true, "user_id": true, "subject": true, "level": true, "status": true,
	"reply_status": true, "created_at": true, "updated_at": true}

// ticketUserView 对应 UserController::transformUserData 的常用字段（金额已换算为元）。
func (a *App) ticketUserView(u *model.User) gin.H {
	if u == nil {
		return nil
	}
	return gin.H{
		"id": u.ID, "email": u.Email, "balance": float64(u.Balance) / 100,
		"commission_balance": float64(u.CommissionBal) / 100, "transfer_enable": u.TransferEnab,
		"u": u.U, "d": u.D, "expired_at": u.ExpiredAt, "plan_id": u.PlanID, "banned": u.Banned,
		"is_admin": u.IsAdmin, "is_staff": u.IsStaff, "invite_user_id": u.InviteUserID,
		"telegram_id": u.TelegramID, "device_id": u.DeviceID,
		"subscribe_url": a.subscribeURL(u.Token, ""), "created_at": u.CreatedAt, "updated_at": u.UpdatedAt,
	}
}

func (a *App) adminTicketFetch(c *gin.Context) {
	if id := c.Query("id"); id != "" {
		var t model.Ticket
		if err := a.S.DB.Where("id = ?", id).First(&t).Error; err != nil {
			resp.Fail(c, 400, "工单不存在")
			return
		}
		var rawMsgs []model.TicketMessage
		a.S.DB.Where("ticket_id = ?", t.ID).Order("id ASC").Find(&rawMsgs)
		// 对应 TicketMessage 的 is_from_user / is_from_admin 访问器：发送者是工单发起人即为用户，否则为管理员。
		// 前端据此决定消息气泡的左右与配色。
		type msgView struct {
			model.TicketMessage
			IsFromUser  bool `json:"is_from_user"`
			IsFromAdmin bool `json:"is_from_admin"`
		}
		msgs := make([]msgView, 0, len(rawMsgs))
		for _, m := range rawMsgs {
			msgs = append(msgs, msgView{TicketMessage: m, IsFromUser: m.UserID == t.UserID, IsFromAdmin: m.UserID != t.UserID})
		}
		var u model.User
		a.S.DB.First(&u, t.UserID)
		resp.OK(c, gin.H{
			"id": t.ID, "user_id": t.UserID, "subject": t.Subject, "level": t.Level,
			"status": t.Status, "reply_status": t.ReplyStatus, "created_at": t.CreatedAt, "updated_at": t.UpdatedAt,
			"messages": msgs, "user": a.ticketUserView(&u),
		})
		return
	}
	cur, size := pageOf(c)
	q := a.S.DB.Model(&model.Ticket{})
	if v := c.Query("status"); v != "" {
		q = q.Where("status = ?", v)
	}
	if e := c.Query("email"); e != "" {
		q = q.Where("user_id IN (?)", a.S.DB.Model(&model.User{}).Select("id").Where("email = ?", e))
	}
	q = adminFilters(c, adminTicketFilter, q)
	var total int64
	q.Count(&total)
	var rows []model.Ticket
	adminSorts(c, adminTicketFilter, q).Order("updated_at DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, t := range rows {
		var u model.User
		a.S.DB.First(&u, t.UserID)
		out = append(out, gin.H{
			"id": t.ID, "user_id": t.UserID, "subject": t.Subject, "level": t.Level, "status": t.Status,
			"reply_status": t.ReplyStatus, "created_at": t.CreatedAt, "updated_at": t.UpdatedAt,
			"user": a.ticketUserView(&u),
		})
	}
	c.JSON(200, gin.H{"data": out, "total": total})
}

// adminTicketReply 对应 TicketService::replyByAdmin：写入回复，并按 30 分钟节流发送邮件通知。
func (a *App) adminTicketReply(c *gin.Context) {
	m := bodyMap(c)
	id, _ := mInt(m, "id")
	text, _ := mStr(m, "message")
	if id == 0 {
		resp.Fail(c, 422, "工单ID不能为空")
		return
	}
	if text == "" {
		resp.Fail(c, 422, "消息不能为空")
		return
	}
	var t model.Ticket
	if err := a.S.DB.First(&t, id).Error; err != nil {
		resp.Fail(c, 400, "工单不存在")
		return
	}
	msg, err := a.replyTicket(&t, text, cur(c).ID)
	if err != nil {
		resp.Fail(c, 400, "工单回复失败")
		return
	}
	a.notifyTicketReply(&t, msg)
	resp.OK(c, true)
}

// notifyTicketReply 对应 TicketService::sendEmailNotify。
func (a *App) notifyTicketReply(t *model.Ticket, msg *model.TicketMessage) {
	var u model.User
	if a.S.DB.First(&u, t.UserID).Error != nil {
		return
	}
	key := fmt.Sprintf("ticket_sendEmailNotify_%d", t.UserID)
	ok, err := a.S.Redis.SetNX(bgCtx(), key, 1, 1800*time.Second).Result()
	if err != nil || !ok {
		return
	}
	app := a.Set.Get("app_name", "Lets VPN")
	subject := "您在" + app + "的工单得到了回复"
	a.mailQ().Enqueue(jobs.MailTask{To: u.Email, Subject: subject, Template: "notify", Values: map[string]any{
		"name": app, "url": a.Set.Get("app_url", ""),
		"content": fmt.Sprintf("主题：%s\r\n回复内容：%s", t.Subject, msg.Message),
	}})
}

// jobsEnv 为 API 层提供与定时任务相同的邮件发送能力。
func (a *App) jobsEnv() *jobs.Env {
	return &jobs.Env{DB: a.S.DB, Redis: jobs.NewRedisAdapter(a.S.Redis), Set: a.Set, Now: time.Now,
		Push: jobs.NewRedisPusher(a.S.Redis)}
}

func (a *App) adminTicketClose(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if id == 0 {
		resp.Fail(c, 422, "工单ID不能为空")
		return
	}
	res := a.S.DB.Model(&model.Ticket{}).Where("id = ?", id).Updates(map[string]any{"status": ticketClosed, "updated_at": now()})
	if res.Error != nil {
		resp.Fail(c, 500, "关闭失败")
		return
	}
	if res.RowsAffected == 0 {
		resp.Fail(c, 400, "工单不存在")
		return
	}
	resp.OK(c, true)
}

// ---- 订单 ----

var adminOrderFilter = map[string]bool{"id": true, "invite_user_id": true, "user_id": true, "plan_id": true,
	"coupon_id": true, "payment_id": true, "type": true, "period": true, "trade_no": true, "total_amount": true,
	"handling_amount": true, "discount_amount": true, "surplus_amount": true, "balance_amount": true,
	"status": true, "commission_status": true, "commission_balance": true, "paid_at": true,
	"created_at": true, "updated_at": true}

// orderView 对应 OrderController::fetch 的输出：订单字段 + plan.name + 旧版周期名。
func (a *App) orderView(o model.Order) gin.H {
	var plan model.Plan
	var planName any
	if a.S.DB.Select("id", "name").First(&plan, o.PlanID).Error == nil {
		planName = gin.H{"id": plan.ID, "name": plan.Name}
	}
	var surplusIDs any
	if len(o.SurplusOrderIDs) > 0 && string(o.SurplusOrderIDs) != "null" {
		surplusIDs = o.SurplusOrderIDs
	}
	return gin.H{
		"id": o.ID, "invite_user_id": o.InviteUserID, "user_id": o.UserID, "plan_id": o.PlanID,
		"coupon_id": o.CouponID, "payment_id": o.PaymentID, "type": o.Type,
		"period": shop.LegacyPeriodName(o.Period), "trade_no": o.TradeNo, "total_amount": o.TotalAmount,
		"handling_amount": o.HandlingAmount, "discount_amount": o.DiscountAmount, "surplus_amount": o.SurplusAmount,
		"surplus_credit": o.SurplusCredit, "surplus_order_ids": surplusIDs,
		"balance_amount": o.BalanceAmount, "status": o.Status, "commission_status": o.CommissionStatus,
		"commission_balance": o.CommissionBalance, "actual_commission_balance": o.ActualCommissionBalance,
		"referral_duration_seconds": o.ReferralDurationSeconds, "callback_no": o.CallbackNo, "paid_at": o.PaidAt,
		"created_at": o.CreatedAt, "updated_at": o.UpdatedAt, "plan": planName,
	}
}

// adminOrderSort 订单列表可排序的列（列键 → SQL 表达式）：普通列沿用过滤白名单，另加关联与计算列。
var adminOrderSort = func() map[string]string {
	m := map[string]string{}
	for k := range adminOrderFilter {
		m[k] = "v2_order." + k
	}
	m["plan"] = "(SELECT p.name FROM v2_plan p WHERE p.id = v2_order.plan_id)"
	m["payment"] = "(SELECT pm.name FROM v2_payment pm WHERE pm.id = v2_order.payment_id)"
	m["payer"] = "(SELECT u.email FROM v2_user u WHERE u.id = v2_order.user_id)"
	// 周期按时长排，新旧两种周期键都认；流量包排最后。
	m["period"] = "CASE v2_order.period" +
		" WHEN 'weekly' THEN 7 WHEN 'week_price' THEN 7 WHEN 'monthly' THEN 30 WHEN 'month_price' THEN 30" +
		" WHEN 'quarterly' THEN 90 WHEN 'quarter_price' THEN 90 WHEN 'half_yearly' THEN 180 WHEN 'half_year_price' THEN 180" +
		" WHEN 'yearly' THEN 365 WHEN 'year_price' THEN 365 WHEN 'two_yearly' THEN 730 WHEN 'two_year_price' THEN 730" +
		" WHEN 'three_yearly' THEN 1095 WHEN 'three_year_price' THEN 1095 WHEN 'onetime' THEN 100000 WHEN 'onetime_price' THEN 100000" +
		" ELSE 200000 END"
	return m
}()

func (a *App) adminOrderFetch(c *gin.Context) {
	cur, size := pageOf(c)
	q := a.S.DB.Model(&model.Order{})
	if c.Query("is_commission") == "1" || c.Query("is_commission") == "true" {
		q = q.Where("invite_user_id IS NOT NULL AND status NOT IN (0, 2) AND commission_balance > 0")
	}
	q = adminFilters(c, adminOrderFilter, q)
	var total int64
	q.Count(&total)
	var rows []model.Order
	adminSortExprs(c, adminOrderSort, q).Order("v2_order.created_at DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	users, payments := a.orderListRefs(rows)
	out := make([]gin.H, 0, len(rows))
	for _, o := range rows {
		v := a.orderView(o)
		// 列表直接展示付款人与支付方式，用户被删时 user 为 null（前端仍按 user_id 跳转）。
		v["user"] = users[o.UserID]
		var pay any
		if o.PaymentID != nil {
			pay = payments[*o.PaymentID]
		}
		v["payment"] = pay
		out = append(out, v)
	}
	paginateJSON(c, total, cur, size, out)
}

// orderListRefs 按页批量取订单的付款人与支付方式（两次查询，而不是每行各查一次）。
func (a *App) orderListRefs(rows []model.Order) (map[int64]any, map[int64]any) {
	users := map[int64]any{}
	payments := map[int64]any{}
	var uids, pids []int64
	seenU, seenP := map[int64]bool{}, map[int64]bool{}
	for _, o := range rows {
		if !seenU[o.UserID] {
			seenU[o.UserID] = true
			uids = append(uids, o.UserID)
		}
		if o.PaymentID != nil && !seenP[*o.PaymentID] {
			seenP[*o.PaymentID] = true
			pids = append(pids, *o.PaymentID)
		}
	}
	if len(uids) > 0 {
		var us []model.User
		a.S.DB.Select("id", "email").Where("id IN ?", uids).Find(&us)
		for _, u := range us {
			users[u.ID] = gin.H{"id": u.ID, "email": u.Email}
		}
	}
	if len(pids) > 0 {
		var ps []model.Payment
		a.S.DB.Select("id", "name", "payment").Where("id IN ?", pids).Find(&ps)
		for _, p := range ps {
			payments[p.ID] = gin.H{"id": p.ID, "name": p.Name, "payment": p.Payment}
		}
	}
	return users, payments
}

// adminOrderDetail 对应 OrderController::detail：订单 + 用户 + 完整套餐 + 佣金流水 + 邀请人。
// 详情弹窗读取 detail.user.email、detail.plan.name、detail.invite_user、detail.commission_log。
func (a *App) adminOrderDetail(c *gin.Context) {
	id, _ := mInt(gin.H{"id": c.Query("id")}, "id")
	var o model.Order
	if err := a.S.DB.First(&o, id).Error; err != nil {
		resp.Fail(c, 400, "订单不存在")
		return
	}
	out := a.orderView(o)
	var plan model.Plan
	if a.S.DB.First(&plan, o.PlanID).Error == nil {
		out["plan"] = plan
	}
	var u model.User
	if a.S.DB.First(&u, o.UserID).Error == nil {
		out["user"] = a.userAdminView(&u)
	}
	var inviter any
	if o.InviteUserID != nil {
		var iu model.User
		if a.S.DB.First(&iu, *o.InviteUserID).Error == nil {
			inviter = a.userAdminView(&iu)
		}
	}
	out["invite_user"] = inviter
	logs := []model.CommissionLog{}
	a.S.DB.Where("trade_no = ?", o.TradeNo).Order("id ASC").Find(&logs)
	out["commission_log"] = logs
	resp.OK(c, out)
}

// adminOrderPaid 对应 OrderController::paid：仅待支付订单可手动确认。
func (a *App) adminOrderPaid(c *gin.Context) {
	tn, _ := mStr(bodyMap(c), "trade_no")
	var o model.Order
	if err := a.S.DB.Where("trade_no = ?", tn).First(&o).Error; err != nil {
		resp.Fail(c, 400, "订单不存在")
		return
	}
	if o.Status != shop.StatusPending {
		resp.Fail(c, 400, "只能对待支付的订单进行操作")
		return
	}
	if err := a.shop().Paid(o.TradeNo, "manual_operation"); err != nil {
		resp.Fail(c, 500, "更新失败")
		return
	}
	resp.OK(c, true)
}

// adminOrderCancel 对应 OrderController::cancel。
func (a *App) adminOrderCancel(c *gin.Context) {
	tn, _ := mStr(bodyMap(c), "trade_no")
	var o model.Order
	if err := a.S.DB.Where("trade_no = ?", tn).First(&o).Error; err != nil {
		resp.Fail(c, 400, "订单不存在")
		return
	}
	if o.Status != shop.StatusPending {
		resp.Fail(c, 400, "只能对待支付的订单进行操作")
		return
	}
	if err := a.shop().Cancel(&o); err != nil {
		resp.Fail(c, 400, "更新失败")
		return
	}
	resp.OK(c, true)
}

// ---- 知识库 ----

func (a *App) adminKnowledgeFetch(c *gin.Context) {
	if id, ok := mInt(gin.H{"id": c.Query("id")}, "id"); ok && id > 0 {
		var k model.Knowledge
		if err := a.S.DB.First(&k, id).Error; err != nil {
			resp.Fail(c, 400, "知识不存在")
			return
		}
		resp.OK(c, k)
		return
	}
	var rows []model.Knowledge
	a.S.DB.Select("id", "title", "updated_at", "category", "`show`").Order("sort ASC").Find(&rows)
	resp.OK(c, rows)
}

func (a *App) adminKnowledgeCategories(c *gin.Context) {
	var cats []string
	a.S.DB.Model(&model.Knowledge{}).Distinct().Pluck("category", &cats)
	resp.OK(c, cats)
}

func (a *App) adminKnowledgeSave(c *gin.Context) {
	m := bodyMap(c)
	upd := map[string]any{}
	for _, k := range []string{"language", "category", "title", "body"} {
		if v, ok := mStr(m, k); ok {
			upd[k] = v
		}
	}
	if v, ok := mInt(m, "sort"); ok {
		upd["sort"] = v
	}
	if v, ok := mBool(m, "show"); ok {
		upd["show"] = v
	}
	if id, ok := mInt(m, "id"); ok && id > 0 {
		upd["updated_at"] = now()
		if err := a.S.DB.Model(&model.Knowledge{}).Where("id = ?", id).Updates(upd).Error; err != nil {
			resp.Fail(c, 500, "创建失败")
			return
		}
		resp.OK(c, true)
		return
	}
	upd["created_at"], upd["updated_at"] = now(), now()
	if err := a.S.DB.Model(&model.Knowledge{}).Create(upd).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) adminKnowledgeShow(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var k model.Knowledge
	if err := a.S.DB.First(&k, id).Error; err != nil {
		resp.Fail(c, 400, "知识不存在")
		return
	}
	if err := a.S.DB.Model(&k).Update("show", !k.Show).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) adminKnowledgeDrop(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if res := a.S.DB.Delete(&model.Knowledge{}, id); res.RowsAffected == 0 {
		resp.Fail(c, 400, "知识不存在")
		return
	}
	resp.OK(c, true)
}

func (a *App) adminKnowledgeSort(c *gin.Context) {
	ids, _ := bodyMap(c)["ids"].([]any)
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		for i, v := range ids {
			id, ok := toNum(v)
			if !ok {
				return errors.New("bad id")
			}
			if err := tx.Model(&model.Knowledge{}).Where("id = ?", int64(id)).Update("sort", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}
