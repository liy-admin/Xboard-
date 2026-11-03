package shop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// 对应 plugins-core：Epay、Bepusdt、Mgate、Btcpay、CoinPayments、Coinbase。

var errSignature = errors.New("HMAC signature does not match")

func init() {
	RegisterGateway("EPay", newEpay)
	RegisterGateway("Bepusdt", newBepusdt)
	RegisterGateway("MGate", newMgate)
	RegisterGateway("BTCPay", newBtcpay)
	RegisterGateway("CoinPayments", newCoinPayments)
	RegisterGateway("Coinbase", newCoinbase)
}

// ---- 易支付（Epay） ----

type epay struct{ cfg map[string]any }

func newEpay(cfg map[string]any) (Gateway, error) { return &epay{cfg}, nil }

func (g *epay) sign(p map[string]string) string {
	return md5Hex(rawSortedQuery(p) + cfgStr(g.cfg, "key"))
}

func (g *epay) Pay(r PayRequest) (PayResult, error) {
	p := map[string]string{
		"money":        yuan(r.TotalAmount),
		"name":         r.TradeNo,
		"notify_url":   r.NotifyURL,
		"return_url":   r.ReturnURL,
		"out_trade_no": r.TradeNo,
		"pid":          cfgStr(g.cfg, "pid"),
	}
	if t := cfgStr(g.cfg, "type"); t != "" {
		p["type"] = t
	}
	p["sign"] = g.sign(p)
	p["sign_type"] = "MD5"
	return PayResult{Type: 1, Data: cfgStr(g.cfg, "url") + "/submit.php?" + encodedSortedQuery(p)}, nil
}

func (g *epay) Notify(req NotifyRequest) (*Notification, error) {
	p := copyMap(req.Params)
	sig := p["sign"]
	delete(p, "sign")
	delete(p, "sign_type")
	if sig != g.sign(p) {
		return nil, errSignature
	}
	return &Notification{TradeNo: p["out_trade_no"], CallbackNo: p["trade_no"]}, nil
}

// ---- Bepusdt ----

type bepusdt struct{ cfg map[string]any }

func newBepusdt(cfg map[string]any) (Gateway, error) { return &bepusdt{cfg}, nil }

const bepusdtStatusSuccess = 2

// sign 对应 Bepusdt::sign：过滤空值后按键排序，拼接 k=v 再接 token，取 md5。
func (g *bepusdt) sign(p map[string]string) string {
	f := map[string]string{}
	for k, v := range p {
		if v != "" {
			f[k] = v
		}
	}
	return md5Hex(rawSortedQuery(f) + cfgStr(g.cfg, "token"))
}

func (g *bepusdt) apiURL() string {
	return strings.TrimRight(cfgStr(g.cfg, "api_url"), "/")
}

func (g *bepusdt) Pay(r PayRequest) (PayResult, error) {
	p := map[string]string{
		"order_id":     r.TradeNo,
		"notify_url":   r.NotifyURL,
		"redirect_url": r.ReturnURL,
		"amount":       yuan(r.TotalAmount),
	}
	if t := cfgStr(g.cfg, "trade_type"); t != "" {
		p["trade_type"] = t
	}
	if f := cfgStr(g.cfg, "fiat"); f != "" {
		p["fiat"] = f
	}
	if t := cfgStr(g.cfg, "timeout"); t != "" {
		if n, err := strconv.Atoi(t); err == nil {
			p["timeout"] = strconv.Itoa(n)
		}
	}
	sig := g.sign(p)
	body := map[string]any{}
	for k, v := range p {
		if k == "amount" {
			f, _ := strconv.ParseFloat(v, 64)
			body[k] = f
			continue
		}
		if k == "timeout" {
			n, _ := strconv.Atoi(v)
			body[k] = n
			continue
		}
		body[k] = v
	}
	body["signature"] = sig
	raw, _ := json.Marshal(body)
	resp, err := httpClient.Post(g.apiURL()+"/api/v1/order/create-transaction", "application/json", bytes.NewReader(raw))
	if err != nil {
		return PayResult{}, err
	}
	defer resp.Body.Close()
	var ret struct {
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&ret)
	if ret.Data == nil || Str(ret.Data["payment_url"]) == "" {
		msg := ret.Message
		if msg == "" {
			msg = "BEpusdt create order failed"
		}
		return PayResult{}, errors.New(msg)
	}
	return PayResult{Type: 1, Data: ret.Data}, nil
}

func (g *bepusdt) Notify(req NotifyRequest) (*Notification, error) {
	p := map[string]string{}
	for k, v := range req.Params {
		p[k] = v
	}
	// 回调为 JSON：优先解析原始 body，保证与 BEpusdt 签名时的值逐字节一致。
	if raw := strings.TrimSpace(string(req.Body)); raw != "" {
		var m map[string]any
		if json.Unmarshal([]byte(raw), &m) == nil {
			p = map[string]string{}
			for k, v := range m {
				if v == nil {
					continue
				}
				p[k] = anyString(v)
			}
		}
	}
	sig := p["signature"]
	delete(p, "signature")
	if sig == "" || !secureEqual(sig, g.sign(p)) {
		return nil, errSignature
	}
	st, _ := strconv.Atoi(p["status"])
	if st != bepusdtStatusSuccess {
		return nil, fmt.Errorf("status %d is not success", st)
	}
	return &Notification{TradeNo: p["order_id"], CallbackNo: p["trade_id"]}, nil
}

// anyString 把 JSON 值转为 PHP 字符串拼接时的表示。
func anyString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "1"
		}
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// ---- MGate ----

type mgate struct{ cfg map[string]any }

func newMgate(cfg map[string]any) (Gateway, error) { return &mgate{cfg}, nil }

func (g *mgate) sign(p map[string]string) string {
	return md5Hex(encodedSortedQuery(p) + cfgStr(g.cfg, "mgate_app_secret"))
}

func (g *mgate) Pay(r PayRequest) (PayResult, error) {
	p := map[string]string{
		"out_trade_no": r.TradeNo,
		"total_amount": strconv.FormatInt(r.TotalAmount, 10),
		"notify_url":   r.NotifyURL,
		"return_url":   r.ReturnURL,
		"app_id":       cfgStr(g.cfg, "mgate_app_id"),
	}
	if c := cfgStr(g.cfg, "mgate_source_currency"); c != "" {
		p["source_currency"] = c
	}
	p["sign"] = g.sign(p)
	resp, err := httpClient.PostForm(cfgStr(g.cfg, "mgate_url")+"/v1/gateway/fetch", urlValues(p))
	if err != nil {
		return PayResult{}, errors.New("网络异常")
	}
	defer resp.Body.Close()
	var ret struct {
		Message string `json:"message"`
		Data    struct {
			TradeNo string `json:"trade_no"`
			PayURL  string `json:"pay_url"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&ret)
	if ret.Data.TradeNo == "" {
		if ret.Message != "" {
			return PayResult{}, errors.New(ret.Message)
		}
		return PayResult{}, errors.New("接口请求失败")
	}
	return PayResult{Type: 1, Data: ret.Data.PayURL}, nil
}

func (g *mgate) Notify(req NotifyRequest) (*Notification, error) {
	p := copyMap(req.Params)
	sig := p["sign"]
	delete(p, "sign")
	if sig != g.sign(p) {
		return nil, errSignature
	}
	return &Notification{TradeNo: p["out_trade_no"], CallbackNo: p["trade_no"]}, nil
}

// ---- BTCPay ----

type btcpay struct{ cfg map[string]any }

func newBtcpay(cfg map[string]any) (Gateway, error) { return &btcpay{cfg}, nil }

var invoiceIDRe = regexp.MustCompile(`^[A-Za-z0-9]+$`)

func (g *btcpay) invoicesURL() string {
	return cfgStr(g.cfg, "btcpay_url") + "api/v1/stores/" + cfgStr(g.cfg, "btcpay_storeId") + "/invoices"
}

func (g *btcpay) Pay(r PayRequest) (PayResult, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonResponse": true,
		"amount":       yuan2(r.TotalAmount),
		"currency":     "CNY",
		"metadata":     map[string]string{"orderId": r.TradeNo},
	})
	req, _ := http.NewRequest(http.MethodPost, g.invoicesURL(), bytes.NewReader(body))
	req.Header.Set("Authorization", "token "+cfgStr(g.cfg, "btcpay_api_key"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return PayResult{}, errors.New("error!")
	}
	defer resp.Body.Close()
	var ret struct {
		CheckoutLink string `json:"checkoutLink"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&ret)
	if ret.CheckoutLink == "" {
		return PayResult{}, errors.New("error!")
	}
	return PayResult{Type: 1, Data: ret.CheckoutLink}, nil
}

func (g *btcpay) Notify(req NotifyRequest) (*Notification, error) {
	payload := strings.TrimSpace(string(req.Body))
	key := cfgStr(g.cfg, "btcpay_webhook_key")
	if key == "" {
		return nil, errors.New("Webhook key is not configured")
	}
	if !secureEqual(req.Headers.Get("BTCPay-Sig"), "sha256="+hmacSHA256Hex(payload, key)) {
		return nil, errSignature
	}
	var ev map[string]any
	_ = json.Unmarshal([]byte(payload), &ev)
	invoiceID := Str(ev["invoiceId"])
	if Str(ev["type"]) != "InvoiceSettled" || !invoiceIDRe.MatchString(invoiceID) {
		return nil, errors.New("not a settled invoice event")
	}
	detailURL := cfgStr(g.cfg, "btcpay_url") + "api/v1/stores/" + cfgStr(g.cfg, "btcpay_storeId") + "/invoices/" + invoiceID
	dreq, _ := http.NewRequest(http.MethodGet, detailURL, nil)
	dreq.Header.Set("Authorization", "token "+cfgStr(g.cfg, "btcpay_api_key"))
	resp, err := httpClient.Do(dreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var inv struct {
		Status   string `json:"status"`
		Metadata struct {
			OrderID string `json:"orderId"`
		} `json:"metadata"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&inv)
	if inv.Status != "Settled" || inv.Metadata.OrderID == "" {
		return nil, errors.New("invoice is not settled")
	}
	return &Notification{TradeNo: inv.Metadata.OrderID, CallbackNo: invoiceID}, nil
}

// ---- CoinPayments ----

type coinPayments struct{ cfg map[string]any }

func newCoinPayments(cfg map[string]any) (Gateway, error) { return &coinPayments{cfg}, nil }

func (g *coinPayments) Pay(r PayRequest) (PayResult, error) {
	u, err := url.Parse(r.ReturnURL)
	if err != nil {
		return PayResult{}, err
	}
	success := u.Scheme + "://" + u.Host
	p := map[string]string{
		"cmd":           "_pay_simple",
		"reset":         "1",
		"merchant":      cfgStr(g.cfg, "coinpayments_merchant_id"),
		"item_name":     r.TradeNo,
		"item_number":   r.TradeNo,
		"want_shipping": "0",
		"currency":      cfgStr(g.cfg, "coinpayments_currency"),
		"amountf":       yuan2(r.TotalAmount),
		"success_url":   success,
		"cancel_url":    r.ReturnURL,
		"ipn_url":       r.NotifyURL,
	}
	return PayResult{Type: 1, Data: "https://www.coinpayments.net/index.php?" + encodedSortedQuery(p)}, nil
}

func (g *coinPayments) Notify(req NotifyRequest) (*Notification, error) {
	p := copyMap(req.Params)
	if p["merchant"] == "" || p["merchant"] != strings.TrimSpace(cfgStr(g.cfg, "coinpayments_merchant_id")) {
		return nil, errors.New("No or incorrect Merchant ID passed")
	}
	hmacSig := req.Headers.Get("Hmac")
	calc := hmacSHA512Hex(encodedSortedQuery(p), strings.TrimSpace(cfgStr(g.cfg, "coinpayments_ipn_secret")))
	if !secureEqual(hmacSig, calc) {
		return nil, errSignature
	}
	st, _ := strconv.Atoi(p["status"])
	switch {
	case st >= 100 || st == 2:
		return &Notification{TradeNo: p["item_number"], CallbackNo: p["txn_id"], CustomResult: "IPN OK"}, nil
	case st < 0:
		return nil, errors.New("Payment Timed Out or Error")
	default:
		return &Notification{Ignore: true, CustomResult: "IPN OK: pending"}, nil
	}
}

// ---- Coinbase Commerce ----

type coinbase struct{ cfg map[string]any }

func newCoinbase(cfg map[string]any) (Gateway, error) { return &coinbase{cfg}, nil }

func (g *coinbase) Pay(r PayRequest) (PayResult, error) {
	form := url.Values{}
	form.Set("name", "订阅套餐")
	form.Set("description", "订单号 "+r.TradeNo)
	form.Set("pricing_type", "fixed_price")
	form.Set("local_price[amount]", yuan2(r.TotalAmount))
	form.Set("local_price[currency]", "CNY")
	form.Set("metadata[outTradeNo]", r.TradeNo)
	req, _ := http.NewRequest(http.MethodPost, cfgStr(g.cfg, "coinbase_url"), strings.NewReader(form.Encode()))
	req.Header.Set("X-CC-Api-Key", cfgStr(g.cfg, "coinbase_api_key"))
	req.Header.Set("X-CC-Version", "2018-03-22")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return PayResult{}, errors.New("error!")
	}
	defer resp.Body.Close()
	var ret struct {
		Data struct {
			HostedURL string `json:"hosted_url"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&ret)
	if ret.Data.HostedURL == "" {
		return PayResult{}, errors.New("error!")
	}
	return PayResult{Type: 1, Data: ret.Data.HostedURL}, nil
}

func (g *coinbase) Notify(req NotifyRequest) (*Notification, error) {
	payload := strings.TrimSpace(string(req.Body))
	key := cfgStr(g.cfg, "coinbase_webhook_key")
	if key == "" {
		return nil, errors.New("Webhook key is not configured")
	}
	if !secureEqual(req.Headers.Get("X-CC-Webhook-Signature"), hmacSHA256Hex(payload, key)) {
		return nil, errSignature
	}
	var ev struct {
		Event struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Data struct {
				Metadata struct {
					OutTradeNo string `json:"outTradeNo"`
				} `json:"metadata"`
			} `json:"data"`
		} `json:"event"`
	}
	_ = json.Unmarshal([]byte(payload), &ev)
	if ev.Event.Type != "charge:confirmed" && ev.Event.Type != "charge:resolved" {
		return nil, errors.New("event is not a settled charge")
	}
	if ev.Event.Data.Metadata.OutTradeNo == "" || ev.Event.ID == "" {
		return nil, errors.New("missing trade number")
	}
	return &Notification{TradeNo: ev.Event.Data.Metadata.OutTradeNo, CallbackNo: ev.Event.ID}, nil
}

// ---- 共用小工具 ----

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func urlValues(p map[string]string) url.Values {
	v := url.Values{}
	for k, s := range p {
		v.Set(k, s)
	}
	return v
}

// Str 与 PHP 字符串化一致的简单转换（仅用于 JSON 解码后的字段）。
func Str(v any) string {
	if v == nil {
		return ""
	}
	return anyString(v)
}
