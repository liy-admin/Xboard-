package api

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 管理端仪表盘统计。对应 V2/Admin/StatController::getStats、getStatRecord、getOrder。
// 节点在线判断：Redis 中 SERVER_<TYPE>_LAST_CHECK_AT_<id> 在 5 分钟内更新或 node_ws_alive:<id> 存在（见 PROGRESS.md 差异）。

// nodeOnline 判断节点是否在线。
func (a *App) nodeOnline(s model.Server) bool {
	ctx := bgCtx()
	if a.S.Redis.Exists(ctx, fmt.Sprintf("node_ws_alive:%d", s.ID)).Val() > 0 {
		return true
	}
	t, err := a.S.Redis.Get(ctx, fmt.Sprintf("SERVER_%s_LAST_CHECK_AT_%d", strings.ToUpper(s.Type), s.ID)).Int64()
	return err == nil && now()-t <= 300
}

// growth 对应 round((cur-prev)/prev*100, 1)，prev 为 0 时返回 0。
func growth(cur, prev int64) float64 {
	if prev <= 0 {
		return 0
	}
	return math.Round(float64(cur-prev)/float64(prev)*1000) / 10
}

func (a *App) sumOrders(from, to int64) int64 {
	var n int64
	a.S.DB.Model(&model.Order{}).Where("created_at >= ? AND created_at < ? AND status NOT IN (0, 2)", from, to).
		Select("COALESCE(SUM(total_amount),0)").Scan(&n)
	return n
}

func (a *App) sumCommission(from, to int64) int64 {
	var n int64
	a.S.DB.Model(&model.CommissionLog{}).Where("created_at >= ? AND created_at < ?", from, to).
		Select("COALESCE(SUM(get_amount),0)").Scan(&n)
	return n
}

func (a *App) countUsers(from, to int64) int64 {
	var n int64
	a.S.DB.Model(&model.User{}).Where("created_at >= ? AND created_at < ?", from, to).Count(&n)
	return n
}

// trafficSum 统计节点流量在时间区间内的上下行合计。
func (a *App) trafficSum(from, to int64) gin.H {
	var r struct {
		Upload, Download, Total int64
	}
	q := a.S.DB.Model(&model.StatServer{})
	if to > 0 {
		q = q.Where("record_at >= ? AND record_at < ?", from, to)
	} else {
		q = q.Where("record_at >= ?", from)
	}
	q.Select("COALESCE(SUM(u),0) AS upload, COALESCE(SUM(d),0) AS download, COALESCE(SUM(u + d),0) AS total").Scan(&r)
	return gin.H{"upload": r.Upload, "download": r.Download, "total": r.Total}
}

// adminStats 对应 getStats。
func (a *App) adminStats(c *gin.Context) {
	y, m, d := time.Now().Date()
	monthStart := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).Unix()
	lastMonthStart := time.Date(y, m-1, 1, 0, 0, 0, 0, time.Local).Unix()
	twoMonthsStart := time.Date(y, m-2, 1, 0, 0, 0, 0, time.Local).Unix()
	todayStart := time.Date(y, m, d, 0, 0, 0, 0, time.Local).Unix()
	yesterdayStart := todayStart - 86400
	nowTs := now()

	var nodes []model.Server
	a.S.DB.Find(&nodes)
	onlineNodes := 0
	for _, s := range nodes {
		if a.nodeOnline(s) {
			onlineNodes++
		}
	}
	var onlineDevices, onlineUsers int64
	a.S.DB.Model(&model.User{}).Where("t >= ?", nowTs-600).Select("COALESCE(SUM(online_count),0)").Scan(&onlineDevices)
	a.S.DB.Model(&model.User{}).Where("t >= ?", nowTs-600).Count(&onlineUsers)

	todayIncome := a.sumOrders(todayStart, nowTs)
	yesterdayIncome := a.sumOrders(yesterdayStart, todayStart)
	curMonthIncome := a.sumOrders(monthStart, nowTs)
	lastMonthIncome := a.sumOrders(lastMonthStart, monthStart)
	twoMonthsIncome := a.sumOrders(twoMonthsStart, lastMonthStart)
	lastMonthComm := a.sumCommission(lastMonthStart, monthStart)
	curMonthComm := a.sumCommission(monthStart, nowTs)
	twoMonthsComm := a.sumCommission(twoMonthsStart, lastMonthStart)
	curMonthUsers := a.countUsers(monthStart, nowTs)
	lastMonthUsers := a.countUsers(lastMonthStart, monthStart)

	var totalUsers, activeUsers, ticketPending, commissionPending int64
	a.S.DB.Model(&model.User{}).Count(&totalUsers)
	a.S.DB.Model(&model.User{}).Where("expired_at >= ? OR expired_at IS NULL", nowTs).Count(&activeUsers)
	a.S.DB.Model(&model.Ticket{}).Where("status = 0").Count(&ticketPending)
	a.S.DB.Model(&model.Order{}).Where("commission_status = 0 AND invite_user_id IS NOT NULL AND status = 3 AND commission_balance > 0").
		Count(&commissionPending)

	resp.OK(c, gin.H{
		"todayIncome": todayIncome, "dayIncomeGrowth": growth(todayIncome, yesterdayIncome),
		"currentMonthIncome": curMonthIncome, "lastMonthIncome": lastMonthIncome,
		"monthIncomeGrowth": growth(curMonthIncome, lastMonthIncome), "lastMonthIncomeGrowth": growth(lastMonthIncome, twoMonthsIncome),
		"currentMonthCommissionPayout": curMonthComm, "lastMonthCommissionPayout": lastMonthComm,
		"commissionGrowth": growth(lastMonthComm, twoMonthsComm), "commissionPendingTotal": commissionPending,
		"currentMonthNewUsers": curMonthUsers, "totalUsers": totalUsers, "activeUsers": activeUsers,
		"userGrowth": growth(curMonthUsers, lastMonthUsers), "onlineUsers": onlineUsers, "onlineDevices": onlineDevices,
		"ticketPendingTotal": ticketPending, "onlineNodes": onlineNodes,
		"todayTraffic": a.trafficSum(todayStart, 0), "monthTraffic": a.trafficSum(monthStart, 0),
		"totalTraffic": a.trafficSum(0, 0),
	})
}

// adminStatRecord 对应 getStatRecord：按日返回 v2_stat 中的指定字段（金额类除以 100）。
func (a *App) adminStatRecord(c *gin.Context) {
	start, end := statRange(c)
	var rows []model.Stat
	q := a.S.DB.Where("record_at >= ? AND record_at < ?", start, end).Order("record_at ASC")
	typ := c.Query("type")
	switch typ {
	case "paid_total", "commission_total":
		q.Find(&rows)
		out := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			v := r.PaidTotal
			if typ == "commission_total" {
				v = r.CommissionTotal
			}
			out = append(out, gin.H{"record_at": r.RecordAt, typ: float64(v) / 100, "record_type": r.RecordType})
		}
		c.JSON(200, gin.H{"data": out})
	case "register_count":
		q.Find(&rows)
		c.JSON(200, gin.H{"data": rows})
	default:
		c.JSON(200, gin.H{"data": []any{}})
	}
}

// statRange 解析 start_date/end_date（YYYY-MM-DD，闭区间），缺省为最近 30 天。
func statRange(c *gin.Context) (int64, int64) {
	loc := time.Local
	end := time.Now().In(loc)
	start := end.AddDate(0, 0, -30)
	if s, err := time.ParseInLocation("2006-01-02", c.Query("start_date"), loc); err == nil {
		start = s
	}
	if e, err := time.ParseInLocation("2006-01-02", c.Query("end_date"), loc); err == nil {
		end = e.AddDate(0, 0, 1)
	}
	return start.Unix(), end.Unix()
}

// statTypeLabel 对应 StatController::getTypeLabel。
var statTypeLabel = map[string]string{
	"paid_total": "收款金额", "paid_count": "收款笔数",
	"commission_total": "佣金金额(已发放)", "commission_count": "佣金笔数(已发放)",
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// adminStatOrder 对应 getOrder：按日返回订单统计列表与区间汇总。
// 金额字段保持分为单位（前端自行换算），响应外层带 code/message，与 PHP 一致。
func (a *App) adminStatOrder(c *gin.Context) {
	startDate, endDate, typ := c.Query("start_date"), c.Query("end_date"), c.Query("type")
	q := a.S.DB.Model(&model.Stat{}).Where("record_type = ?", "d")
	if startDate != "" {
		t, err := time.ParseInLocation("2006-01-02", startDate, shop.Loc)
		if err != nil {
			resp.Fail(c, 422, "start_date 格式应为 Y-m-d")
			return
		}
		q = q.Where("record_at >= ?", t.Unix())
	}
	if endDate != "" {
		t, err := time.ParseInLocation("2006-01-02", endDate, shop.Loc)
		if err != nil {
			resp.Fail(c, 422, "end_date 格式应为 Y-m-d")
			return
		}
		q = q.Where("record_at <= ?", t.Unix()+86399)
	}
	if typ != "" {
		if _, ok := statTypeLabel[typ]; !ok {
			resp.Fail(c, 422, "type 参数有误")
			return
		}
	}
	var rows []model.Stat
	q.Order("record_at DESC").Find(&rows)

	day := func(ts int64) string { return time.Unix(ts, 0).In(shop.Loc).Format("2006-01-02") }
	first, last := int64(0), int64(0) // PHP 的 first()/last()：DESC 排序下最新与最早
	if len(rows) > 0 {
		first, last = rows[0].RecordAt, rows[len(rows)-1].RecordAt
	}
	sum := struct{ paidTotal, paidCount, commTotal, commCount int64 }{}
	list := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		sum.paidTotal += r.PaidTotal
		sum.paidCount += r.PaidCount
		sum.commTotal += r.CommissionTotal
		sum.commCount += r.CommissionCount
		if typ != "" {
			var v int64
			switch typ {
			case "paid_total":
				v = r.PaidTotal
			case "paid_count":
				v = r.PaidCount
			case "commission_total":
				v = r.CommissionTotal
			case "commission_count":
				v = r.CommissionCount
			}
			list = append(list, gin.H{"date": day(r.RecordAt), "value": v, "type": statTypeLabel[typ]})
			continue
		}
		avgOrder, avgComm := 0.0, 0.0
		if r.PaidCount > 0 {
			avgOrder = round2(float64(r.PaidTotal) / float64(r.PaidCount))
		}
		if r.CommissionCount > 0 {
			avgComm = round2(float64(r.CommissionTotal) / float64(r.CommissionCount))
		}
		list = append(list, gin.H{
			"date": day(r.RecordAt), "paid_total": r.PaidTotal, "paid_count": r.PaidCount,
			"commission_total": r.CommissionTotal, "commission_count": r.CommissionCount,
			"avg_order_amount": avgOrder, "avg_commission_amount": avgComm,
		})
	}
	// 列表按时间正序返回（PHP 的 array_reverse）。
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	summary := gin.H{
		"paid_total": sum.paidTotal, "paid_count": sum.paidCount,
		"commission_total": sum.commTotal, "commission_count": sum.commCount,
		"start_date": orStr(startDate, day(last)), "end_date": orStr(endDate, day(first)),
		"avg_paid_amount": 0.0, "avg_commission_amount": 0.0, "commission_rate": 0.0,
	}
	if sum.paidCount > 0 {
		summary["avg_paid_amount"] = round2(float64(sum.paidTotal) / float64(sum.paidCount))
	}
	if sum.commCount > 0 {
		summary["avg_commission_amount"] = round2(float64(sum.commTotal) / float64(sum.commCount))
	}
	if sum.paidTotal > 0 {
		summary["commission_rate"] = round2(float64(sum.commTotal) / float64(sum.paidTotal) * 100)
	}
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": gin.H{"list": list, "summary": summary}})
}

func orStr(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
