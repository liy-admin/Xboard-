package api

import (
	"crypto/rand"
	"math/big"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/resp"
)

// registerUserExt 挂载 V1 用户侧扩展接口（邀请、推荐、抽奖、公告、知识库、流量、客服、优惠券）。
func (a *App) registerUserExt(u *gin.RouterGroup) {
	u.GET("/invite/save", a.inviteSave)
	u.GET("/invite/fetch", a.inviteFetch)
	u.GET("/invite/details", a.inviteDetails)
	u.GET("/referral/info", a.referralInfo)
	u.GET("/lucky_draw/info", a.luckyInfo)
	u.POST("/lucky_draw/claim", a.luckyClaim)
	u.GET("/notice/fetch", a.noticeFetch)
	u.GET("/knowledge/fetch", a.knowledgeFetch)
	u.GET("/knowledge/getCategory", a.knowledgeCategory)
	u.GET("/stat/getTrafficLog", a.trafficLog)
	u.GET("/support/config", a.supportConfig)
	u.POST("/coupon/check", a.couponCheck)
	a.registerUserMisc(u)
	a.registerTicketRoutes(u)
	u.GET("/devices/fetch", a.deviceFetch)
	u.POST("/gift-card/check", a.giftCheck)
	u.POST("/gift-card/redeem", a.giftRedeem)
	u.GET("/gift-card/history", a.giftHistory)
	u.GET("/gift-card/detail", a.giftDetail)
	u.GET("/gift-card/types", a.giftTypes)
	u.POST("/devices/kick", a.deviceKick)
}

// randomChars 对应 Helper::randomChar：大小写字母与数字。
func randomChars(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	out := make([]byte, n)
	for i := range out {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		out[i] = charset[v.Int64()]
	}
	return string(out)
}

func (a *App) inviteSave(c *gin.Context) {
	u := cur(c)
	var n int64
	a.S.DB.Model(&model.InviteCode{}).Where("user_id = ? AND status = 0", u.ID).Count(&n)
	if n >= int64(a.Set.Int("invite_gen_limit", 5)) {
		resp.Fail(c, 400, "The maximum number of creations has been reached")
		return
	}
	ts := now()
	code := model.InviteCode{UserID: u.ID, Code: randomChars(8), CreatedAt: ts, UpdatedAt: ts}
	if err := a.S.DB.Create(&code).Error; err != nil {
		resp.Fail(c, 500, "Internal error")
		return
	}
	resp.OK(c, true)
}

// inviteStats 计算邀请页统计，对应 InviteController::fetch 的五个数字。
// 返回 [已注册人数, 有效佣金, 确认中佣金, 佣金比例, 可用佣金]。
func (a *App) inviteStats(u *model.User) []any {
	rate := int64(a.Set.Int("invite_commission", 10))
	if u.CommissionRate != nil && *u.CommissionRate != 0 {
		rate = *u.CommissionRate
	}
	var registered, gotAmount, unchecked int64
	a.S.DB.Model(&model.User{}).Where("invite_user_id = ?", u.ID).Count(&registered)
	a.S.DB.Model(&model.CommissionLog{}).Where("invite_user_id = ?", u.ID).
		Select("COALESCE(SUM(get_amount),0)").Scan(&gotAmount)
	a.S.DB.Model(&model.Order{}).Where("status = 3 AND commission_status = 0 AND invite_user_id = ?", u.ID).
		Select("COALESCE(SUM(commission_balance),0)").Scan(&unchecked)

	var pending any = unchecked
	if a.Set.Bool("commission_distribution_enable", false) {
		l1, _ := strconv.ParseFloat(a.Set.Get("commission_distribution_l1", "0"), 64)
		pending = float64(unchecked) * (l1 / 100)
	}
	return []any{registered, gotAmount, pending, rate, u.CommissionBal}
}

func (a *App) inviteFetch(c *gin.Context) {
	u := cur(c)
	var codes []model.InviteCode
	a.S.DB.Where("user_id = ? AND status = 0", u.ID).Find(&codes)
	resp.OK(c, gin.H{"codes": codes, "stat": a.inviteStats(u)})
}

func (a *App) inviteDetails(c *gin.Context) {
	u := cur(c)
	current, pageSize := pageParams(c)
	q := a.S.DB.Model(&model.CommissionLog{}).Where("invite_user_id = ? AND get_amount > 0", u.ID)
	var total int64
	q.Count(&total)
	var rows []model.CommissionLog
	q.Order("created_at DESC").Offset((current - 1) * pageSize).Limit(pageSize).Find(&rows)
	// 旧接口直接返回 {data,total}，不包 status/message 外壳。
	c.JSON(200, gin.H{"data": rows, "total": total})
}

// pageParams 复刻 PHP 的分页参数处理：current 默认 1，page_size 不小于 10。
func pageParams(c *gin.Context) (int, int) {
	current, _ := strconv.Atoi(c.Query("current"))
	if current < 1 {
		current = 1
	}
	size, _ := strconv.Atoi(c.Query("page_size"))
	if size < 10 {
		size = 10
	}
	return current, size
}

// shareLink 对应 ReferralController::buildShareLink。
func shareLink(template, appURL, code string) string {
	if template != "" {
		s := strings.ToLower(template)
		if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
			template = ""
		}
	}
	if template != "" {
		if strings.Contains(template, ":code") {
			return strings.ReplaceAll(template, ":code", code)
		}
		return template
	}
	if appURL == "" {
		return ""
	}
	return strings.TrimRight(appURL, "/") + "/#/register?referral_code=" + code
}

// newReferralCode 生成 000000-999999 的 6 位数字推荐码。
func newReferralCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	s := strconv.FormatInt(n.Int64(), 10)
	return strings.Repeat("0", 6-len(s)) + s
}

func (a *App) referralInfo(c *gin.Context) {
	u := cur(c)
	if u.ReferralCode == nil || *u.ReferralCode == "" {
		code := a.assignReferralCode(u)
		u.ReferralCode = &code
	}
	code := *u.ReferralCode
	var invited int64
	a.S.DB.Model(&model.ReferralDurationLog{}).Where("inviter_id = ?", u.ID).
		Distinct("invitee_id").Count(&invited)
	var secs int64
	a.S.DB.Model(&model.ReferralDurationLog{}).Where("inviter_id = ?", u.ID).
		Select("COALESCE(SUM(seconds_credited),0)").Scan(&secs)

	var share any
	if l := shareLink(a.Set.Get("referral_share_url", ""), a.Set.Get("app_url", ""), code); l != "" {
		share = l
	}
	resp.OK(c, gin.H{
		"referral_code":  code,
		"share_link":     share,
		"invite_count":   invited,
		"total_hours":    roundOne(float64(secs) / 3600),
		"reward_percent": a.Set.Float("referral_duration_percent", 20),
		"headline":       a.Set.Get("referral_headline", "推荐朋友，领永久会员"),
		"subtext": a.Set.Get("referral_subtext",
			"好友安装快连后，填写您的推荐码(:code)即算推荐成功；当其每次获得会员，您均可薅取 :percent% 的时长！永久有效！"),
	})
}

func roundOne(f float64) float64 {
	return float64(int64(f*10+0.5)) / 10
}

// assignReferralCode 首次访问时按需生成推荐码，冲突则重抽。
func (a *App) assignReferralCode(u *model.User) string {
	for {
		code := newReferralCode()
		var n int64
		a.S.DB.Model(&model.User{}).Where("referral_code = ?", code).Count(&n)
		if n == 0 {
			a.S.DB.Model(u).Update("referral_code", code)
			return code
		}
	}
}

func (a *App) couponCheck(c *gin.Context) {
	u := cur(c)
	var in struct {
		Code   string `json:"code" form:"code"`
		PlanID int64  `json:"plan_id" form:"plan_id"`
		Period string `json:"period" form:"period"`
	}
	_ = c.ShouldBind(&in)
	if in.Code == "" {
		resp.Fail(c, 422, "Coupon cannot be empty")
		return
	}
	cp, err := a.shop().CheckCoupon(in.Code, in.PlanID, u.ID, in.Period)
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	resp.OK(c, couponView(cp))
}

// couponView 对应 CouponResource：限制周期转换为旧版键名，限制套餐转为字符串数组。
func couponView(cp *model.Coupon) map[string]any {
	planIDs := idStrings(cp.LimitPlanIDs)
	periods := periodLegacyNames(cp.LimitPeriod)
	return gin.H{
		"id": cp.ID, "code": cp.Code, "name": cp.Name, "type": cp.Type, "value": cp.Value,
		"show": cp.Show, "limit_use": cp.LimitUse, "limit_use_with_user": cp.LimitUseWithUser,
		"limit_plan_ids": planIDs, "limit_period": periods,
		"started_at": cp.StartedAt, "ended_at": cp.EndedAt,
		"created_at": cp.CreatedAt, "updated_at": cp.UpdatedAt,
	}
}
