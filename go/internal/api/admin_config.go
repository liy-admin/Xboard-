package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/resp"
)

// 管理端配置：读取与保存。对应 V2/Admin/ConfigController::fetch / save / testSendMail。
// 配置表 configSections 由 scratchpad/gen_config.pl 从 PHP 生成（config_map.go）。

// configDefaultOverrides 为依赖运行时常量的缺省值（PHP 中为常量表达式或 hash 计算）。
func (a *App) configDefaultOverrides() map[string]any {
	return map[string]any{
		"secure_path": fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(a.AppKey))),
		// 对应 Dict::WITHDRAW_METHOD_WHITELIST_DEFAULT
		"commission_withdraw_method": []any{"支付宝", "USDT", "Paypal"},
		"email_whitelist_suffix": []any{"gmail.com", "qq.com", "163.com", "yahoo.com", "sina.com", "126.com",
			"outlook.com", "yeah.net", "foxmail.com"},
	}
}

// configValue 读取单个配置项并按类型转换；缺失时使用缺省值。
func (a *App) configValue(f cfgField, overrides map[string]any) any {
	def := f.Def
	if v, ok := overrides[f.Key]; ok && def == nil {
		def = v
	}
	if !a.Set.Has(f.Setting) {
		if f.Kind == "bool" { // PHP 里是 (bool) admin_setting(...)：缺省值也要是布尔
			return phpTruthy(def)
		}
		return def
	}
	raw := a.Set.Get(f.Setting, "")
	switch f.Kind {
	case "bool":
		return a.Set.Bool(f.Setting, false)
	case "int":
		return a.Set.Int(f.Setting, 0)
	case "float":
		return a.Set.Float(f.Setting, 0)
	}
	// raw：对应 Support\Setting::load——字符串值只要是合法 JSON 就按 JSON 解码，
	// 所以 "60" 是数字 60、"true" 是布尔、数组以 JSON 存储；解不了的保持原字符串。
	return decodeSettingValue(raw)
}

// decodeSettingValue 复刻 PHP json_decode 对设置值的解码：整数保持为整数，其余数值为浮点。
func decodeSettingValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return raw
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil || dec.More() {
		return raw
	}
	return normalizeJSONNumbers(out)
}

func normalizeJSONNumbers(v any) any {
	switch t := v.(type) {
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return i
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case []any:
		for i := range t {
			t[i] = normalizeJSONNumbers(t[i])
		}
		return t
	case map[string]any:
		for k := range t {
			t[k] = normalizeJSONNumbers(t[k])
		}
		return t
	}
	return v
}

// phpTruthy 按 PHP 的 (bool) 规则判断真假（0、""、"0"、null、空数组为假）。
func phpTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	case string:
		return t != "" && t != "0"
	case []any:
		return len(t) > 0
	}
	return true
}

func (a *App) configFetch(c *gin.Context) {
	overrides := a.configDefaultOverrides()
	key := c.Query("key")
	out := map[string]any{}
	known := false
	for _, sec := range configSections {
		known = known || sec.Name == key
	}
	for _, sec := range configSections {
		if known && sec.Name != key { // 只取单个分组时不必计算其它分组（订阅模板还要查库）
			continue
		}
		if sec.Name == "subscribe_template" {
			tpl := map[string]any{}
			for _, name := range []string{"singbox", "clash", "clashmeta", "stash", "surge", "surfboard"} {
				tpl["subscribe_template_"+name] = a.subscribeTemplateContent(name, name == "singbox")
			}
			out["subscribe_template"] = tpl
			continue
		}
		fields := map[string]any{}
		for _, f := range sec.Fields {
			fields[f.Key] = a.configValue(f, overrides)
		}
		out[sec.Name] = fields
	}
	if key != "" {
		if v, ok := out[key]; ok {
			resp.OK(c, map[string]any{key: v})
			return
		}
	}
	resp.OK(c, out)
}

// subscribeTemplateContent 对应 formatTemplateContent：sing-box 模板美化为 4 空格缩进的 JSON。
func (a *App) subscribeTemplateContent(name string, pretty bool) string {
	var t model.SubscribeTemplate
	if a.S.DB.Where("name = ?", name).First(&t).Error != nil || t.Content == nil {
		return ""
	}
	if pretty {
		var buf bytes.Buffer
		if json.Indent(&buf, []byte(strings.TrimSpace(*t.Content)), "", "    ") == nil {
			return buf.String()
		}
	}
	return *t.Content
}

// configSave 对应 ConfigController::save：订阅模板写入 v2_subscribe_templates，其余写入 v2_settings。
func (a *App) configSave(c *gin.Context) {
	m := bodyMap(c)
	tpl := a.ruleSetURLTemplate()
	if v, ok := m[settingRuleSetURLTemplate]; ok {
		tpl, _ = v.(string)
		tpl = strings.TrimSpace(tpl)
		if err := ValidateRuleSetURLTemplate(tpl); err != nil {
			resp.Fail(c, 422, err.Error())
			return
		}
		m[settingRuleSetURLTemplate] = tpl
	}
	if v, ok := m[settingClientDirectDNS]; ok {
		dns, _ := v.(string)
		dns = strings.TrimSpace(dns)
		if err := ValidateClientDirectDNS(dns); err != nil {
			resp.Fail(c, 422, err.Error())
			return
		}
		m[settingClientDirectDNS] = dns
	}
	if v, ok := m[settingClientRoutingRules]; ok && v != nil {
		raw, isStr := v.(string)
		if !isStr {
			b, _ := json.Marshal(v)
			raw = string(b)
		}
		if _, err := ParseClientRoutingRules(raw, tpl); err != nil {
			resp.Fail(c, 422, err.Error())
			return
		}
	}
	ts := time.Now()
	for k, v := range m {
		if strings.HasPrefix(k, "subscribe_template_") {
			name := strings.TrimPrefix(k, "subscribe_template_")
			content, _ := v.(string)
			var t model.SubscribeTemplate
			if a.S.DB.Where("name = ?", name).First(&t).Error == nil {
				a.S.DB.Model(&t).Updates(map[string]any{"content": content, "updated_at": ts})
			} else {
				a.S.DB.Exec("INSERT INTO v2_subscribe_templates (name, content, created_at, updated_at) VALUES (?, ?, ?, ?)",
					name, content, ts, ts)
			}
			continue
		}
		if err := a.settingUpsert(k, v); err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
	}
	a.Set.Invalidate()
	resp.OK(c, true)
}

// settingUpsert 按 Setting::createOrUpdate 的规则写入：数组/对象存 JSON，布尔 true 为 "1"、false 为空串。
// settingStoredValue 把要保存的值转成 v2_settings.value 的存储形式。
// 任何 slice / map / struct 都必须按 JSON 存：此前只认 []any 与 map[string]any，
// 保存订阅预设时传入的 []map[string]any 会落到 fmt.Sprint，写成 "[map[k:v]]" 这种无法解析的文本。
func settingStoredValue(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case bool:
		if x {
			return "1", nil
		}
		return "", nil
	case string:
		return x, nil
	case json.RawMessage:
		return string(x), nil
	case []byte:
		return string(x), nil
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return string(b), nil
	}
	return fmt.Sprint(v), nil
}

func (a *App) settingUpsert(name string, v any) error {
	stored, err := settingStoredValue(v)
	if err != nil {
		return err
	}
	var row model.Setting
	if a.S.DB.Where("name = ?", name).First(&row).Error == nil {
		return a.S.DB.Model(&row).Update("value", stored).Error
	}
	return a.S.DB.Create(&model.Setting{Name: name, Value: stringPtr(stored)}).Error
}

func stringPtr(v any) *string {
	if v == nil {
		return nil
	}
	s := fmt.Sprint(v)
	return &s
}

// configTestSendMail 对应 testSendMail：向指定邮箱（缺省为当前管理员）发送测试邮件。
func (a *App) configTestSendMail(c *gin.Context) {
	email, _ := mStr(bodyMap(c), "email")
	if email == "" {
		email = cur(c).Email
	}
	app := a.Set.Get("app_name", "Lets VPN")
	vars := map[string]any{"name": app, "content": "This is xboard test email", "url": a.Set.Get("app_url", "")}
	err := a.jobsEnv().SendTemplateMail(email, "This is xboard test email", "notify", vars)
	if err != nil {
		c.JSON(200, gin.H{"data": gin.H{"error": err.Error(), "hint": "发送失败，请检查邮件配置"}})
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"error": nil, "hint": nil}})
}
