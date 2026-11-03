package shop

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// 对应 plugins-core/AlipayF2f：支付宝当面付（alipay.trade.precreate，RSA2 签名）。

func init() {
	RegisterGateway("AlipayF2F", newAlipayF2F)
}

const alipayGateway = "https://openapi.alipay.com/gateway.do"

type alipayF2F struct{ cfg map[string]any }

func newAlipayF2F(cfg map[string]any) (Gateway, error) { return &alipayF2F{cfg}, nil }

// stripPEM 去掉 PEM 头尾与空白，只保留 base64 正文；兼容单行与多行两种粘贴方式。
func stripPEM(s, header string) string {
	s = strings.ReplaceAll(s, header, "")
	for _, h := range []string{"-----BEGIN PUBLIC KEY-----", "-----END PUBLIC KEY-----",
		"-----BEGIN RSA PRIVATE KEY-----", "-----END RSA PRIVATE KEY-----",
		"-----BEGIN PRIVATE KEY-----", "-----END PRIVATE KEY-----"} {
		s = strings.ReplaceAll(s, h, "")
	}
	return strings.Join(strings.Fields(s), "")
}

// parsePrivateKey 支持 PKCS#8 与 PKCS#1 两种私钥格式。
func parsePrivateKey(raw string) (*rsa.PrivateKey, error) {
	der, err := base64.StdEncoding.DecodeString(stripPEM(raw, ""))
	if err != nil {
		return nil, fmt.Errorf("支付宝私钥格式错误: %w", err)
	}
	if k, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, errors.New("支付宝私钥不是 RSA 类型")
	}
	return x509.ParsePKCS1PrivateKey(der)
}

// parsePublicKey 解析支付宝公钥（PKIX / SubjectPublicKeyInfo）。
func parsePublicKey(raw string) (*rsa.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(stripPEM(raw, ""))
	if err != nil {
		return nil, fmt.Errorf("支付宝公钥格式错误: %w", err)
	}
	k, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	rk, ok := k.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("支付宝公钥不是 RSA 类型")
	}
	return rk, nil
}

// signRSA2 对应 AlipayF2F::buildSign（SHA256WithRSA，结果 base64）。
func signRSA2(data string, key *rsa.PrivateKey) (string, error) {
	h := sha256.Sum256([]byte(data))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, h[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func (g *alipayF2F) Pay(r PayRequest) (PayResult, error) {
	priv, err := parsePrivateKey(cfgStr(g.cfg, "private_key"))
	if err != nil {
		return PayResult{}, err
	}
	subject := r.AppName + " - 订阅"
	if v, ok := g.cfg["product_name"]; ok && v != nil {
		subject = cfgStr(g.cfg, "product_name")
	}
	biz, _ := json.Marshal(struct {
		Subject    string `json:"subject"`
		OutTradeNo string `json:"out_trade_no"`
		Total      string `json:"total_amount"`
	}{subject, r.TradeNo, yuan(r.TotalAmount)})

	params := map[string]string{
		"app_id":         cfgStr(g.cfg, "app_id"),
		"method":         "alipay.trade.precreate",
		"charset":        "UTF-8",
		"sign_type":      "RSA2",
		"timestamp":      newestTimestamp(),
		"biz_content":    string(biz),
		"version":        "1.0",
		"_input_charset": "UTF-8",
	}
	if r.NotifyURL != "" {
		params["notify_url"] = r.NotifyURL
	}
	sig, err := signRSA2(rawSortedQuery(params), priv)
	if err != nil {
		return PayResult{}, err
	}
	params["sign"] = sig

	resp, err := httpClient.Get(alipayGateway + "?" + encodedSortedQuery(params))
	if err != nil {
		return PayResult{}, fmt.Errorf("从支付宝请求失败: %w", err)
	}
	defer resp.Body.Close()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return PayResult{}, errors.New("从支付宝请求失败")
	}
	raw, ok := body["alipay_trade_precreate_response"]
	if !ok {
		return PayResult{}, errors.New("从支付宝请求失败")
	}
	var res struct {
		Msg    string `json:"msg"`
		SubMsg string `json:"sub_msg"`
		QRCode string `json:"qr_code"`
	}
	_ = json.Unmarshal(raw, &res)
	if res.Msg != "Success" {
		return PayResult{}, errors.New(res.SubMsg)
	}
	if res.QRCode == "" {
		return PayResult{}, errors.New("获取付款二维码失败")
	}
	return PayResult{Type: 0, Data: res.QRCode}, nil
}

func (g *alipayF2F) Notify(req NotifyRequest) (*Notification, error) {
	p := copyMap(req.Params)
	if p["trade_status"] != "TRADE_SUCCESS" {
		return nil, errors.New("trade status is not success")
	}
	pub, err := parsePublicKey(cfgStr(g.cfg, "public_key"))
	if err != nil {
		return nil, err
	}
	sig, err := base64.StdEncoding.DecodeString(p["sign"])
	if err != nil {
		return nil, errSignature
	}
	delete(p, "sign")
	delete(p, "sign_type")
	h := sha256.Sum256([]byte(rawSortedQuery(p)))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig); err != nil {
		return nil, errSignature
	}
	return &Notification{TradeNo: p["out_trade_no"], CallbackNo: p["trade_no"]}, nil
}
