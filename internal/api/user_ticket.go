package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"xboard/internal/model"
	"xboard/internal/resp"
)

// 工单状态与回复状态（与 v2_ticket 注释一致）。
const (
	ticketOpen   = 0
	ticketClosed = 1
	replyWaiting = 0
	replyReplied = 1
)

// defaultWithdrawMethods 对应 Dict::WITHDRAW_METHOD_WHITELIST_DEFAULT。
var defaultWithdrawMethods = []string{"支付宝", "USDT", "Paypal"}

// ticketReplyHook 为工单回复后的通知钩子，由邮件/Telegram 模块（第 5 步）注入；默认为空实现。
var ticketReplyHook func(t *model.Ticket, msg *model.TicketMessage)

func (a *App) registerTicketRoutes(u *gin.RouterGroup) {
	u.GET("/ticket/fetch", a.ticketFetch)
	u.POST("/ticket/save", a.ticketSave)
	u.POST("/ticket/reply", a.ticketReply)
	u.POST("/ticket/close", a.ticketClose)
	u.POST("/ticket/withdraw", a.ticketWithdraw)
}

// ticketMessageView 对应 MessageResource，额外标注 is_me。
type ticketMessageView struct {
	ID        int64  `json:"id"`
	TicketID  int64  `json:"ticket_id"`
	UserID    int64  `json:"user_id"`
	Message   string `json:"message"`
	IsMe      bool   `json:"is_me"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

func (a *App) ticketFetch(c *gin.Context) {
	u := cur(c)
	if id := c.Query("id"); id != "" {
		var t model.Ticket
		if err := a.S.DB.Where("id = ? AND user_id = ?", id, u.ID).First(&t).Error; err != nil {
			resp.Fail(c, 400, "Ticket does not exist")
			return
		}
		var msgs []model.TicketMessage
		a.S.DB.Where("ticket_id = ?", t.ID).Order("id ASC").Find(&msgs)
		views := make([]ticketMessageView, 0, len(msgs))
		for _, m := range msgs {
			views = append(views, ticketMessageView{ID: m.ID, TicketID: m.TicketID, UserID: m.UserID,
				Message: m.Message, IsMe: m.UserID == t.UserID, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt})
		}
		resp.OK(c, gin.H{
			"id": t.ID, "level": t.Level, "reply_status": t.ReplyStatus, "status": t.Status,
			"subject": t.Subject, "user_id": t.UserID, "message": views,
			"created_at": t.CreatedAt, "updated_at": t.UpdatedAt,
		})
		return
	}
	var ts []model.Ticket
	a.S.DB.Where("user_id = ?", u.ID).Order("created_at DESC").Find(&ts)
	out := make([]gin.H, 0, len(ts))
	for _, t := range ts {
		out = append(out, gin.H{"id": t.ID, "level": t.Level, "reply_status": t.ReplyStatus, "status": t.Status,
			"subject": t.Subject, "user_id": t.UserID, "message": nil, "created_at": t.CreatedAt, "updated_at": t.UpdatedAt})
	}
	resp.OK(c, out)
}

func (a *App) ticketSave(c *gin.Context) {
	u := cur(c)
	var in struct {
		Subject string `json:"subject" form:"subject" binding:"required"`
		Level   *int   `json:"level" form:"level" binding:"required"`
		Message string `json:"message" form:"message" binding:"required"`
	}
	if err := c.ShouldBind(&in); err != nil || in.Level == nil || *in.Level < 0 || *in.Level > 2 {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	t, err := a.createTicket(u.ID, in.Subject, *in.Level, in.Message)
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	if ticketHook != nil {
		ticketHook(t, &model.TicketMessage{UserID: u.ID, TicketID: t.ID, Message: in.Message})
	}
	resp.OK(c, true)
}

// createTicket 对应 TicketService::createTicket：存在未关闭工单时拒绝新建。
func (a *App) createTicket(uid int64, subject string, level int, message string) (*model.Ticket, error) {
	var t *model.Ticket
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		var n int64
		tx.Model(&model.Ticket{}).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("status = ? AND user_id = ?", ticketOpen, uid).Count(&n)
		if n > 0 {
			return errors.New("存在未关闭的工单")
		}
		ts := now()
		tk := model.Ticket{UserID: uid, Subject: subject, Level: level, ReplyStatus: replyWaiting,
			Status: ticketOpen, CreatedAt: ts, UpdatedAt: ts}
		if err := tx.Create(&tk).Error; err != nil {
			return errors.New("工单创建失败")
		}
		if err := tx.Create(&model.TicketMessage{UserID: uid, TicketID: tk.ID, Message: message,
			CreatedAt: ts, UpdatedAt: ts}).Error; err != nil {
			return errors.New("工单消息创建失败")
		}
		t = &tk
		return nil
	})
	return t, err
}

// replyTicket 对应 TicketService::reply：用户回复置为待回复，管理员回复置为已回复。
func (a *App) replyTicket(t *model.Ticket, text string, uid int64) (*model.TicketMessage, error) {
	ts := now()
	msg := model.TicketMessage{UserID: uid, TicketID: t.ID, Message: text, CreatedAt: ts, UpdatedAt: ts}
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}
		if uid != t.UserID {
			t.ReplyStatus = replyReplied
		} else {
			t.ReplyStatus = replyWaiting
		}
		t.LastReplyUserID = uid
		t.UpdatedAt = ts
		return tx.Model(t).Updates(map[string]any{
			"reply_status": t.ReplyStatus, "last_reply_user_id": uid, "updated_at": ts,
		}).Error
	})
	return &msg, err
}

func (a *App) ticketReply(c *gin.Context) {
	u := cur(c)
	var in struct {
		ID      int64  `json:"id" form:"id"`
		Message string `json:"message" form:"message"`
	}
	_ = c.ShouldBind(&in)
	if in.ID == 0 {
		resp.Fail(c, 400, "Invalid parameter")
		return
	}
	if in.Message == "" {
		resp.Fail(c, 400, "Message cannot be empty")
		return
	}
	var t model.Ticket
	if err := a.S.DB.Where("id = ? AND user_id = ?", in.ID, u.ID).First(&t).Error; err != nil {
		resp.Fail(c, 400, "Ticket does not exist")
		return
	}
	if t.Status != ticketOpen {
		resp.Fail(c, 400, "The ticket is closed and cannot be replied")
		return
	}
	if a.Set.Bool("ticket_must_wait_reply", false) {
		var last model.TicketMessage
		if a.S.DB.Where("ticket_id = ?", t.ID).Order("id DESC").First(&last).Error == nil && last.UserID == u.ID {
			resp.Fail(c, 400, "Please wait for the technical enginneer to reply")
			return
		}
	}
	msg, err := a.replyTicket(&t, in.Message, u.ID)
	if err != nil {
		resp.Fail(c, 400, "Ticket reply failed")
		return
	}
	if ticketReplyHook != nil {
		ticketReplyHook(&t, msg)
	}
	if ticketHook != nil {
		ticketHook(&t, msg)
	}
	resp.OK(c, true)
}

func (a *App) ticketClose(c *gin.Context) {
	u := cur(c)
	id, _ := strconv.ParseInt(c.Query("id"), 10, 64)
	if id == 0 {
		var in struct {
			ID int64 `json:"id" form:"id"`
		}
		_ = c.ShouldBind(&in)
		id = in.ID
	}
	if id == 0 {
		resp.Fail(c, 422, "Invalid parameter")
		return
	}
	res := a.S.DB.Model(&model.Ticket{}).Where("id = ? AND user_id = ?", id, u.ID).
		Updates(map[string]any{"status": ticketClosed, "updated_at": now()})
	if res.Error != nil {
		resp.Fail(c, 500, "Close failed")
		return
	}
	if res.RowsAffected == 0 {
		resp.Fail(c, 400, "Ticket does not exist")
		return
	}
	resp.OK(c, true)
}

// withdrawMethods 解析 commission_withdraw_method：支持 JSON 数组或逗号分隔，缺省用内置白名单。
func (a *App) withdrawMethods() []string {
	raw := a.Set.Get("commission_withdraw_method", "")
	if raw == "" {
		return defaultWithdrawMethods
	}
	var list []string
	if json.Unmarshal([]byte(raw), &list) == nil && len(list) > 0 {
		return list
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return defaultWithdrawMethods
	}
	return out
}

func (a *App) ticketWithdraw(c *gin.Context) {
	u := cur(c)
	var in struct {
		Method  string `json:"withdraw_method" form:"withdraw_method" binding:"required"`
		Account string `json:"withdraw_account" form:"withdraw_account" binding:"required"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	if a.Set.Bool("withdraw_close_enable", false) {
		resp.Fail(c, 400, "Unsupported withdraw")
		return
	}
	ok := false
	for _, m := range a.withdrawMethods() {
		if m == in.Method {
			ok = true
			break
		}
	}
	if !ok {
		resp.Fail(c, 422, "Unsupported withdrawal method")
		return
	}
	limit := a.Set.Int("commission_withdraw_limit", 100)
	if int64(limit) > u.CommissionBal/100 {
		resp.Fail(c, 422, fmt.Sprintf("The current required minimum withdrawal commission is %d", limit))
		return
	}
	subject := "[Commission Withdrawal Request] This ticket is opened by the system"
	message := "Withdrawal method：" + in.Method + "\r\nWithdrawal account：" + in.Account
	if _, err := a.createTicket(u.ID, subject, 2, message); err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	resp.OK(c, true)
}
