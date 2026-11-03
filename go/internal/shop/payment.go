package shop

import (
	"fmt"
	"net/http"
	"sync"
)

// PayRequest 是下单支付时传给网关的参数（对应 PaymentService::pay 的入参）。
type PayRequest struct {
	TradeNo     string
	TotalAmount int64
	UserID      int64
	NotifyURL   string
	ReturnURL   string
	StripeToken string
	AppName     string
}

// PayResult 与 PHP 网关返回的 {type, data} 对应：type 决定前端如何展示 data。
type PayResult struct {
	Type int
	Data any
}

// NotifyRequest 是网关回调的原始输入：表单参数、请求头与原始 body（签名常基于原始 body 计算）。
type NotifyRequest struct {
	Params  map[string]string
	Headers http.Header
	Body    []byte
}

// Notification 为网关回调验签后的结果。
type Notification struct {
	TradeNo    string
	CallbackNo string
	// Ignore 表示回调合法但订单尚未支付（如 pending），不推进订单，只回应 CustomResult。
	Ignore bool
	// CustomResult 为回调应答体，缺省时返回 "success"。
	CustomResult string
}

// Gateway 是支付网关的统一接口。原 PHP 插件（plugins-core/*）逐个实现该接口。
type Gateway interface {
	Pay(req PayRequest) (PayResult, error)
	// Notify 校验回调签名。签名不通过返回错误。
	Notify(req NotifyRequest) (*Notification, error)
}

// GatewayFactory 根据后台保存的配置（v2_payment.config）构造网关实例。
type GatewayFactory func(config map[string]any) (Gateway, error)

var (
	gwMu       sync.RWMutex
	gwRegistry = map[string]GatewayFactory{}
)

// RegisterGateway 注册网关实现，name 对应 v2_payment.payment 字段。
func RegisterGateway(name string, f GatewayFactory) {
	gwMu.Lock()
	defer gwMu.Unlock()
	gwRegistry[name] = f
}

// NewGateway 按名称构造网关；未注册时返回明确错误，不静默失败。
func NewGateway(name string, config map[string]any) (Gateway, error) {
	gwMu.RLock()
	f, ok := gwRegistry[name]
	gwMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("payment gateway %q is not implemented", name)
	}
	return f(config)
}

// GatewayNames 返回已注册的支付网关名称（后台 getPaymentMethods 使用）。
func GatewayNames() []string {
	gwMu.RLock()
	defer gwMu.RUnlock()
	out := make([]string, 0, len(gwRegistry))
	for n := range gwRegistry {
		out = append(out, n)
	}
	return out
}
