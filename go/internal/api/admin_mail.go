package api

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/notify"
	"xboard/internal/resp"
	"xboard/internal/sub"
)

// 管理端群发邮件与用户导出。对应 V2/Admin/UserController::sendMail、dumpCSV。
// scope：all（忽略过滤）、filtered（使用 filter/sort）、selected（使用 user_ids）。

// adminUserScope 按 scope 返回用户查询。
func (a *App) adminUserScope(c *gin.Context, scope string, ids []int64) (*gorm.DB, error) {
	q := a.S.DB.Model(&model.User{})
	switch scope {
	case "selected":
		if len(ids) == 0 {
			return nil, fmt.Errorf("user_ids不能为空")
		}
		q = q.Where("id IN ?", ids)
	case "filtered":
		// 被静默忽略的筛选项会让「按筛选结果操作」变成「对全部用户操作」，所以必须拒绝，而不是放行。
		if err := checkFilters(c, adminUserFilter); err != nil {
			return nil, err
		}
		if len(requestFilters(c)) == 0 {
			return nil, fmt.Errorf("筛选条件为空，请先设置筛选条件，或改用「全部」/「已选」")
		}
		q = adminFilters(c, adminUserFilter, q)
	}
	return q, nil
}

func idList(v any) []int64 {
	arr, _ := v.([]any)
	out := make([]int64, 0, len(arr))
	for _, x := range arr {
		if f, ok := toNum(x); ok {
			out = append(out, int64(f))
		}
	}
	return out
}

func (a *App) adminSendMail(c *gin.Context) {
	m := bodyMap(c)
	scope, _ := mStr(m, "scope")
	q, err := a.adminUserScope(c, scope, idList(m["user_ids"]))
	if err != nil {
		resp.Fail(c, 422, err.Error())
		return
	}
	subject, _ := mStr(m, "subject")
	content, _ := mStr(m, "content")
	if subject == "" || content == "" {
		resp.Fail(c, 422, "参数有误")
		return
	}
	app := a.Set.Get("app_name", "Lets VPN")
	appURL := a.Set.Get("app_url", "")
	var users []model.User
	q.Order("id DESC").Find(&users)
	// 群发经邮件队列异步投递，与 PHP 的队列投递一致。
	mq := a.mailQ()
	for _, u := range users {
		vars := userMailVars(u, a.planNameOf(u), app, appURL)
		subj := notify.RenderPlaceholdersStr(subject, vars)
		body := notify.RenderPlaceholdersStr(content, vars)
		values := map[string]any{"name": app, "url": appURL, "content": body, "content_mode": "text"}
		mq.Enqueue(jobs.MailTask{To: u.Email, Subject: subj, Template: "notify", Values: values})
	}
	resp.OK(c, true)
}

// userMailVars 对应群发邮件中的变量表（user.* 与 app.*）。
func userMailVars(u model.User, planName, app, appURL string) map[string]string {
	exp := ""
	if u.ExpiredAt != nil {
		exp = time.Unix(*u.ExpiredAt, 0).Format("2006-01-02 15:04:05")
	}
	used := u.U + u.D
	return map[string]string{
		"app.name": app, "app.url": appURL, "now": time.Now().Format("2006-01-02 15:04:05"),
		"user.id": strconv.FormatInt(u.ID, 10), "user.email": u.Email, "user.uuid": u.UUID,
		"user.plan_name": planName, "user.expired_at": exp,
		"user.transfer_enable": strconv.FormatInt(u.TransferEnab, 10),
		"user.transfer_used":   strconv.FormatInt(used, 10),
		"user.transfer_left":   transferLeftText(u, used),
	}
}

// adminDumpCSV 对应 dumpCSV：带 BOM 的 UTF-8 CSV，列与 PHP 一致。
func (a *App) adminDumpCSV(c *gin.Context) {
	m := requestParams(c)
	scope, _ := m["scope"].(string)
	ids := idList(m["user_ids"])
	q, err := a.adminUserScope(c, scope, ids)
	if err != nil {
		resp.Fail(c, 422, err.Error())
		return
	}
	var users []model.User
	q.Order("id ASC").Find(&users)
	var b strings.Builder
	b.WriteString("FEFF")
	w := csv.NewWriter(&b)
	_ = w.Write([]string{"邮箱", "余额", "推广佣金", "总流量", "剩余流量", "套餐到期时间", "订阅计划", "订阅地址"})
	for _, u := range users {
		plan := a.planNameOf(u)
		if plan == "" {
			plan = "无订阅"
		}
		exp := "长期有效"
		if u.ExpiredAt != nil {
			exp = time.Unix(*u.ExpiredAt, 0).Format("2006-01-02 15:04:05")
		}
		_ = w.Write([]string{
			u.Email,
			formatMoney(float64(u.Balance) / 100),
			formatMoney(float64(u.CommissionBal) / 100),
			trafficTotalText(u),
			trafficLeftText(u),
			exp, plan, a.subscribeURL(u.Token, ""),
		})
	}
	w.Flush()
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="users_%s.csv"`, time.Now().Format("2006-01-02_150405")))
	c.Data(200, "text/csv; charset=utf-8", []byte(b.String()))
}

// formatMoney 对应 number_format($v, 2)：千分位分隔、两位小数。
func formatMoney(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	intPart, frac, _ := strings.Cut(s, ".")
	var out []byte
	for i, ch := range []byte(intPart) {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, ch)
	}
	res := string(out) + "." + frac
	if neg {
		res = "-" + res
	}
	return res
}

// planNameOf 返回用户当前套餐名称，无套餐时为空串。
func (a *App) planNameOf(u model.User) string {
	if u.PlanID == nil {
		return ""
	}
	var p model.Plan
	if a.S.DB.Select("id", "name").First(&p, *u.PlanID).Error != nil {
		return ""
	}
	return p.Name
}

// 流量上限为 0 视为无限流量：邮件占位符与导出表里显示「无限」，而不是 0 或负数。
func transferLeftText(u model.User, used int64) string {
	if u.TransferEnab == 0 {
		return "无限"
	}
	return strconv.FormatInt(u.TransferEnab-used, 10)
}

func trafficTotalText(u model.User) string {
	if u.TransferEnab == 0 {
		return "无限"
	}
	return sub.TrafficConvert(float64(u.TransferEnab))
}

func trafficLeftText(u model.User) string {
	if u.TransferEnab == 0 {
		return "无限"
	}
	return sub.TrafficConvert(float64(u.TransferEnab - (u.U + u.D)))
}
