package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
	"xboard/internal/sub"
)

// 管理端节点管理的其余动作：测速、复制、流量清零、批量操作；订单 update/assign；
// 统计扩展（overview、排行、用户流量）；礼品卡导出与统计；系统状态。

func idsFrom(m map[string]any) []int64 { return idList(m["ids"]) }

// ---- 节点：测速 ----

// probeLatency 对应 ManageController::probeLatency：后台异步测速，同时只允许一个任务运行。
func (a *App) probeLatency(c *gin.Context) {
	m := bodyMap(c)
	ids := idsFrom(m)
	env := a.jobsEnv()
	if st, ok := env.LatencyStatusOf(); ok && st.Running && time.Now().Unix()-st.UpdatedAt < 120 {
		c.JSON(200, gin.H{"data": gin.H{"started": false, "running": true}})
		return
	}
	var total int64
	q := a.S.DB.Model(&model.Server{}).Where("parent_id IS NULL")
	if len(ids) > 0 {
		q = q.Where("id IN ?", ids)
	}
	q.Count(&total)
	go func() {
		if err := env.CheckServerLatency(ids); err != nil {
			log.Printf("probeLatency: %v", err)
		}
	}()
	resp.OK(c, gin.H{"started": true, "total": total})
}

func (a *App) probeStatus(c *gin.Context) {
	st, ok := a.jobsEnv().LatencyStatusOf()
	if !ok {
		st = jobs.LatencyStatus{Running: false}
	}
	if st.Running && time.Now().Unix()-st.UpdatedAt > 120 {
		st.Running = false
	}
	// 附带测速方案与下次计划时间，供空闲时展示倒计时（对应 ManageController::probeScheme）。
	raw, _ := json.Marshal(st)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	out["scheme"] = a.probeScheme()
	// 已点「停止」、正在等当前节点测完时为 true，前端据此显示「正在停止」。
	out["stopping"] = st.Running && a.jobsEnv().LatencyStopRequested()
	resp.OK(c, out)
}

// probeStop 停止正在进行的测速（手动、定时、「可用国家」触发的都适用）。
// 当前正在测的节点会测完，之后不再继续；只测了一部分时不做自动显隐。
func (a *App) probeStop(c *gin.Context) {
	if !a.jobsEnv().RequestLatencyStop() {
		resp.OK(c, gin.H{"stopping": false, "message": "当前没有正在进行的测速"})
		return
	}
	resp.OK(c, gin.H{"stopping": true})
}

// probeScheme 返回当前测速方案：是否启用、间隔、模式、自动显隐，以及下次计划时间。
// 调度器每分钟 tick 一次，并用 Redis 键 server_latency_next（有效期 = 间隔）节流，
// 所以下次运行时间 = 该键过期后的第一个整分钟。
func (a *App) probeScheme() gin.H {
	enabled := a.Set.Bool("server_latency_probe_enable", true)
	interval := a.Set.Int("server_latency_probe_interval", 3)
	if interval < 1 {
		interval = 1
	}
	if interval > 1440 {
		interval = 1440
	}
	var next any
	if enabled {
		nowTs := time.Now().Unix()
		at := nowTs
		if ttl := a.S.Redis.TTL(bgCtx(), "server_latency_next").Val(); ttl > 0 {
			at = nowTs + int64(ttl.Seconds()+0.999)
		}
		next = (at + 59) / 60 * 60
	}
	return gin.H{
		"enabled": enabled, "interval_minutes": interval,
		"mode":            a.Set.Get("node_probe_mode", "singbox"),
		"auto_visibility": a.Set.Bool("server_auto_manage_visibility", false),
		"next_run_at":     next,
	}
}

// ---- 节点：复制与流量 ----

func (a *App) nodeCopy(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var s model.Server
	if err := a.S.DB.First(&s, id).Error; err != nil {
		resp.Fail(c, 400, "服务器不存在")
		return
	}
	cp := s
	cp.ID = 0
	cp.Show = false
	cp.Code = nil
	cp.U, cp.D = 0, 0
	if err := a.S.DB.Create(&cp).Error; err != nil {
		resp.Fail(c, 500, "复制失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) nodeResetTraffic(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if err := a.S.DB.Model(&model.Server{}).Where("id = ?", id).Updates(map[string]any{"u": 0, "d": 0}).Error; err != nil {
		resp.Fail(c, 500, "重置失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) nodeBatchResetTraffic(c *gin.Context) {
	ids := idsFrom(bodyMap(c))
	if len(ids) == 0 {
		resp.Fail(c, 400, "请选择要重置的节点")
		return
	}
	a.S.DB.Model(&model.Server{}).Where("id IN ?", ids).Updates(map[string]any{"u": 0, "d": 0})
	resp.OK(c, true)
}

func (a *App) nodeBatchDelete(c *gin.Context) {
	ids := idsFrom(bodyMap(c))
	if len(ids) == 0 {
		resp.Fail(c, 400, "请选择要删除的节点")
		return
	}
	if err := a.S.DB.Where("id IN ?", ids).Delete(&model.Server{}).Error; err != nil {
		resp.Fail(c, 500, "批量删除失败")
		return
	}
	resp.OK(c, true)
}

// nodeBatchUpdate 对应 batchUpdate：只允许 show / enabled / machine_id。
func (a *App) nodeBatchUpdate(c *gin.Context) {
	m := bodyMap(c)
	ids := idsFrom(m)
	if len(ids) == 0 {
		resp.Fail(c, 400, "请选择要更新的节点")
		return
	}
	upd := map[string]any{}
	if v, ok := mInt(m, "show"); ok {
		upd["show"] = v != 0
	}
	if v, ok := mBool(m, "enabled"); ok {
		upd["enabled"] = v
	}
	if v, ok := mInt(m, "machine_id"); ok {
		if v == 0 {
			upd["machine_id"] = nil
		} else {
			upd["machine_id"] = v
		}
	}
	if len(upd) == 0 {
		resp.Fail(c, 400, "没有可更新的字段")
		return
	}
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&model.Server{}).Where("id IN ?", ids).Updates(upd).Error
	})
	if err != nil {
		resp.Fail(c, 500, "批量更新失败")
		return
	}
	resp.OK(c, true)
}

// ---- 订单：update / assign ----

func (a *App) adminOrderUpdate(c *gin.Context) {
	m := bodyMap(c)
	tn, _ := mStr(m, "trade_no")
	var o model.Order
	if err := a.S.DB.Where("trade_no = ?", tn).First(&o).Error; err != nil {
		resp.Fail(c, 400, "订单不存在")
		return
	}
	if v, ok := mInt(m, "commission_status"); ok {
		if err := a.S.DB.Model(&o).Update("commission_status", v).Error; err != nil {
			resp.Fail(c, 500, "更新失败")
			return
		}
	}
	resp.OK(c, true)
}

// adminOrderAssign 对应 OrderController::assign：为用户手动分配一笔订单。
func (a *App) adminOrderAssign(c *gin.Context) {
	m := bodyMap(c)
	email, _ := mStr(m, "email")
	var u model.User
	if email == "" || a.S.DB.Where("email = ?", email).First(&u).Error != nil {
		resp.Fail(c, 400, "该用户不存在")
		return
	}
	pid, _ := mInt(m, "plan_id")
	var p model.Plan
	if pid == 0 || a.S.DB.First(&p, pid).Error != nil {
		resp.Fail(c, 400, "该订阅不存在")
		return
	}
	var pending int64
	a.S.DB.Model(&model.Order{}).Where("user_id = ? AND status = 0", u.ID).Count(&pending)
	if pending > 0 {
		resp.Fail(c, 400, "该用户还有待支付的订单，无法分配")
		return
	}
	period, _ := mStr(m, "period")
	periodKey := shop.PeriodKey(period)
	total, _ := mInt(m, "total_amount")
	o := model.Order{
		UserID: u.ID, PlanID: p.ID, Period: periodKey, TradeNo: uuid.NewString(), TotalAmount: total,
	}
	switch {
	case periodKey == shop.PeriodReset:
		o.Type = shop.TypeResetTraffic
	case u.PlanID != nil && *u.PlanID != p.ID:
		o.Type = shop.TypeUpgrade
	case u.ExpiredAt != nil && *u.ExpiredAt > now() && u.PlanID != nil && *u.PlanID == p.ID:
		o.Type = shop.TypeRenewal
	default:
		o.Type = shop.TypeNewPurchase
	}
	ts := now()
	o.CreatedAt, o.UpdatedAt = ts, ts
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		if err := a.shop().ApplyInvite(tx, &u, &o); err != nil {
			return err
		}
		return tx.Create(&o).Error
	})
	if err != nil {
		resp.Fail(c, 500, "订单创建失败")
		return
	}
	resp.OK(c, o.TradeNo)
}

// ---- 统计扩展 ----

// adminStatOverride 对应 getOverride：收入、注册、待处理工单/佣金、在线情况与流量概览。
func (a *App) adminStatOverride(c *gin.Context) {
	y, m, d := time.Now().In(shop.Loc).Date()
	monthStart := time.Date(y, m, 1, 0, 0, 0, 0, shop.Loc).Unix()
	todayStart := time.Date(y, m, d, 0, 0, 0, 0, shop.Loc).Unix()
	lastMonthStart := time.Date(y, m-1, 1, 0, 0, 0, 0, shop.Loc).Unix()
	nowTs := now()
	var ticket, commission int64
	a.S.DB.Model(&model.Ticket{}).Where("status = 0").Count(&ticket)
	a.S.DB.Model(&model.Order{}).Where("commission_status = 0 AND invite_user_id IS NOT NULL AND status NOT IN (0, 2) AND commission_balance > 0").Count(&commission)
	var nodes []model.Server
	a.S.DB.Find(&nodes)
	online := 0
	for _, s := range nodes {
		if a.nodeOnline(s) {
			online++
		}
	}
	// 在线设备/用户：最近 10 分钟有流量记录（t）的用户。
	var onlineDevices, onlineUsers int64
	a.S.DB.Model(&model.User{}).Where("t >= ?", nowTs-600).
		Select("COALESCE(SUM(online_count),0)").Scan(&onlineDevices)
	a.S.DB.Model(&model.User{}).Where("t >= ?", nowTs-600).Count(&onlineUsers)
	resp.OK(c, gin.H{
		"month_income":                 a.sumOrders(monthStart, nowTs),
		"month_register_total":         a.countUsers(monthStart, nowTs),
		"ticket_pending_total":         ticket,
		"commission_pending_total":     commission,
		"day_income":                   a.sumOrders(todayStart, nowTs),
		"last_month_income":            a.sumOrders(lastMonthStart, monthStart),
		"commission_month_payout":      a.sumCommission(monthStart, nowTs),
		"commission_last_month_payout": a.sumCommission(lastMonthStart, monthStart),
		"online_nodes":                 online,
		"online_devices":               onlineDevices,
		"online_users":                 onlineUsers,
		"today_traffic":                a.trafficSum(todayStart, nowTs),
		"month_traffic":                a.trafficSum(monthStart, nowTs),
		"total_traffic":                a.trafficSum(0, 0),
	})
}

// statRankRange 解析 start/end 时间戳参数，缺省为 0 到当前。
func statRankRange(c *gin.Context, def int64) (int64, int64) {
	start, _ := strconv.ParseInt(c.Query("start_time"), 10, 64)
	end, _ := strconv.ParseInt(c.Query("end_time"), 10, 64)
	if start == 0 {
		start = def
	}
	if end == 0 {
		end = now()
	}
	return start, end
}

type serverRankRow struct {
	ServerID   int64  `json:"server_id"`
	ServerName string `json:"server_name"`
	ServerType string `json:"server_type"`
	U          int64  `json:"u"`
	D          int64  `json:"d"`
	Total      int64  `json:"total"`
}

// serverRank 对应 StatisticalService::getServerRank：按节点汇总日统计（record_type = d），按总量降序。
func (a *App) serverRank(start, end int64) []serverRankRow {
	var rows []serverRankRow
	a.S.DB.Raw(`SELECT s.id AS server_id, s.name AS server_name, s.type AS server_type,
		COALESCE(SUM(st.u),0) AS u, COALESCE(SUM(st.d),0) AS d, COALESCE(SUM(st.u + st.d),0) AS total
		FROM v2_server s JOIN v2_stat_server st ON st.server_id = s.id
		WHERE st.record_at >= ? AND st.record_at < ? AND st.record_type = 'd'
		GROUP BY s.id, s.name, s.type ORDER BY total DESC`, start, end).Scan(&rows)
	return rows
}

func (a *App) adminServerLastRank(c *gin.Context) {
	resp.OK(c, a.serverRank(0, now()+86400))
}

func (a *App) adminServerYesterdayRank(c *gin.Context) {
	y, m, d := time.Now().Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.Local).Unix()
	resp.OK(c, a.serverRank(today-86400, today))
}

// adminStatUser 对应 getStatUser：用户流量日明细分页。
func (a *App) adminStatUser(c *gin.Context) {
	uid, _ := strconv.ParseInt(c.Query("user_id"), 10, 64)
	cur, size := pageOf(c)
	if c.Query("pageSize") == "" {
		size = 10
	}
	q := a.S.DB.Model(&model.StatUser{}).Where("user_id = ?", uid)
	var total int64
	q.Count(&total)
	var rows []model.StatUser
	q.Order("record_at DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	paginateJSON(c, total, cur, size, rows)
}

// adminRanking 对应 getRanking：server_traffic_rank / user_consumption_rank / invite_rank，默认最近 30 天。
func (a *App) adminRanking(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 200 {
		limit = 20
	}
	start, end := statRankRange(c, now()-30*86400)
	switch c.Query("type") {
	case "server_traffic_rank":
		rows := a.serverRank(start, end)
		if len(rows) > limit {
			rows = rows[:limit]
		}
		resp.OK(c, rows)
	case "user_consumption_rank":
		type row struct {
			UserID int64
			U, D   int64
			Total  int64
		}
		var rows []row
		a.S.DB.Raw(`SELECT user_id, COALESCE(SUM(u),0) AS u, COALESCE(SUM(d),0) AS d, COALESCE(SUM(u+d),0) AS total
			FROM v2_stat_user WHERE record_at >= ? AND record_at < ? GROUP BY user_id ORDER BY total DESC LIMIT ?`, start, end, limit).Scan(&rows)
		emails := a.emailsByID(func(add func(int64)) {
			for _, r := range rows {
				add(r.UserID)
			}
		})
		out := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			out = append(out, gin.H{"user_id": r.UserID, "email": emailOrNil(emails, r.UserID), "u": r.U, "d": r.D, "total": r.Total})
		}
		resp.OK(c, out)
	case "invite_rank":
		type row struct {
			InviteUserID int64
			Count        int64
		}
		var rows []row
		a.S.DB.Raw(`SELECT invite_user_id, COUNT(*) AS count FROM v2_user
			WHERE created_at >= ? AND created_at < ? AND invite_user_id IS NOT NULL
			GROUP BY invite_user_id ORDER BY count DESC LIMIT ?`, start, end, limit).Scan(&rows)
		emails := a.emailsByID(func(add func(int64)) {
			for _, r := range rows {
				add(r.InviteUserID)
			}
		})
		out := make([]gin.H, 0, len(rows))
		for _, r := range rows {
			out = append(out, gin.H{"invite_user_id": r.InviteUserID, "email": emailOrNil(emails, r.InviteUserID), "count": r.Count})
		}
		resp.OK(c, out)
	default:
		resp.Fail(c, 422, "无效的排行类型")
	}
}

// adminTrafficRank 对应 getTrafficRank：当前时段前 10 名，并与上一等长时段对比增长率。
func (a *App) adminTrafficRank(c *gin.Context) {
	typ := c.Query("type")
	if typ != "node" && typ != "user" {
		resp.Fail(c, 422, "type 参数有误")
		return
	}
	start, end := statRankRange(c, now()-7*86400)
	prevStart := start - (end - start)
	type row struct {
		ID    int64
		Value int64
	}
	var cur []row
	var prev []row
	if typ == "node" {
		a.S.DB.Raw(`SELECT server_id AS id, COALESCE(SUM(u+d),0) AS value FROM v2_stat_server
			WHERE record_at >= ? AND record_at <= ? GROUP BY server_id ORDER BY value DESC LIMIT 10`, start, end).Scan(&cur)
	} else {
		a.S.DB.Raw(`SELECT user_id AS id, COALESCE(SUM(u+d),0) AS value FROM v2_stat_user
			WHERE record_at >= ? AND record_at <= ? GROUP BY user_id ORDER BY value DESC LIMIT 10`, start, end).Scan(&cur)
	}
	prevMap := map[int64]int64{}
	if len(cur) > 0 {
		ids := make([]int64, 0, len(cur))
		for _, r := range cur {
			ids = append(ids, r.ID)
		}
		if typ == "node" {
			a.S.DB.Raw(`SELECT server_id AS id, COALESCE(SUM(u+d),0) AS value FROM v2_stat_server
				WHERE record_at >= ? AND record_at < ? AND server_id IN ? GROUP BY server_id`, prevStart, start, ids).Scan(&prev)
		} else {
			a.S.DB.Raw(`SELECT user_id AS id, COALESCE(SUM(u+d),0) AS value FROM v2_stat_user
				WHERE record_at >= ? AND record_at < ? AND user_id IN ? GROUP BY user_id`, prevStart, start, ids).Scan(&prev)
		}
		for _, r := range prev {
			prevMap[r.ID] = r.Value
		}
	}
	// 名称：节点取 name，用户取邮箱；查不到时与 PHP 一样给出占位名。
	names := map[int64]string{}
	if len(cur) > 0 {
		ids := make([]int64, 0, len(cur))
		for _, r := range cur {
			ids = append(ids, r.ID)
		}
		if typ == "node" {
			var ss []model.Server
			a.S.DB.Select("id", "name").Where("id IN ?", ids).Find(&ss)
			for _, s := range ss {
				names[s.ID] = s.Name
			}
		} else {
			var us []model.User
			a.S.DB.Select("id", "email").Where("id IN ?", ids).Find(&us)
			for _, u := range us {
				names[u.ID] = u.Email
			}
		}
	}
	stamp := time.Unix(end, 0).In(shop.Loc).Format(time.RFC3339)
	out := make([]gin.H, 0, len(cur))
	for _, r := range cur {
		name, ok := names[r.ID]
		if !ok {
			if typ == "node" {
				name = fmt.Sprintf("Node %d", r.ID)
			} else {
				name = fmt.Sprintf("User %d", r.ID)
			}
		}
		prevV := prevMap[r.ID]
		change := 0.0
		if prevV > 0 {
			change = math.Round(float64(r.Value-prevV)/float64(prevV)*1000) / 10
		}
		out = append(out, gin.H{
			"id": strconv.FormatInt(r.ID, 10), "name": name, "value": r.Value,
			"previousValue": prevV, "change": change, "timestamp": stamp,
		})
	}
	// 与 PHP 一致：响应不带 status/message 包装，外层带 timestamp。
	c.JSON(200, gin.H{"timestamp": time.Now().In(shop.Loc).Format(time.RFC3339), "data": out})
}

// ---- 礼品卡：导出与统计 ----

func (a *App) giftExport(c *gin.Context) {
	batch, _ := mStr(requestParams(c), "batch_id")
	if batch == "" {
		resp.Fail(c, 422, "batch_id 不能为空")
		return
	}
	var codes []model.GiftCardCode
	a.S.DB.Select("code").Where("batch_id = ?", batch).Order("created_at ASC").Find(&codes)
	lines := make([]byte, 0, len(codes)*20)
	for i, cd := range codes {
		if i > 0 {
			lines = append(lines, '\n')
		}
		lines = append(lines, cd.Code...)
	}
	c.Header("Content-Disposition", `attachment; filename="gift_cards_`+batch+`.txt"`)
	c.Data(200, "text/plain; charset=utf-8", lines)
}

// giftStatistics 对应 GiftCardController::statistics：总量、按天使用次数、按模板类型统计。
func (a *App) giftStatistics(c *gin.Context) {
	start := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	end := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))
	var tplCount, activeTpl, codes, used, usages int64
	a.S.DB.Model(&model.GiftCardTemplate{}).Count(&tplCount)
	a.S.DB.Model(&model.GiftCardTemplate{}).Where("status = 1").Count(&activeTpl)
	a.S.DB.Model(&model.GiftCardCode{}).Count(&codes)
	a.S.DB.Model(&model.GiftCardCode{}).Where("status = ?", giftCodeUsed).Count(&used)
	a.S.DB.Model(&model.GiftCardUsage{}).Count(&usages)
	type daily struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var daily_ []daily
	a.S.DB.Raw(`SELECT DATE(FROM_UNIXTIME(created_at)) AS date, COUNT(*) AS count FROM v2_gift_card_usage
		WHERE DATE(FROM_UNIXTIME(created_at)) BETWEEN ? AND ? GROUP BY date ORDER BY date`, start, end).Scan(&daily_)
	type typeStat struct {
		TemplateID   int64
		TemplateName string
		Type         int
		Count        int64
	}
	var ts []typeStat
	a.S.DB.Raw(`SELECT u.template_id AS template_id, t.name AS template_name, t.type AS type, COUNT(*) AS count
		FROM v2_gift_card_usage u LEFT JOIN v2_gift_card_template t ON t.id = u.template_id
		GROUP BY u.template_id, t.name, t.type`).Scan(&ts)
	types := make([]gin.H, 0, len(ts))
	for _, s := range ts {
		types = append(types, gin.H{"template_name": s.TemplateName, "type_name": giftTypeName(s.Type), "count": s.Count})
	}
	resp.OK(c, gin.H{
		"total_stats": gin.H{"templates_count": tplCount, "active_templates_count": activeTpl,
			"codes_count": codes, "used_codes_count": used, "usages_count": usages},
		"daily_usages": daily_,
		"type_stats":   types,
	})
}

// ---- 系统状态 ----

// adminSystemStatus 返回数据库、Redis、节点推送通道与定时任务的健康信息（Go 版没有 Horizon，队列项不适用）。

// adminSyncStats 对应 ManageController::syncStats：按分组计算节点 revision，统计已同步的用户数。
func (a *App) adminSyncStats(c *gin.Context) {
	var lastChange int64
	var raw struct {
		Updated *int64
	}
	a.S.DB.Raw("SELECT UNIX_TIMESTAMP(MAX(updated_at)) AS updated FROM v2_server").Scan(&raw)
	if raw.Updated != nil {
		lastChange = *raw.Updated
	}
	var groups []int64
	a.S.DB.Model(&model.User{}).Where("group_id IS NOT NULL").Distinct().Pluck("group_id", &groups)
	synced := int64(0)
	for _, gid := range groups {
		g := gid
		probe := &model.User{GroupID: &g, UUID: "00000000-0000-0000-0000-000000000000"}
		servers, err := sub.AvailableServers(bgCtx(), a.S.DB, a.S.Redis, probe)
		if err != nil {
			continue
		}
		rev := nodeRevision(servers)
		var n int64
		a.S.DB.Model(&model.User{}).Where("group_id = ? AND node_revision = ?", gid, rev).Count(&n)
		synced += n
	}
	var reported, total int64
	a.S.DB.Model(&model.User{}).Where("node_revision IS NOT NULL").Count(&reported)
	a.S.DB.Model(&model.User{}).Count(&total)
	var lastAt any
	if lastChange > 0 {
		lastAt = lastChange
	}
	// 配置问题：没设权限组的套餐（买了也分不到节点）、有有效套餐却没有权限组的用户、客户端推送是否开启。
	type planNoGroup struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Users int64  `json:"users"`
	}
	var noGroup []planNoGroup
	a.S.DB.Raw(`SELECT p.id, p.name, COUNT(u.id) AS users FROM v2_plan p
		LEFT JOIN v2_user u ON u.plan_id = p.id AND (u.expired_at IS NULL OR u.expired_at > ?)
		WHERE p.group_id IS NULL GROUP BY p.id, p.name ORDER BY p.id`, now()).Scan(&noGroup)
	if noGroup == nil {
		noGroup = []planNoGroup{}
	}
	var usersNoGroup int64
	a.S.DB.Model(&model.User{}).
		Where("plan_id IS NOT NULL AND group_id IS NULL AND banned = 0 AND (expired_at IS NULL OR expired_at > ?)", now()).
		Count(&usersNoGroup)
	resp.OK(c, gin.H{"synced": synced, "reported": reported, "total": total, "last_node_change_at": lastAt,
		"plans_without_group": noGroup, "users_without_group": usersNoGroup,
		"client_push_enabled": a.Set.Bool("client_ws_enable", false)})
}

// adminECHKey 对应 generateEchKey：生成 X25519 的 ECH 密钥与配置（PEM 格式，字节布局与 PHP 一致）。
func (a *App) adminECHKey(c *gin.Context) {
	name, _ := mStr(requestParams(c), "public_name")
	if name == "" {
		name = "ech.example.com"
	}
	if len(name) > 253 {
		resp.Fail(c, 400, "public_name must be a valid domain (1-253 bytes)")
		return
	}
	priv := make([]byte, 32)
	_, _ = rand.Read(priv)
	key, err := ecdh.X25519().NewPrivateKey(priv)
	if err != nil {
		resp.Fail(c, 500, "生成失败")
		return
	}
	pub := key.PublicKey().Bytes()
	var idb [1]byte
	_, _ = rand.Read(idb[:])

	var contents []byte
	contents = append(contents, idb[0])
	contents = binary.BigEndian.AppendUint16(contents, 0x0020)
	contents = binary.BigEndian.AppendUint16(contents, 32)
	contents = append(contents, pub...)
	contents = binary.BigEndian.AppendUint16(contents, 8)
	contents = binary.BigEndian.AppendUint16(contents, 0x0001)
	contents = binary.BigEndian.AppendUint16(contents, 0x0001)
	contents = binary.BigEndian.AppendUint16(contents, 0x0001)
	contents = binary.BigEndian.AppendUint16(contents, 0x0003)
	contents = append(contents, 0)
	contents = append(contents, byte(len(name)))
	contents = append(contents, name...)
	contents = binary.BigEndian.AppendUint16(contents, 0)

	echConfig := binary.BigEndian.AppendUint16(nil, 0xfe0d)
	echConfig = binary.BigEndian.AppendUint16(echConfig, uint16(len(contents)))
	echConfig = append(echConfig, contents...)
	echConfigList := binary.BigEndian.AppendUint16(nil, uint16(len(echConfig)))
	echConfigList = append(echConfigList, echConfig...)

	payload := binary.BigEndian.AppendUint16(nil, 32)
	payload = append(payload, priv...)
	payload = binary.BigEndian.AppendUint16(payload, uint16(len(echConfig)))
	payload = append(payload, echConfig...)

	resp.OK(c, gin.H{
		"key":    pemBlock("ECH KEYS", payload),
		"config": pemBlock("ECH CONFIGS", echConfigList),
	})
}

// pemBlock 对应 "-----BEGIN X-----\n" + chunk_split(base64, 64) + "-----END X-----"。
func pemBlock(label string, data []byte) string {
	enc := base64.StdEncoding.EncodeToString(data)
	var b strings.Builder
	b.WriteString("-----BEGIN " + label + "-----\n")
	for len(enc) > 64 {
		b.WriteString(enc[:64] + "\n")
		enc = enc[64:]
	}
	if enc != "" {
		b.WriteString(enc + "\n")
	}
	b.WriteString("-----END " + label + "-----")
	return b.String()
}
