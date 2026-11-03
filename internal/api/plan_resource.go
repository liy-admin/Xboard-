package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 套餐资源格式化，对应 app/Http/Resources/PlanResource.php。
// 价格以旧版字段名（month_price 等）输出，金额乘以 100（PHP 的 PRICE_MULTIPLIER）。

const priceMultiplier = 100

// systemResetMethod 对应 admin_setting('reset_traffic_method', MONTHLY)，流量重置方式的文字与 PHP 的 default 分支一致。
func resetMethodText(method *int, sys int) string {
	m := sys
	if method != nil {
		m = *method
	}
	switch m {
	case shop.ResetFirstDayMonth:
		return "每月1号"
	case shop.ResetMonthly:
		return "按月"
	case shop.ResetNever:
		return "不重置"
	case shop.ResetFirstDayYear:
		return "每年1月1日"
	case shop.ResetYearly:
		return "按年"
	}
	return "按月"
}

// planContent 替换套餐描述中的模板变量。
func planContent(p *model.Plan, resetText string) string {
	if p.Content == nil {
		return ""
	}
	speed := "不限制"
	if p.SpeedLimit != nil {
		speed = itoa64(*p.SpeedLimit)
	}
	devices := "不限制"
	if p.DeviceLimit != nil {
		devices = itoa64(*p.DeviceLimit)
	}
	return strings.NewReplacer(
		"{{transfer}}", itoa64(derefInt64(p.TransferEnable)),
		"{{speed}}", speed,
		"{{devices}}", devices,
		"{{reset_method}}", resetText,
	).Replace(*p.Content)
}

func itoa64(v int64) string { return strconv.FormatInt(v, 10) }

// planCapacity 对应 getFormattedCapacityLimit：null 保持 null，<=0 为 "已售罄"。
func planCapacity(v *int64) any {
	if v == nil {
		return nil
	}
	if *v <= 0 {
		return "已售罄"
	}
	return *v
}

// planResource 生成与 PHP PlanResource::toArray 同结构的映射。sysReset 为系统流量重置设置。
func planResource(p *model.Plan, sysReset int) gin.H {
	prices := shop.PlanPrices(p)
	out := gin.H{
		"id": p.ID, "group_id": p.GroupID, "name": p.Name, "tags": json.RawMessage(nilRaw(p.Tags)),
		"content":         planContent(p, resetMethodText(p.ResetTrafficMethod, sysReset)),
		"capacity_limit":  planCapacity(p.CapacityLimit),
		"transfer_enable": p.TransferEnable, "speed_limit": p.SpeedLimit, "device_limit": p.DeviceLimit,
		"show": p.Show, "sell": p.Sell, "renew": p.Renew,
		"reset_traffic_method": p.ResetTrafficMethod, "sort": p.Sort,
		"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
	}
	for legacy, newKey := range shop.LegacyPeriodMapping() {
		if v, ok := prices[newKey]; ok {
			out[legacy] = v * priceMultiplier
		} else {
			out[legacy] = nil
		}
	}
	return out
}

// planRaw 与 PHP 直接序列化 Plan 模型的结果一致（App\Models\Plan::toArray）。
// show / renew 有布尔转换；sell 没有，保持数据库里的 0/1；tags / prices 按 JSON 原样输出。
func planRaw(p *model.Plan) gin.H {
	b2i := func(v bool) int {
		if v {
			return 1
		}
		return 0
	}
	var tags any
	if len(p.Tags) > 0 {
		_ = json.Unmarshal(p.Tags, &tags)
	}
	var prices any
	if len(p.Prices) > 0 {
		_ = json.Unmarshal(p.Prices, &prices)
	}
	return gin.H{
		"id": p.ID, "group_id": p.GroupID, "transfer_enable": p.TransferEnable, "name": p.Name,
		"speed_limit": p.SpeedLimit, "show": p.Show, "sort": p.Sort, "renew": p.Renew,
		"content": p.Content, "prices": prices, "reset_traffic_method": p.ResetTrafficMethod,
		"capacity_limit": p.CapacityLimit, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		"sell": b2i(p.Sell), "device_limit": p.DeviceLimit, "tags": tags,
	}
}

func nilRaw(r json.RawMessage) []byte {
	if len(r) == 0 {
		return []byte("null")
	}
	return r
}

func (a *App) systemResetMethod() int { return a.Set.Int("reset_traffic_method", shop.ResetMonthly) }

// planFetch 对应 V1 PlanController::fetch：指定 id 返回单个（需对用户可购或为续费），否则返回可售列表。
func (a *App) planFetchV1(c *gin.Context) {
	u := cur(c)
	sys := a.systemResetMethod()
	s := a.shop()
	if id := c.Query("id"); id != "" {
		var p model.Plan
		if a.S.DB.Where("id = ?", id).First(&p).Error != nil || !planAvailableForUser(s, &p, u) {
			resp.Fail(c, 400, "Subscription plan does not exist")
			return
		}
		resp.OK(c, planResource(&p, sys))
		return
	}
	var plans []model.Plan
	a.S.DB.Where("`show` = ? AND sell = ?", true, true).Order("sort").Find(&plans)
	out := make([]gin.H, 0, len(plans))
	for i := range plans {
		if s.HasCapacity(&plans[i]) {
			out = append(out, planResource(&plans[i], sys))
		}
	}
	resp.OK(c, out)
}

// planAvailableForUser 对应 PlanService::isPlanAvailableForUser。
func planAvailableForUser(s *shop.Service, p *model.Plan, u *model.User) bool {
	if u.PlanID != nil && *u.PlanID == p.ID {
		return p.Renew
	}
	return p.Show && p.Sell && s.HasCapacity(p)
}

// resetDayOf 对应 UserService::getResetDay：距下次重置的天数（向上取整），无重置时间返回 nil。
func (a *App) resetDayOf(u *model.User) any {
	var plan *model.Plan
	if u.PlanID != nil {
		var p model.Plan
		if a.S.DB.First(&p, *u.PlanID).Error == nil {
			plan = &p
		}
	}
	next := a.shop().CalcNextReset(u, plan, time.Now())
	if next == nil {
		return nil
	}
	now := time.Now().Unix()
	if next.Unix() <= now {
		return 0
	}
	return (next.Unix() - now + 86399) / 86400
}

// getSubscribe 对应 V1 UserController::getSubscribe。
func (a *App) getSubscribe(c *gin.Context) {
	u := cur(c)
	var fresh model.User
	if err := a.S.DB.First(&fresh, u.ID).Error; err != nil {
		resp.Fail(c, 400, "The user does not exist")
		return
	}
	out := gin.H{
		"id": fresh.ID, "plan_id": fresh.PlanID, "token": fresh.Token, "expired_at": fresh.ExpiredAt,
		"u": fresh.U, "d": fresh.D, "transfer_enable": fresh.TransferEnab, "email": fresh.Email,
		"uuid": fresh.UUID, "device_limit": fresh.DeviceLimit, "speed_limit": fresh.SpeedLimit,
		"next_reset_at": fresh.NextResetAt,
	}
	if fresh.PlanID != nil {
		var p model.Plan
		if err := a.S.DB.First(&p, *fresh.PlanID).Error; err != nil {
			resp.Fail(c, 400, "Subscription plan does not exist")
			return
		}
		// PHP 这里返回的是原始 Plan 模型（Plan::find），不是 PlanResource：
		// 保留 prices 对象、未替换模板变量的 content、原始 capacity_limit，且没有旧版 *_price 字段。
		out["plan"] = planRaw(&p)
	}
	out["subscribe_url"] = a.encSubscribeURL(fresh.Token)
	out["enc_key"] = a.encKey(&fresh)
	out["reset_day"] = a.resetDayOf(&fresh)
	resp.OK(c, out)
}

// resetSecurity 对应 V1 UserController::resetSecurity：重置 uuid、token 与 enc_key，返回普通订阅地址。
func (a *App) resetSecurity(c *gin.Context) {
	u := cur(c)
	upd := map[string]any{
		"uuid":    uuid.NewString(),
		"token":   auth.RandHex(16),
		"enc_key": newEncKey(),
	}
	if err := a.S.DB.Model(&model.User{}).Where("id = ?", u.ID).Updates(upd).Error; err != nil {
		resp.Fail(c, 400, "Reset failed")
		return
	}
	resp.OK(c, a.subscribeURL(upd["token"].(string), ""))
}
