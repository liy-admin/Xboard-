package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 管理端支付方式。对应 V2/Admin/PaymentController：fetch、save、drop、show（启停）、sort、getPaymentMethods。
// 未包含：getPaymentForm（网关表单定义在 PHP 插件中，Go 侧尚未提供）、uploadIcon。

func (a *App) paymentNotifyURL(p model.Payment) string {
	raw := a.subscribeRoot() + fmt.Sprintf("/api/v1/guest/payment/notify/%s/%s", p.Payment, p.UUID)
	if p.NotifyDomain != nil && *p.NotifyDomain != "" {
		if u, err := url.Parse(raw); err == nil {
			return strings.TrimRight(*p.NotifyDomain, "/") + u.Path
		}
	}
	return raw
}

// subscribeRoot 对应 url('/')：后台 app_url，未配置时为空（相对路径）。
func (a *App) subscribeRoot() string {
	return strings.TrimRight(a.Set.Get("app_url", ""), "/")
}

func (a *App) paymentFetch(c *gin.Context) {
	var rows []model.Payment
	a.S.DB.Order("sort ASC").Find(&rows)
	out := make([]gin.H, 0, len(rows))
	for _, p := range rows {
		out = append(out, gin.H{
			"id": p.ID, "uuid": p.UUID, "name": p.Name, "payment": p.Payment, "icon": p.Icon,
			"config": rawJSONOrString(p.Config), "enable": p.Enable, "handling_fee_fixed": p.HandlingFeeFixed,
			"handling_fee_percent": p.HandlingFeePercent, "notify_domain": p.NotifyDomain, "sort": p.Sort,
			"notify_url": a.paymentNotifyURL(p),
		})
	}
	resp.OK(c, out)
}

// rawJSONOrString 把数据库中的配置字符串尽量还原为 JSON 对象，失败则原样返回。
func rawJSONOrString(s string) any {
	var v any
	if json.Unmarshal([]byte(s), &v) == nil {
		return v
	}
	return s
}

func (a *App) paymentSave(c *gin.Context) {
	if a.subscribeRoot() == "" {
		resp.Fail(c, 400, "请在站点配置中配置站点地址")
		return
	}
	m := bodyMap(c)
	upd, msg := paymentSaveFields(m)
	if msg != "" {
		resp.Fail(c, 422, msg)
		return
	}
	if id, ok := mInt(m, "id"); ok && id > 0 {
		var p model.Payment
		if err := a.S.DB.First(&p, id).Error; err != nil {
			resp.Fail(c, 400, "支付方式不存在")
			return
		}
		upd["updated_at"] = now()
		if err := a.S.DB.Model(&p).Updates(upd).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		resp.OK(c, true)
		return
	}
	upd["uuid"] = randomChars(8)
	upd["created_at"], upd["updated_at"] = now(), now()
	if err := a.S.DB.Model(&model.Payment{}).Create(upd).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) paymentDrop(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	res := a.S.DB.Delete(&model.Payment{}, id)
	if res.RowsAffected == 0 {
		resp.Fail(c, 400, "支付方式不存在")
		return
	}
	resp.OK(c, true)
}

func (a *App) paymentShow(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var p model.Payment
	if err := a.S.DB.First(&p, id).Error; err != nil {
		resp.Fail(c, 400, "支付方式不存在")
		return
	}
	if err := a.S.DB.Model(&p).Update("enable", !p.Enable).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) paymentSort(c *gin.Context) {
	var ids []any
	if err := json.NewDecoder(c.Request.Body).Decode(&struct {
		IDs *[]any `json:"ids"`
	}{&ids}); err != nil || ids == nil {
		resp.Fail(c, 422, "参数有误")
		return
	}
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		for i, v := range ids {
			id, ok := toNum(v)
			if !ok {
				return fmt.Errorf("bad id")
			}
			if err := tx.Model(&model.Payment{}).Where("id = ?", int64(id)).Update("sort", i+1).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) paymentMethods(c *gin.Context) {
	resp.OK(c, sortedNames(shop.GatewayNames()))
}

// paymentSaveFields 校验并生成要写入的字段，对应 PaymentController::save 的 validate。
//
// 可选项（图标、通知域名、固定/百分比手续费）在请求里出现且为空时写成 NULL：
// PHP 的 ConvertEmptyStringsToNull 会把空串变成 null，再由 update() 清空对应列。
// 之前 Go 版会忽略空值，导致后台清空「通知域名」「手续费」后保存不生效。
func paymentSaveFields(m map[string]any) (map[string]any, string) {
	name, _ := mStr(m, "name")
	payment, _ := mStr(m, "payment")
	if name == "" {
		return nil, "显示名称不能为空"
	}
	if payment == "" {
		return nil, "网关参数不能为空"
	}
	cfg, ok := m["config"]
	if !ok || cfg == nil {
		return nil, "配置参数不能为空"
	}
	cfgJSON, _ := json.Marshal(cfg)
	upd := map[string]any{"name": name, "payment": payment, "config": string(cfgJSON)}

	if _, present := m["icon"]; present {
		if icon, _ := mStr(m, "icon"); icon != "" {
			upd["icon"] = icon
		} else {
			upd["icon"] = nil
		}
	}
	if _, present := m["notify_domain"]; present {
		nd, _ := mStr(m, "notify_domain")
		switch {
		case nd == "":
			upd["notify_domain"] = nil
		case !strings.HasPrefix(nd, "http://") && !strings.HasPrefix(nd, "https://"):
			return nil, "自定义通知域名格式有误"
		default:
			upd["notify_domain"] = nd
		}
	}
	if _, present := m["handling_fee_fixed"]; present {
		if v, ok := mInt(m, "handling_fee_fixed"); ok {
			upd["handling_fee_fixed"] = v
		} else {
			upd["handling_fee_fixed"] = nil
		}
	}
	if v, present := m["handling_fee_percent"]; present {
		if f, ok := toNum(v); ok {
			if f < 0 || f > 100 {
				return nil, "百分比手续费范围须在0-100之间"
			}
			upd["handling_fee_percent"] = f
		} else {
			upd["handling_fee_percent"] = nil
		}
	}
	return upd, ""
}
