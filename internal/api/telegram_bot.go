package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/notify"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// Telegram 机器人：webhook 入口与内置命令（对应 TelegramController 与 plugins-core/Telegram）。
// 插件未启用时命令与通知都不生效；文案优先取插件配置，缺省为内置默认值（与 config.json 一致）。

// telegramCommand 为一条解析后的消息。
type telegramCommand struct {
	command   string
	args      []string
	chatID    int64
	text      string
	replyText string
	isPrivate bool
	isReply   bool
}

var ticketReplyRe = regexp.MustCompile(`(📮.*?工单提醒.*?#?|工单ID: ?)(\d+)`)

func escapeMD(s string) string {
	return strings.NewReplacer("_", `\_`, "*", `\*`, "`", "\\`", "[", `\[`).Replace(s)
}

func (a *App) tgClient() *notify.TelegramClient {
	return notify.NewTelegram(a.Set.Get("telegram_bot_token", ""))
}

// telegramText 读取插件文案：配置优先，其次内置默认值；字面量 \n 还原为换行（PHP 对欢迎语做同样的替换）。
func (a *App) telegramText(key, def string) string {
	return strings.ReplaceAll(a.pluginConfigText("telegram", key, def), `\n`, "\n")
}

// telegramWebhook 对应 TelegramController::webhook：access_token 为 md5(bot_token)。
func (a *App) telegramWebhook(c *gin.Context) {
	token := a.Set.Get("telegram_bot_token", "")
	sum := md5.Sum([]byte(token))
	access, _ := requestParams(c)["access_token"].(string)
	if token == "" || access != hex.EncodeToString(sum[:]) {
		resp.Fail(c, 401, "access_token is error")
		return
	}
	body, _ := io.ReadAll(c.Request.Body)
	var data map[string]any
	_ = json.Unmarshal(body, &data)
	a.handleJoinRequest(data)
	if msg, ok := a.formatTelegramMessage(data); ok {
		a.handleTelegramMessage(msg)
	}
	c.Status(200)
}

// formatTelegramMessage 对应 formatMessage：文本消息拆分为命令与参数，回复消息带上被回复的文本。
func (a *App) formatTelegramMessage(data map[string]any) (telegramCommand, bool) {
	m, ok := data["message"].(map[string]any)
	if !ok {
		return telegramCommand{}, false
	}
	text, ok := m["text"].(string)
	if !ok {
		return telegramCommand{}, false
	}
	parts := strings.Split(text, " ")
	chat, _ := m["chat"].(map[string]any)
	chatID, _ := anyInt(chat["id"])
	cmd := telegramCommand{command: parts[0], args: parts[1:], chatID: chatID, text: text,
		isPrivate: fmt.Sprint(chat["type"]) == "private"}
	if reply, ok := m["reply_to_message"].(map[string]any); ok {
		if rt, ok := reply["text"].(string); ok {
			cmd.isReply = true
			cmd.replyText = rt
		}
	}
	return cmd, true
}

// botNameOf 返回机器人用户名（用于去掉 /cmd@botname 中的后缀）。
func (a *App) botNameOf() string {
	raw, err := a.tgClient().GetMe()
	if err != nil {
		return ""
	}
	var me struct {
		Username string `json:"username"`
	}
	_ = json.Unmarshal(raw, &me)
	return me.Username
}

func (a *App) handleTelegramMessage(msg telegramCommand) {
	if !a.pluginEnabled("telegram") {
		return // 插件未启用时 PHP 不会加载命令处理
	}
	if i := strings.Index(msg.command, "@"); i > 0 {
		if strings.EqualFold(msg.command[i+1:], a.botNameOf()) {
			msg.command = msg.command[:i]
		}
	}
	tg := a.tgClient()
	send := func(text string) { _ = tg.SendMessage(msg.chatID, text, "markdown") }
	if msg.isReply {
		if m := ticketReplyRe.FindStringSubmatch(msg.replyText); m != nil {
			a.telegramTicketReply(msg, m[2], send)
		}
		return
	}
	switch msg.command {
	case "/start":
		a.telegramStart(msg, send)
	case "/bind":
		a.telegramBind(msg, send)
	case "/traffic":
		a.telegramTraffic(msg, send)
	case "/getlatesturl":
		a.telegramLatestURL(msg, send)
	case "/unbind":
		a.telegramUnbind(msg, send)
	default:
		if msg.isPrivate {
			send(a.telegramText("help_text", "请使用以下命令：\\n/bind - 绑定账号\\n/traffic - 查看流量\\n/getlatesturl - 获取最新链接"))
		}
	}
}

// telegramBoundUser 找到绑定该聊天的账号；非私聊或未绑定时发送提示并返回 nil。
func (a *App) telegramBoundUser(msg telegramCommand, send func(string)) *model.User {
	if !msg.isPrivate {
		send("请在私聊中使用此命令")
		return nil
	}
	var u model.User
	if a.S.DB.Where("telegram_id = ?", msg.chatID).First(&u).Error != nil {
		send("请先绑定账号")
		return nil
	}
	return &u
}

// telegramStart 对应 handleStartCommand：欢迎语由标题、介绍、绑定状态提示与底部提示组成。
func (a *App) telegramStart(msg telegramCommand, send func(string)) {
	var u model.User
	bound := a.S.DB.Where("telegram_id = ?", msg.chatID).First(&u).Error == nil
	text := a.telegramText("start_welcome_title", "🎉 欢迎使用 XBoard Telegram Bot！") + "\n\n" +
		a.telegramText("start_bot_description", "🤖 我是您的专属助手，可以帮助您：\\n• 绑定您的 XBoard 账号\\n• 查看流量使用情况\\n• 获取最新订阅链接\\n• 管理账号绑定状态") + "\n\n"
	if bound {
		text += "✅ 您已绑定账号：" + u.Email + "\n\n"
		text += a.telegramText("start_unbind_guide", "📋 可用命令：\\n/traffic - 查看流量使用情况\\n/getlatesturl - 获取订阅链接\\n/unbind - 解绑账号")
	} else {
		text += a.telegramText("start_bind_guide", "🔗 请先绑定您的 XBoard 账号：\\n1. 登录您的 XBoard 账户\\n2. 复制您的订阅链接\\n3. 发送 /bind + 订阅链接")
		text += "\n\n" // PHP 此处直接拼接，导致两段连在同一行；这里补充分隔（见 PROGRESS.md 差异）
		text += a.telegramText("start_bind_commands", "📋 可用命令：\\n/bind [订阅链接] - 绑定账号")
	}
	text += "\n\n" + a.telegramText("start_footer", "💡 提示：所有命令都需要在私聊中使用")
	send(text)
}

// extractTokenFromURL 对应 extractTokenFromUrl：优先取 token 查询参数，否则取路径最后一段。
func extractTokenFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if t := u.Query().Get("token"); t != "" {
		return t
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return parts[len(parts)-1]
}

func (a *App) telegramBind(msg telegramCommand, send func(string)) {
	if !msg.isPrivate {
		send("请在私聊中使用此命令")
		return
	}
	if len(msg.args) == 0 {
		send("参数有误，请携带订阅地址发送")
		return
	}
	token := extractTokenFromURL(msg.args[0])
	if token == "" {
		send("订阅地址无效")
		return
	}
	var u model.User
	if a.S.DB.Where("token = ?", token).First(&u).Error != nil {
		send("用户不存在")
		return
	}
	if u.TelegramID != nil {
		send("该账号已经绑定了Telegram账号")
		return
	}
	if err := a.S.DB.Model(&u).Update("telegram_id", msg.chatID).Error; err != nil {
		send("设置失败")
		return
	}
	send("绑定成功")
}

func (a *App) telegramTraffic(msg telegramCommand, send func(string)) {
	u := a.telegramBoundUser(msg, send)
	if u == nil {
		return
	}
	used := u.U + u.D
	total := u.TransferEnab
	pct := 0.0
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}
	if total == 0 { // 流量上限为 0 视为无限流量
		send(fmt.Sprintf("📊 流量使用情况\n\n已用流量：%sG\n总流量：无限\n剩余流量：无限", gbText(used)))
		return
	}
	send(fmt.Sprintf("📊 流量使用情况\n\n已用流量：%sG\n总流量：%sG\n剩余流量：%sG\n使用率：%.2f%%",
		gbText(used), gbText(total), gbText(total-used), pct))
}

// gbText 对应 number_format(transferToGB, 2, '.', ”)。
func gbText(b int64) string {
	return strconv.FormatFloat(float64(b)/1073741824, 'f', 2, 64)
}

func (a *App) telegramLatestURL(msg telegramCommand, send func(string)) {
	u := a.telegramBoundUser(msg, send)
	if u == nil {
		return
	}
	send(fmt.Sprintf("🔗 您的订阅链接：\n\n%s", a.subscribeURL(u.Token, "")))
}

func (a *App) telegramUnbind(msg telegramCommand, send func(string)) {
	u := a.telegramBoundUser(msg, send)
	if u == nil {
		return
	}
	if err := a.S.DB.Model(u).Update("telegram_id", nil).Error; err != nil {
		send("解绑失败")
		return
	}
	send("解绑成功")
}

// telegramTicketReply 对应 handleTicketReply：绑定账号回复工单提醒即可在工单下回复。
func (a *App) telegramTicketReply(msg telegramCommand, idRaw string, send func(string)) {
	u := a.telegramBoundUser(msg, send)
	if u == nil {
		return
	}
	id, _ := strconv.ParseInt(idRaw, 10, 64)
	var t model.Ticket
	if a.S.DB.First(&t, id).Error != nil {
		send("工单不存在")
		return
	}
	msgRow, err := a.replyTicket(&t, msg.text, u.ID)
	if err != nil {
		send("工单回复失败")
		return
	}
	a.notifyTicketReply(&t, msgRow)
	send(fmt.Sprintf("工单 #%d 回复成功", id))
}

// handleJoinRequest 对应 formatChatJoinRequest：仅对绑定且可用的用户通过入群申请。
func (a *App) handleJoinRequest(data map[string]any) {
	jr, ok := data["chat_join_request"].(map[string]any)
	if !ok {
		return
	}
	chat, _ := jr["chat"].(map[string]any)
	from, _ := jr["from"].(map[string]any)
	chatID, _ := anyInt(chat["id"])
	userID, _ := anyInt(from["id"])
	if chatID == 0 || userID == 0 {
		return
	}
	tg := a.tgClient()
	var u model.User
	if a.S.DB.Where("telegram_id = ?", userID).First(&u).Error != nil || !shop.IsAvailable(&u) {
		_ = tg.DeclineJoinRequest(chatID, userID)
		return
	}
	_ = tg.ApproveJoinRequest(chatID, userID)
}

// ticketNotifyTelegram 对应 Plugin::sendTicketNotify：工单创建或用户回复后通知管理员。
func (a *App) ticketNotifyTelegram(t *model.Ticket, msg *model.TicketMessage) {
	if !a.pluginEnabled("telegram") || !truthy(a.pluginConfigValue("telegram", "enable_ticket_notify", true)) {
		return
	}
	var u model.User
	if a.S.DB.First(&u, t.UserID).Error != nil {
		return
	}
	var p model.Plan
	hasPlan := u.PlanID != nil && a.S.DB.First(&p, *u.PlanID).Error == nil
	expired := "长期有效"
	if u.ExpiredAt != nil {
		expired = time.Unix(*u.ExpiredAt, 0).Format("2006-01-02 15:04:05")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "📮 *工单提醒* #%d\n", t.ID)
	b.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(&b, "📧 邮箱: `%s`\n", u.Email)
	if hasPlan {
		fmt.Fprintf(&b, "📦 套餐: `%s`\n", escapeMD(p.Name))
		if u.TransferEnab == 0 { // 流量上限为 0 视为无限流量
			b.WriteString("📊 流量: `无限`\n")
		} else {
			fmt.Fprintf(&b, "📊 流量: `%sG / %sG` (剩余/总计)\n", gbText(u.TransferEnab-u.U-u.D), gbText(u.TransferEnab))
		}
		fmt.Fprintf(&b, "⬆️⬇️ 已用: `%sG / %sG`\n", gbText(u.U), gbText(u.D))
		fmt.Fprintf(&b, "⏰ 到期: `%s`\n", expired)
	} else {
		b.WriteString("📦 套餐: `未订购任何套餐`\n")
	}
	fmt.Fprintf(&b, "💰 余额: `%s元`\n", formatMoney(float64(u.Balance)/100))
	fmt.Fprintf(&b, "💸 佣金: `%s元`\n", formatMoney(float64(u.CommissionBal)/100))
	b.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(&b, "📝 *主题*: `%s`\n", escapeMD(t.Subject))
	fmt.Fprintf(&b, "💬 *内容*: `%s`", escapeMD(msg.Message))
	a.sendToStaff(b.String())
}

// paymentNotifyTelegram 对应 Plugin::sendPaymentNotify。
func (a *App) paymentNotifyTelegram(o *model.Order) {
	if !a.pluginEnabled("telegram") || !truthy(a.pluginConfigValue("telegram", "enable_payment_notify", true)) {
		return
	}
	var p model.Payment
	name, method := "", ""
	if o.PaymentID != nil && a.S.DB.First(&p, *o.PaymentID).Error == nil {
		name, method = p.Name, p.Payment
	}
	a.sendToStaff(fmt.Sprintf("💰成功收款%s元\n———————————————\n支付接口：%s\n支付渠道：%s\n本站订单：`%s`",
		formatMoney(float64(o.TotalAmount)/100), escapeMD(method), escapeMD(name), o.TradeNo))
}

// sendToStaff 对应 sendMessageWithAdmin(message, isStaff=true)：发给绑定了 Telegram 的管理员与员工。
func (a *App) sendToStaff(text string) {
	var users []model.User
	a.S.DB.Where("telegram_id IS NOT NULL AND (is_admin = 1 OR is_staff = 1)").Find(&users)
	tg := a.tgClient()
	for _, u := range users {
		if u.TelegramID != nil {
			_ = tg.SendMessage(*u.TelegramID, text, "markdown")
		}
	}
}

// registerTelegramRoutes 挂载 webhook，并注册工单与付款事件的通知钩子。
func (a *App) registerTelegramRoutes(r *gin.Engine) {
	r.POST("/api/v1/guest/telegram/webhook", a.telegramWebhook)
	shop.PaidHook = a.paymentNotifyTelegram
	ticketHook = a.ticketNotifyTelegram
}

// ticketHook 在工单创建或用户回复后调用（由 registerTelegramRoutes 注入）。
var ticketHook func(t *model.Ticket, msg *model.TicketMessage)

// syncTelegramCommands 对应 registerBotCommands：插件启用且配置了机器人时注册命令菜单（启动时调用一次）。
func (a *App) syncTelegramCommands() {
	if !a.pluginEnabled("telegram") || a.Set.Get("telegram_bot_token", "") == "" {
		return
	}
	a.registerBotCommands(a.tgClient())
}

// registerBotCommands 注册机器人命令菜单（后台「设置 Webhook」与启动时共用）。
func (a *App) registerBotCommands(cl *notify.TelegramClient) {
	_ = cl.SetMyCommands([]map[string]string{
		{"command": "start", "description": "开始使用"},
		{"command": "bind", "description": "绑定账号"},
		{"command": "traffic", "description": "查看流量"},
		{"command": "getlatesturl", "description": "获取订阅链接"},
		{"command": "unbind", "description": "解绑账号"},
	})
}
