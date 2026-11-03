package api

import (
	"log"

	"github.com/gin-gonic/gin"

	"xboard/internal/resp"
	"xboard/internal/shop"
)

// serverCountries 返回当前可用的国家列表，登录即可访问，不要求有会员。
// 只含国家、可用节点数与最优延迟；不含任何节点地址或凭据。列表每天更新一次（也可由管理员手动更新）。
func (a *App) serverCountries(c *gin.Context) {
	data := a.countriesPayload()
	data["has_plan"] = shop.HasPlan(cur(c)) // 客户端据此提示「开通会员后可连接」
	resp.OK(c, data)
}

// guestServerCountries 与 serverCountries 相同的国家列表，但不需要登录（用于登录/注册前展示）。
// 数据来自每日快照，不含节点地址和凭据；没有 has_plan 字段。
func (a *App) guestServerCountries(c *gin.Context) {
	resp.OK(c, a.countriesPayload())
}

// countriesPayload 把快照整理成对外字段：只暴露国家、可用节点数与最优延迟。
// tags（国家码 → 角标文字）单独下发、不挂在国家项上：快照一天才更新一次，可能是空的或缺新国家，
// 而客户端展示的国家来自订阅，角标不能受快照影响。
func (a *App) countriesPayload() gin.H {
	env := a.jobsEnv()
	snap := env.CountrySnapshotOrBuild()
	type item struct {
		Code          string `json:"code"`
		Name          string `json:"name"`
		Flag          string `json:"flag"`
		Nodes         int    `json:"nodes"`
		BestLatencyMs *int64 `json:"best_latency_ms"`
	}
	list := make([]item, 0, len(snap.Countries))
	for _, e := range snap.Countries {
		list = append(list, item{e.Code, e.Name, e.Flag, e.Nodes, e.BestLatencyMs})
	}
	return gin.H{"updated_at": snap.UpdatedAt, "countries": list, "tags": env.CountryTags()}
}

// countriesStatus 管理端：当前快照、是否正在更新，以及自动更新时间。
func (a *App) countriesStatus(c *gin.Context) {
	env := a.jobsEnv()
	snap, ok := env.LoadCountrySnapshot()
	resp.OK(c, gin.H{
		"snapshot":   snap,
		"exists":     ok,
		"refreshing": env.CountriesRefreshing(),
		"auto_at":    env.CountryRefreshTime(),
	})
}

// countriesRefresh 管理端：手动更新。默认先对全部节点测速再生成快照（异步，可用 countriesStatus 轮询）；
// body.probe 为 false 时只按现有测速结果立即重算。
func (a *App) countriesRefresh(c *gin.Context) {
	m := bodyMap(c)
	probe := true
	if v, ok := m["probe"]; ok {
		if b, isBool := v.(bool); isBool {
			probe = b
		}
	}
	env := a.jobsEnv()
	if env.CountriesRefreshing() {
		resp.OK(c, gin.H{"started": false, "refreshing": true})
		return
	}
	if !probe {
		snap, started := env.RefreshCountriesManual(false)
		resp.OK(c, gin.H{"started": started, "refreshing": false, "snapshot": snap})
		return
	}
	go func() {
		if _, started := env.RefreshCountriesManual(true); !started {
			log.Printf("countriesRefresh: 已有更新在进行，跳过")
		}
	}()
	resp.OK(c, gin.H{"started": true, "refreshing": true})
}
