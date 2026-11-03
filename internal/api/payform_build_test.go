package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPaymentFormFillsStoredValuesInPluginOrder(t *testing.T) {
	stored := map[string]any{"url": "https://pay.example", "pid": "1001", "key": "secret", "type": "alipay"}
	form := buildPaymentForm(payGatewayForms["Epay"], stored)
	raw, err := json.Marshal(form)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)

	// 顺序必须是插件里定义的顺序：url、pid、key、type（而不是字母序 key、pid、type、url）
	idx := func(k string) int { return strings.Index(s, `"`+k+`":{`) }
	if !(idx("url") < idx("pid") && idx("pid") < idx("key") && idx("key") < idx("type") && idx("url") >= 0) {
		t.Fatalf("field order must follow the plugin definition: %s", s)
	}

	var back map[string]map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back["key"]["value"] != "secret" || back["pid"]["value"] != "1001" || back["url"]["value"] != "https://pay.example" {
		t.Fatalf("stored values must be returned so the edit form is not blank: %v", back)
	}
	for _, k := range []string{"label", "type", "required", "description", "placeholder", "value", "options"} {
		if _, ok := back["url"][k]; !ok {
			t.Errorf("field is missing %q (the admin UI reads it)", k)
		}
	}
}

func TestBuildPaymentFormDefaultsToEmptyString(t *testing.T) {
	raw, _ := json.Marshal(buildPaymentForm(payGatewayForms["Epay"], map[string]any{"url": "https://x"}))
	var back map[string]map[string]any
	_ = json.Unmarshal(raw, &back)
	if back["pid"]["value"] != "" {
		t.Fatalf("unset field must be an empty string, got %v", back["pid"]["value"])
	}
}

func TestPaymentSaveFieldsRequiredAndValidation(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{"name": "支付宝", "payment": "EPay", "config": map[string]any{"url": "https://x"}}
	}
	if _, msg := paymentSaveFields(map[string]any{"payment": "EPay", "config": map[string]any{}}); msg != "显示名称不能为空" {
		t.Errorf("name: %q", msg)
	}
	if _, msg := paymentSaveFields(map[string]any{"name": "n", "config": map[string]any{}}); msg != "网关参数不能为空" {
		t.Errorf("payment: %q", msg)
	}
	if _, msg := paymentSaveFields(map[string]any{"name": "n", "payment": "EPay"}); msg != "配置参数不能为空" {
		t.Errorf("config: %q", msg)
	}
	m := base()
	m["notify_domain"] = "pay.example.com"
	if _, msg := paymentSaveFields(m); msg != "自定义通知域名格式有误" {
		t.Errorf("domain: %q", msg)
	}
	m = base()
	m["handling_fee_percent"] = 101.0
	if _, msg := paymentSaveFields(m); msg != "百分比手续费范围须在0-100之间" {
		t.Errorf("percent: %q", msg)
	}
}

func TestPaymentSaveFieldsCanClearOptionalColumns(t *testing.T) {
	// 后台把通知域名、手续费清空后保存，必须写成 NULL（之前被忽略，旧值一直留着）
	m := map[string]any{
		"name": "n", "payment": "EPay", "config": map[string]any{"url": "https://x"},
		"icon": "", "notify_domain": "", "handling_fee_fixed": nil, "handling_fee_percent": nil,
	}
	upd, msg := paymentSaveFields(m)
	if msg != "" {
		t.Fatal(msg)
	}
	for _, k := range []string{"icon", "notify_domain", "handling_fee_fixed", "handling_fee_percent"} {
		v, present := upd[k]
		if !present || v != nil {
			t.Errorf("%s must be present and nil to clear the column, got present=%v value=%v", k, present, v)
		}
	}
}

func TestPaymentSaveFieldsKeepsValuesAndLeavesAbsentKeysAlone(t *testing.T) {
	m := map[string]any{
		"name": "n", "payment": "EPay", "config": map[string]any{"url": "https://x"},
		"notify_domain": "https://pay.example.com", "handling_fee_percent": 12.0, "handling_fee_fixed": 300.0,
	}
	upd, msg := paymentSaveFields(m)
	if msg != "" {
		t.Fatal(msg)
	}
	if upd["notify_domain"] != "https://pay.example.com" || upd["handling_fee_percent"] != 12.0 || upd["handling_fee_fixed"] != int64(300) {
		t.Fatalf("values must be kept: %v", upd)
	}
	// 请求里没带的键不应出现在更新里（不能误清空）
	delete(m, "notify_domain")
	upd, _ = paymentSaveFields(m)
	if _, present := upd["notify_domain"]; present {
		t.Fatal("absent keys must not be touched")
	}
}

func TestSortedNamesIsStable(t *testing.T) {
	in := []string{"MGate", "EPay", "Coinbase", "AlipayF2F"}
	got := sortedNames(in)
	want := []string{"AlipayF2F", "Coinbase", "EPay", "MGate"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v", got)
	}
	if in[0] != "MGate" {
		t.Fatal("input must not be modified")
	}
}
