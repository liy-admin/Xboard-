package shop

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRegistryHasAllPluginGateways(t *testing.T) {
	for _, name := range []string{"EPay", "AlipayF2F", "Bepusdt", "MGate", "BTCPay", "CoinPayments", "Coinbase"} {
		if _, err := NewGateway(name, map[string]any{}); err != nil {
			t.Errorf("gateway %s not registered: %v", name, err)
		}
	}
	if _, err := NewGateway("Nope", nil); err == nil {
		t.Error("unknown gateway must error")
	}
}

func TestEpayRoundTrip(t *testing.T) {
	cfg := map[string]any{"url": "https://pay.example", "pid": "1001", "key": "secret"}
	g, _ := newEpay(cfg)
	res, err := g.Pay(PayRequest{TradeNo: "T1", TotalAmount: 1050, NotifyURL: "https://n", ReturnURL: "https://r"})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(res.Data.(string))
	q := u.Query()
	if q.Get("money") != "10.5" || q.Get("sign_type") != "MD5" || q.Get("sign") == "" {
		t.Fatalf("bad submit url: %s", res.Data)
	}
	params := map[string]string{}
	for k := range q {
		params[k] = q.Get(k)
	}
	params["trade_no"] = "PAY-9"
	delete(params, "sign_type")
	params["sign"] = md5Hex(rawSortedQuery(withoutKeys(params, "sign")) + "secret")
	n, err := g.Notify(NotifyRequest{Params: params})
	if err != nil || n.TradeNo != "T1" || n.CallbackNo != "PAY-9" {
		t.Fatalf("notify failed: %v %+v", err, n)
	}
	params["sign"] = "bad"
	if _, err := g.Notify(NotifyRequest{Params: params}); err == nil {
		t.Error("bad signature must be rejected")
	}
}

func withoutKeys(m map[string]string, keys ...string) map[string]string {
	c := copyMap(m)
	for _, k := range keys {
		delete(c, k)
	}
	return c
}

func TestBepusdtNotifyStatus(t *testing.T) {
	g, _ := newBepusdt(map[string]any{"token": "tk"})
	bg := g.(*bepusdt)
	payload := map[string]any{"order_id": "T2", "trade_id": "TX", "status": 2, "amount": 10.5}
	p := map[string]string{"order_id": "T2", "trade_id": "TX", "status": "2", "amount": "10.5"}
	payload["signature"] = bg.sign(p)
	raw, _ := json.Marshal(payload)
	n, err := bg.Notify(NotifyRequest{Body: raw})
	if err != nil || n.TradeNo != "T2" || n.CallbackNo != "TX" {
		t.Fatalf("notify: %v %+v", err, n)
	}
	payload["status"] = 1
	payload["signature"] = bg.sign(map[string]string{"order_id": "T2", "trade_id": "TX", "status": "1", "amount": "10.5"})
	raw, _ = json.Marshal(payload)
	if _, err := bg.Notify(NotifyRequest{Body: raw}); err == nil {
		t.Error("pending status must not mark order paid")
	}
}

func TestCoinPaymentsIPN(t *testing.T) {
	g, _ := newCoinPayments(map[string]any{"coinpayments_merchant_id": "M", "coinpayments_ipn_secret": "S"})
	p := map[string]string{"merchant": "M", "status": "100", "item_number": "T3", "txn_id": "C1"}
	h := http.Header{}
	h.Set("Hmac", hmacSHA512Hex(encodedSortedQuery(p), "S"))
	n, err := g.Notify(NotifyRequest{Params: p, Headers: h})
	if err != nil || n.TradeNo != "T3" || n.CustomResult != "IPN OK" {
		t.Fatalf("ipn: %v %+v", err, n)
	}
	h.Set("Hmac", "wrong")
	if _, err := g.Notify(NotifyRequest{Params: p, Headers: h}); err == nil {
		t.Error("bad hmac must be rejected")
	}
}

func TestCoinbaseWebhook(t *testing.T) {
	g, _ := newCoinbase(map[string]any{"coinbase_webhook_key": "wk"})
	body := []byte(`{"event":{"id":"EV1","type":"charge:confirmed","data":{"metadata":{"outTradeNo":"T4"}}}}`)
	h := http.Header{}
	h.Set("X-CC-Webhook-Signature", hmacSHA256Hex(string(body), "wk"))
	n, err := g.Notify(NotifyRequest{Body: body, Headers: h})
	if err != nil || n.TradeNo != "T4" || n.CallbackNo != "EV1" {
		t.Fatalf("webhook: %v %+v", err, n)
	}
}

func TestAlipayRSA2RoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(priv)
	pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	g, _ := newAlipayF2F(map[string]any{
		"app_id":      "2021",
		"private_key": base64.StdEncoding.EncodeToString(pkcs8),
		"public_key":  base64.StdEncoding.EncodeToString(pubDER),
	})
	ag := g.(*alipayF2F)

	params := map[string]string{"out_trade_no": "T5", "trade_no": "AL1", "trade_status": "TRADE_SUCCESS", "app_id": "2021"}
	sig, err := signRSA2(rawSortedQuery(params), priv)
	if err != nil {
		t.Fatal(err)
	}
	params["sign"] = sig
	params["sign_type"] = "RSA2"
	n, err := ag.Notify(NotifyRequest{Params: params})
	if err != nil || n.TradeNo != "T5" || n.CallbackNo != "AL1" {
		t.Fatalf("alipay notify: %v %+v", err, n)
	}
	params["out_trade_no"] = "T6"
	if _, err := ag.Notify(NotifyRequest{Params: params}); err == nil {
		t.Error("tampered params must fail verification")
	}
	if strings.Contains(sig, " ") {
		t.Error("signature must be plain base64")
	}
}
