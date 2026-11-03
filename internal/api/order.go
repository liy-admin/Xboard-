package api

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

func (a *App) shop() *shop.Service { return &shop.Service{DB: a.S.DB, Set: a.Set} }

// registerOrderRoutes 挂载用户侧订单接口（/user/order/*），路径与 V1 保持一致。
func (a *App) registerOrderRoutes(u *gin.RouterGroup) {
	u.GET("/order/fetch", a.orderFetch)
	u.GET("/order/detail", a.orderDetail)
	u.POST("/order/save", a.orderSave)
	u.POST("/order/checkout", a.orderCheckout)
	u.GET("/order/check", a.orderCheck)
	u.GET("/order/getPaymentMethod", a.orderPaymentMethods)
	u.POST("/order/cancel", a.orderCancel)
}

// orderView 对应 OrderResource：period 转为旧版写法（month_price…，用户端页面按它显示周期名），
// 套餐走 PlanResource（含 week_price 等旧版价格字段），支付方式只带 id/name/payment/icon。
type orderView struct {
	model.Order
	Period        string `json:"period"`
	Plan          any    `json:"plan,omitempty"`
	Payment       any    `json:"payment,omitempty"`
	TryOutPlanID  *int   `json:"try_out_plan_id,omitempty"`
	SurplusOrders any    `json:"surplus_orders,omitempty"`
}

func (a *App) withPlans(orders []model.Order) []orderView {
	ids := make([]int64, 0, len(orders))
	for _, o := range orders {
		ids = append(ids, o.PlanID)
	}
	plans := map[int64]model.Plan{}
	if len(ids) > 0 {
		var ps []model.Plan
		a.S.DB.Where("id IN ?", ids).Find(&ps)
		for _, p := range ps {
			plans[p.ID] = p
		}
	}
	sys := a.systemResetMethod()
	out := make([]orderView, 0, len(orders))
	for _, o := range orders {
		v := orderView{Order: o, Period: shop.LegacyPeriodName(o.Period)}
		if p, ok := plans[o.PlanID]; ok {
			pp := p
			v.Plan = planResource(&pp, sys)
		}
		out = append(out, v)
	}
	return out
}

func (a *App) orderFetch(c *gin.Context) {
	q := a.S.DB.Where("user_id = ?", cur(c).ID)
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	var orders []model.Order
	q.Order("created_at DESC").Find(&orders)
	resp.OK(c, a.withPlans(orders))
}

func (a *App) orderDetail(c *gin.Context) {
	tn := c.Query("trade_no")
	if tn == "" {
		resp.Fail(c, 422, "trade_no required")
		return
	}
	var o model.Order
	if err := a.S.DB.Where("user_id = ? AND trade_no = ?", cur(c).ID, tn).First(&o).Error; err != nil {
		resp.Fail(c, 400, "Order does not exist or has been paid")
		return
	}
	v := a.withPlans([]model.Order{o})[0]
	if v.Plan == nil {
		resp.Fail(c, 400, "Subscription plan does not exist")
		return
	}
	if o.PaymentID != nil {
		var p model.Payment
		if a.S.DB.First(&p, *o.PaymentID).Error == nil {
			v.Payment = gin.H{"id": p.ID, "name": p.Name, "payment": p.Payment, "icon": p.Icon}
		}
	}
	tryOut := a.Set.Int("try_out_plan_id", 0)
	v.TryOutPlanID = &tryOut
	// 对应 OrderController::detail：升级折抵的原订单一并返回。
	var surplusIDs []int64
	if len(o.SurplusOrderIDs) > 0 {
		_ = json.Unmarshal(o.SurplusOrderIDs, &surplusIDs)
	}
	if len(surplusIDs) > 0 {
		var so []model.Order
		a.S.DB.Where("id IN ?", surplusIDs).Find(&so)
		v.SurplusOrders = so
	}
	resp.OK(c, v)
}

func (a *App) orderSave(c *gin.Context) {
	var in struct {
		PlanID     int64  `json:"plan_id" form:"plan_id" binding:"required"`
		Period     string `json:"period" form:"period" binding:"required"`
		CouponCode string `json:"coupon_code" form:"coupon_code"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	u := cur(c)
	var pending int64
	a.S.DB.Model(&model.Order{}).Where("user_id = ? AND status IN (0, 1)", u.ID).Count(&pending)
	if pending > 0 {
		resp.Fail(c, 400, "You have an unpaid or pending order, please try again later or cancel it")
		return
	}
	var p model.Plan
	if err := a.S.DB.First(&p, in.PlanID).Error; err != nil {
		resp.Fail(c, 400, "Subscription plan does not exist")
		return
	}
	o, err := a.shop().CreateOrder(u, &p, in.Period, in.CouponCode)
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	resp.OK(c, o.TradeNo)
}

func (a *App) orderCheckout(c *gin.Context) {
	var in struct {
		TradeNo string `json:"trade_no" form:"trade_no" binding:"required"`
		Method  int64  `json:"method" form:"method"`
		Token   string `json:"token" form:"token"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "Invalid parameters")
		return
	}
	u := cur(c)
	var o model.Order
	if err := a.S.DB.Where("trade_no = ? AND user_id = ? AND status = 0", in.TradeNo, u.ID).First(&o).Error; err != nil {
		resp.Fail(c, 400, "Order does not exist or has been paid")
		return
	}
	svc := a.shop()
	if o.TotalAmount <= 0 {
		if err := svc.Paid(o.TradeNo, ""); err != nil {
			resp.Fail(c, 400, "支付失败")
			return
		}
		c.JSON(200, gin.H{"type": -1, "data": true})
		return
	}
	var p model.Payment
	if err := a.S.DB.First(&p, in.Method).Error; err != nil || !p.Enable {
		resp.Fail(c, 400, "Payment method is not available")
		return
	}
	var handling *int64
	if (p.HandlingFeeFixed != nil && *p.HandlingFeeFixed != 0) || (p.HandlingFeePercent != nil && *p.HandlingFeePercent != 0) {
		pct := 0.0
		if p.HandlingFeePercent != nil {
			pct = *p.HandlingFeePercent
		}
		fixed := int64(0)
		if p.HandlingFeeFixed != nil {
			fixed = *p.HandlingFeeFixed
		}
		h := int64(math.Round(float64(o.TotalAmount)*(pct/100) + float64(fixed)))
		handling = &h
	}
	pid := p.ID
	o.PaymentID = &pid
	o.HandlingAmount = handling
	if err := a.S.DB.Model(&o).Updates(map[string]any{"payment_id": pid, "handling_amount": handling}).Error; err != nil {
		resp.Fail(c, 400, "Request failed, please try again later")
		return
	}
	total := o.TotalAmount
	if handling != nil {
		total += *handling
	}
	cfg, err := a.paymentConfig(&p)
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	gw, err := shop.NewGateway(p.Payment, cfg)
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	res, err := gw.Pay(shop.PayRequest{
		TradeNo: o.TradeNo, TotalAmount: total, UserID: o.UserID, StripeToken: in.Token,
		NotifyURL: a.notifyURL(c, &p), ReturnURL: a.Set.Get("app_url", "") + "/#/order/" + o.TradeNo,
		AppName: a.Set.Get("app_name", "Lets VPN"),
	})
	if err != nil {
		resp.Fail(c, 400, err.Error())
		return
	}
	c.JSON(200, gin.H{"type": res.Type, "data": res.Data})
}

// paymentConfig 解析 v2_payment.config（JSON 字符串）并附加 PHP 版注入的 enable/id/uuid 字段。
func (a *App) paymentConfig(p *model.Payment) (map[string]any, error) {
	cfg := map[string]any{}
	if p.Config != "" {
		if err := json.Unmarshal([]byte(p.Config), &cfg); err != nil {
			return nil, errors.New("payment config is invalid")
		}
	}
	cfg["enable"] = p.Enable
	cfg["id"] = p.ID
	cfg["uuid"] = p.UUID
	cfg["notify_domain"] = ""
	if p.NotifyDomain != nil {
		cfg["notify_domain"] = *p.NotifyDomain
	}
	return cfg, nil
}

// notifyURL 对应 PaymentService::pay 中的回调地址拼接（支持自定义 notify_domain）。
func (a *App) notifyURL(c *gin.Context, p *model.Payment) string {
	base := a.Set.Get("app_url", "")
	if base == "" {
		base = "http://" + c.Request.Host
	}
	u := base + "/api/v1/guest/payment/notify/" + p.Payment + "/" + p.UUID
	if p.NotifyDomain != nil && *p.NotifyDomain != "" {
		u = strings.TrimRight(*p.NotifyDomain, "/") + "/api/v1/guest/payment/notify/" + p.Payment + "/" + p.UUID
	}
	return u
}

func (a *App) orderCheck(c *gin.Context) {
	var o model.Order
	if err := a.S.DB.Where("trade_no = ? AND user_id = ?", c.Query("trade_no"), cur(c).ID).First(&o).Error; err != nil {
		resp.Fail(c, 400, "Order does not exist")
		return
	}
	resp.OK(c, o.Status)
}

func (a *App) orderPaymentMethods(c *gin.Context) {
	var ps []model.Payment
	a.S.DB.Where("enable = 1").Order("sort ASC").Find(&ps)
	out := make([]gin.H, 0, len(ps))
	for _, p := range ps {
		out = append(out, gin.H{
			"id": p.ID, "name": p.Name, "payment": p.Payment, "icon": p.Icon,
			"handling_fee_fixed": p.HandlingFeeFixed, "handling_fee_percent": p.HandlingFeePercent,
		})
	}
	resp.OK(c, out)
}

func (a *App) orderCancel(c *gin.Context) {
	var in struct {
		TradeNo string `json:"trade_no" form:"trade_no"`
	}
	_ = c.ShouldBind(&in)
	if in.TradeNo == "" {
		resp.Fail(c, 422, "Invalid parameter")
		return
	}
	var o model.Order
	if err := a.S.DB.Where("trade_no = ? AND user_id = ?", in.TradeNo, cur(c).ID).First(&o).Error; err != nil {
		resp.Fail(c, 400, "Order does not exist")
		return
	}
	if o.Status != shop.StatusPending {
		resp.Fail(c, 400, "You can only cancel pending orders")
		return
	}
	if err := a.shop().Cancel(&o); err != nil {
		resp.Fail(c, 400, "Cancel failed")
		return
	}
	resp.OK(c, true)
}

// guestPaymentNotify 对应 V1 Guest PaymentController::notify：验签后推进订单。
func (a *App) guestPaymentNotify(c *gin.Context) {
	method, uuid := c.Param("method"), c.Param("uuid")
	var p model.Payment
	if err := a.S.DB.Where("payment = ? AND uuid = ?", method, uuid).First(&p).Error; err != nil {
		resp.Fail(c, 400, "payment not found")
		return
	}
	cfg, err := a.paymentConfig(&p)
	if err != nil || !p.Enable {
		resp.Fail(c, 400, "gate is not enable")
		return
	}
	gw, err := shop.NewGateway(p.Payment, cfg)
	if err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	params := map[string]string{}
	for k, v := range c.Request.URL.Query() {
		params[k] = v[0]
	}
	if err := c.Request.ParseForm(); err == nil {
		for k, v := range c.Request.PostForm {
			params[k] = v[0]
		}
	}
	raw, _ := io.ReadAll(c.Request.Body)
	n, err := gw.Notify(shop.NotifyRequest{Params: params, Headers: c.Request.Header, Body: raw})
	if err != nil {
		resp.Fail(c, 422, "verify error")
		return
	}
	if n.Ignore {
		c.String(200, n.CustomResult)
		return
	}
	var o model.Order
	if err := a.S.DB.Where("trade_no = ?", n.TradeNo).First(&o).Error; err != nil {
		resp.Fail(c, 400, "order is not found")
		return
	}
	if o.Status == shop.StatusPending {
		if err := a.shop().Paid(n.TradeNo, n.CallbackNo); err != nil {
			resp.Fail(c, 400, "handle error")
			return
		}
	}
	body := n.CustomResult
	if body == "" {
		body = "success"
	}
	c.String(200, body)
}
