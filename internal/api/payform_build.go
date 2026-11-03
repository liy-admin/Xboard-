package api

import (
	"bytes"
	"encoding/json"
	"sort"

	"github.com/gin-gonic/gin"
)

// orderedPaymentForm 以字段定义顺序输出 JSON 对象。
// Go 的 map 会按键名字母序序列化，而 PHP 的表单顺序（如 EPay：网关地址、商户ID、通信密钥、支付类型）
// 是插件里定义的顺序，前端按对象键顺序渲染，所以必须保持。
type orderedPaymentForm []orderedPaymentField

type orderedPaymentField struct {
	Key   string
	Value gin.H
}

func (o orderedPaymentForm) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, f := range o {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, _ := json.Marshal(f.Key)
		v, err := json.Marshal(f.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// buildPaymentForm 对应 PaymentService::form()：每个字段包含 type / label / placeholder / description / value / options，
// value 取已保存配置，缺失时为空串。
func buildPaymentForm(fields []payFormField, stored map[string]any) orderedPaymentForm {
	out := make(orderedPaymentForm, 0, len(fields))
	for _, f := range fields {
		var value any = ""
		if v, ok := stored[f.Key]; ok && v != nil {
			value = v
		}
		out = append(out, orderedPaymentField{Key: f.Key, Value: gin.H{
			"label": f.Label, "type": f.Type, "required": f.Required, "description": f.Description,
			"placeholder": "", "value": value, "options": []any{},
		}})
	}
	return out
}

// sortedNames 返回排序后的副本，避免依赖 map 遍历顺序（下拉选项每次刷新顺序都会变）。
func sortedNames(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
