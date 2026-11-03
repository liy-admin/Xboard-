package api

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/notify"
	"xboard/internal/resp"
	"xboard/internal/sub"
)

// 用户侧杂项接口：统计、通讯配置、Stripe 公钥、Telegram 机器人信息、V1 节点列表。

func (a *App) registerUserMisc(u *gin.RouterGroup) {
	u.GET("/getStat", a.userGetStat)
	u.POST("/transfer", a.userTransfer)
	u.GET("/getActiveSession", a.userSessions)
	u.POST("/removeActiveSession", a.userRemoveSession)
	u.POST("/cancelAccount", a.userCancelAccount)
	u.POST("/getQuickLoginUrl", a.userQuickLoginURL)
	u.POST("/update", a.userUpdate)
	u.GET("/comm/config", a.commConfig)
	u.POST("/comm/getStripePublicKey", a.commStripeKey)
	u.GET("/telegram/getBotInfo", a.telegramBotInfo)
	u.GET("/server/fetch", a.serverFetchV1)
	u.GET("/server/revision", a.serverRevisionV1)
	u.GET("/server/countries", a.serverCountries)
	u.POST("/server/synced", a.serverSyncedV1)
}

// userGetStat 对应 UserController::getStat：[待支付订单数, 待处理工单数, 邀请人数]。
func (a *App) userGetStat(c *gin.Context) {
	u := cur(c)
	var orders, tickets, invited int64
	a.S.DB.Model(&model.Order{}).Where("status = 0 AND user_id = ?", u.ID).Count(&orders)
	a.S.DB.Model(&model.Ticket{}).Where("status = 0 AND user_id = ?", u.ID).Count(&tickets)
	a.S.DB.Model(&model.User{}).Where("invite_user_id = ?", u.ID).Count(&invited)
	resp.OK(c, []int64{orders, tickets, invited})
}

// settingOrNil 对应 admin_setting($key) 不带默认值时的返回：缺失为 null。
func (a *App) settingOrNil(name string) any {
	if !a.Set.Has(name) {
		return nil
	}
	return a.Set.Get(name, "")
}

// commConfig 对应 CommController::config。
func (a *App) commConfig(c *gin.Context) {
	resp.OK(c, gin.H{
		"is_telegram":                    boolInt(a.Set.Bool("telegram_bot_enable", false)),
		"telegram_discuss_link":          a.settingOrNil("telegram_discuss_link"),
		"stripe_pk":                      a.settingOrNil("stripe_pk_live"),
		"withdraw_methods":               a.withdrawMethods(),
		"withdraw_close":                 boolInt(a.Set.Bool("withdraw_close_enable", false)),
		"currency":                       a.Set.Get("currency", "CNY"),
		"currency_symbol":                a.Set.Get("currency_symbol", "¥"),
		"commission_distribution_enable": boolInt(a.Set.Bool("commission_distribution_enable", false)),
		"commission_distribution_l1":     a.settingOrNil("commission_distribution_l1"),
		"commission_distribution_l2":     a.settingOrNil("commission_distribution_l2"),
		"commission_distribution_l3":     a.settingOrNil("commission_distribution_l3"),
	})
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// commStripeKey 对应 CommController::getStripePublicKey：返回 StripeCredit 支付方式配置中的 stripe_pk_live。
func (a *App) commStripeKey(c *gin.Context) {
	var in struct {
		ID int64 `json:"id" form:"id"`
	}
	_ = c.ShouldBind(&in)
	var p model.Payment
	if err := a.S.DB.Where("id = ? AND payment = ?", in.ID, "StripeCredit").First(&p).Error; err != nil {
		resp.Fail(c, 500, "payment is not found")
		return
	}
	cfg := map[string]any{}
	_ = json.Unmarshal([]byte(p.Config), &cfg)
	resp.OK(c, cfg["stripe_pk_live"])
}

// telegramBotInfo 对应 TelegramController::getBotInfo。
func (a *App) telegramBotInfo(c *gin.Context) {
	tg := notify.NewTelegram(a.Set.Get("telegram_bot_token", ""))
	raw, err := tg.GetMe()
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	var me struct {
		Username string `json:"username"`
	}
	_ = json.Unmarshal(raw, &me)
	resp.OK(c, gin.H{"username": me.Username})
}

// serverListV1 对应 ServerController::fetch 的数据来源：用户可用时返回可用节点，否则为空。
func (a *App) serverListV1(u *model.User) ([]sub.Map, error) {
	if !userAvailable(u) {
		return []sub.Map{}, nil
	}
	servers, err := sub.AvailableServers(bgCtx(), a.S.DB, a.S.Redis, u)
	if err != nil {
		return nil, err
	}
	return servers, nil
}

// nodeRevision 对应 ServerService::calcNodeRevision：sha1(json_encode(cache_key 列表))。
func nodeRevision(servers []sub.Map) string {
	keys := make([]string, 0, len(servers))
	for _, s := range servers {
		keys = append(keys, sub.Str(s["cache_key"]))
	}
	b, _ := json.Marshal(keys)
	return fmt.Sprintf("%x", sha1.Sum(b))
}

// nodeResourceV1 对应 NodeResource 的字段。
func (a *App) nodeResourceV1(s sub.Map) gin.H {
	code := sub.Str(s["code"])
	var codeV any
	if code != "" {
		codeV = code
	}
	var lastCheck any
	if v, err := a.S.Redis.Get(bgCtx(), fmt.Sprintf("SERVER_%s_LAST_CHECK_AT_%s", strings.ToUpper(sub.Str(s["type"])), sub.Str(s["id"]))).Int64(); err == nil {
		lastCheck = v
	}
	return gin.H{
		"id": s["id"], "code": codeV, "type": s["type"], "version": nil, "name": s["name"],
		"rate": s["rate"], "tags": s["tags"], "is_online": sub.Int(s["is_online"]),
		"cache_key": s["cache_key"], "last_check_at": lastCheck,
	}
}

func (a *App) currentUserFresh(c *gin.Context) (*model.User, bool) {
	var u model.User
	if err := a.S.DB.First(&u, cur(c).ID).Error; err != nil {
		resp.Fail(c, 400, "The user does not exist")
		return nil, false
	}
	return &u, true
}

// serverFetchV1 对应 V1 ServerController::fetch，支持 If-None-Match 返回 304。
func (a *App) serverFetchV1(c *gin.Context) {
	u, ok := a.currentUserFresh(c)
	if !ok {
		return
	}
	servers, err := a.serverListV1(u)
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	etag := nodeRevision(servers)
	if strings.Contains(c.GetHeader("If-None-Match"), etag) {
		c.Status(304)
		return
	}
	data := make([]gin.H, 0, len(servers))
	for _, s := range servers {
		data = append(data, a.nodeResourceV1(s))
	}
	c.Header("ETag", "\""+etag+"\"")
	c.JSON(200, gin.H{"data": data})
}

// serverRevisionV1 对应 V1 ServerController::revision。
func (a *App) serverRevisionV1(c *gin.Context) {
	u, ok := a.currentUserFresh(c)
	if !ok {
		return
	}
	servers, err := a.serverListV1(u)
	if err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"revision": nodeRevision(servers)}})
}

// serverSyncedV1 对应 V1 ServerController::synced：记录客户端已同步的 revision。
func (a *App) serverSyncedV1(c *gin.Context) {
	var in struct {
		Revision string `json:"revision" form:"revision" binding:"required"`
	}
	if err := c.ShouldBind(&in); err != nil || len(in.Revision) != 40 {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	u := cur(c)
	// PHP 在这里只记录警告，写入失败不影响返回值。
	a.S.DB.Model(&model.User{}).Where("id = ?", u.ID).
		Updates(map[string]any{"node_revision": in.Revision, "node_revision_at": time.Now().Unix()})
	resp.OK(c, true)
}
