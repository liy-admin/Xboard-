package api

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/shop"
)

// 管理端 API。路径前缀是后台的 secure_path（与 PHP AdminRoute 的 admin_setting('secure_path') 一致），
// 因此不能写死：这里挂一个通配路由，校验前缀后交给内部的 admin 引擎分发。
//
// 当前覆盖：套餐、公告、优惠券、用户（列表/封禁/重置订阅）。其余模块见 PROGRESS.md。

// adminPathSetting 对应 admin_setting('secure_path', admin_setting('frontend_admin_path', crc32b(app.key)))。
// AdminPath 返回后台访问路径（secure_path 设置；缺省为 APP_KEY 的 crc32b）。
func (a *App) AdminPath() string { return a.adminPath(a.AppKey) }

func (a *App) adminPath(appKey string) string {
	def := fmt.Sprintf("%08x", crc32.ChecksumIEEE([]byte(appKey)))
	if v := a.Set.Get("secure_path", ""); v != "" {
		return strings.Trim(v, "/")
	}
	return a.Set.Get("frontend_admin_path", def)
}

// buildAdminEngine 构造内部分发引擎；路径不含 secure_path 前缀。
func (a *App) buildAdminEngine() *gin.Engine {
	e := gin.New()
	e.Use(gin.Recovery())
	g := e.Group("/", a.paramsMW, a.adminAuthMW, a.auditMW)

	g.POST("/plan/save", a.pushAfter("plan"), a.planSave)
	g.GET("/plan/fetch", a.planFetch2)
	g.POST("/plan/drop", a.pushAfter("plan"), a.planDrop)
	g.POST("/plan/update", a.pushAfter("plan"), a.planUpdate)
	g.POST("/plan/sort", a.planSort)

	g.GET("/notice/fetch", a.noticeFetchAdmin)
	// 与 PHP AdminRoute 对齐的补充接口（admin_gaps.go）。
	g.GET("/config/getEmailTemplate", a.configGetEmailTemplate)
	g.GET("/config/getThemeTemplate", a.configGetThemeTemplate)
	g.POST("/config/setTelegramWebhook", a.configSetTelegramWebhook)
	g.POST("/config/uploadDiscoverHtml", a.configUploadDiscoverHTML)
	g.POST("/coupon/drop", a.couponDrop)
	g.POST("/notice/update", a.noticeSave)
	g.GET("/server/manage/pluginSubscription", a.pluginSubscription)
	g.GET("/server/manage/pluginDoctor", a.pluginDoctor)
	g.GET("/server/manage/pluginSetup", a.pluginSetup)
	g.GET("/server/manage/pluginImport", a.pluginImport)
	g.POST("/notice/save", a.noticeSave)
	g.POST("/notice/show", a.noticeToggle)
	g.POST("/notice/drop", a.noticeDrop)
	g.POST("/notice/sort", a.noticeSort)

	g.GET("/coupon/fetch", a.couponFetch)

	g.POST("/coupon/fetch", a.couponFetch)
	g.POST("/coupon/generate", a.couponGenerate)
	g.POST("/coupon/show", a.couponToggle)
	g.POST("/coupon/update", a.couponUpdate)

	g.GET("/ticket/fetch", a.adminTicketFetch)

	g.POST("/ticket/fetch", a.adminTicketFetch)
	g.POST("/ticket/reply", a.adminTicketReply)
	g.POST("/ticket/close", a.adminTicketClose)

	g.GET("/order/fetch", a.adminOrderFetch)

	g.POST("/order/fetch", a.adminOrderFetch)
	g.GET("/order/detail", a.adminOrderDetail)
	g.POST("/order/detail", a.adminOrderDetail)
	g.POST("/order/paid", a.adminOrderPaid)
	g.POST("/order/cancel", a.adminOrderCancel)
	g.GET("/order/cleanupStatus", a.orderCleanupStatus)
	g.POST("/order/cleanupSave", a.orderCleanupSave)
	g.POST("/order/cleanupRun", a.orderCleanupRun)

	g.GET("/knowledge/fetch", a.adminKnowledgeFetch)
	g.GET("/knowledge/getCategory", a.adminKnowledgeCategories)
	g.POST("/knowledge/save", a.adminKnowledgeSave)
	g.POST("/knowledge/show", a.adminKnowledgeShow)
	g.POST("/knowledge/drop", a.adminKnowledgeDrop)
	g.POST("/knowledge/sort", a.adminKnowledgeSort)

	g.GET("/user/fetch", a.adminUserList)

	g.POST("/user/fetch", a.adminUserList)

	g.GET("/device/fetch", a.adminDeviceFetch)
	g.POST("/device/fetch", a.adminDeviceFetch)
	g.GET("/device/detail", a.adminDeviceDetail)
	g.POST("/device/detail", a.adminDeviceDetail)
	g.POST("/device/setTrialClaimed", a.adminDeviceSetTrialClaimed)

	g.GET("/server/group/fetch", a.groupFetch)
	g.POST("/server/group/save", a.pushAfter("group"), a.groupSave)
	g.POST("/server/group/drop", a.pushAfter("group"), a.groupDrop)
	g.GET("/server/route/fetch", a.routeFetch)
	g.POST("/server/route/save", a.pushAfter("route"), a.routeSave)
	g.POST("/server/route/drop", a.pushAfter("route"), a.routeDrop)

	g.GET("/stat/getStats", a.adminStats)
	g.GET("/stat/getStatRecord", a.adminStatRecord)
	g.GET("/stat/getOrder", a.adminStatOrder)

	g.GET("/payment/fetch", a.paymentFetch)
	g.GET("/payment/getPaymentMethods", a.paymentMethods)
	g.POST("/payment/save", a.paymentSave)
	g.POST("/payment/drop", a.paymentDrop)
	g.POST("/payment/show", a.paymentShow)
	g.POST("/payment/sort", a.paymentSort)

	g.GET("/gift-card/templates", a.giftTemplates)

	g.POST("/gift-card/templates", a.giftTemplates)
	g.POST("/gift-card/create-template", a.giftTemplateCreate)
	g.POST("/gift-card/update-template", a.giftTemplateUpdate)
	g.POST("/gift-card/delete-template", a.giftTemplateDelete)
	g.POST("/gift-card/generate-codes", a.giftGenerateCodes)
	g.GET("/gift-card/codes", a.giftCodes)
	g.POST("/gift-card/codes", a.giftCodes)
	g.POST("/gift-card/toggle-code", a.giftCodeToggle)
	g.POST("/gift-card/update-code", a.giftCodeUpdate)
	g.POST("/gift-card/delete-code", a.giftCodeDelete)
	g.GET("/gift-card/usages", a.giftUsages)
	g.POST("/gift-card/usages", a.giftUsages)
	g.GET("/gift-card/types", a.adminGiftTypes)

	g.GET("/server/machine/fetch", a.machineFetch)
	g.POST("/server/machine/save", a.machineSave)
	g.POST("/server/machine/drop", a.machineDrop)
	g.POST("/server/machine/resetToken", a.machineResetToken)
	g.POST("/server/machine/getToken", a.machineGetToken)
	g.GET("/server/machine/getToken", a.machineGetToken)
	g.POST("/server/machine/installCommand", a.machineInstall)
	g.GET("/server/machine/installCommand", a.machineInstall)
	g.GET("/server/machine/nodes", a.machineNodesAdmin)
	g.GET("/server/machine/history", a.machineHistory)
	g.POST("/server/machine/quickNode", a.pushAfter("node"), a.machineQuickNode)
	g.POST("/server/machine/testNode", a.machineNodeTest)
	g.GET("/server/machine/nodeLinks", a.machineNodeLinks)

	g.POST("/user/sendMail", a.adminSendMail)
	g.GET("/user/dumpCSV", a.adminDumpCSV)
	g.POST("/user/dumpCSV", a.adminDumpCSV)

	g.POST("/server/manage/probeLatency", a.probeLatency)
	g.GET("/server/manage/probeStatus", a.probeStatus)
	g.POST("/server/manage/probeStop", a.probeStop)
	g.POST("/server/manage/probeDiagnose", a.probeDiagnose)
	g.GET("/server/manage/probeDiagnose", a.probeDiagnose)
	g.POST("/server/manage/refreshCountries", a.countriesRefresh)
	g.GET("/server/manage/countriesStatus", a.countriesStatus)
	g.POST("/server/manage/singboxDownload", a.singboxDownload)
	g.GET("/server/manage/singboxDownloadStatus", a.singboxDownloadStatus)
	g.POST("/server/manage/copy", a.pushAfter("node"), a.pushMachineAfterNode(), a.nodeCopy)
	g.POST("/server/manage/resetTraffic", a.pushAfter("node"), a.nodeResetTraffic)
	g.POST("/server/manage/batchResetTraffic", a.pushAfter("node"), a.nodeBatchResetTraffic)
	g.POST("/server/manage/batchDelete", a.pushAfter("node"), a.pushMachineAfterNode(), a.nodeBatchDelete)
	g.GET("/server/manage/countByTag", a.nodeCountByTag)
	g.POST("/server/manage/deleteByTag", a.pushAfter("node"), a.nodeDeleteByTag)
	g.POST("/server/manage/batchUpdate", a.pushAfter("node"), a.pushMachineAfterNode(), a.nodeBatchUpdate)
	g.POST("/server/manage/batchSave", a.pushAfter("import"), a.nodeBatchSave)
	g.POST("/server/manage/batchImport", a.pushAfter("import"), a.nodeBatchImport)
	g.POST("/server/manage/parseNodes", a.nodeParse)
	g.GET("/server/manage/importSubscriptions", a.importSubscriptionsGet)
	g.POST("/server/manage/importSubscriptions", a.importSubscriptionsSave)

	g.POST("/order/update", a.adminOrderUpdate)
	g.POST("/order/assign", a.adminOrderAssign)

	g.GET("/stat/getOverride", a.adminStatOverride)
	g.GET("/stat/getServerLastRank", a.adminServerLastRank)
	g.GET("/stat/getServerYesterdayRank", a.adminServerYesterdayRank)
	g.GET("/stat/getStatUser", a.adminStatUser)
	g.POST("/stat/getStatUser", a.adminStatUser)
	g.GET("/stat/getRanking", a.adminRanking)
	g.GET("/stat/getTrafficRank", a.adminTrafficRank)

	g.GET("/gift-card/export-codes", a.giftExport)
	g.GET("/gift-card/statistics", a.giftStatistics)
	g.POST("/gift-card/statistics", a.giftStatistics)

	g.GET("/system/getSystemStatus", a.systemStatus)
	g.GET("/system/getQueueStats", a.queueStats)
	g.GET("/system/getQueueWorkload", a.queueWorkload)
	g.GET("/system/getHorizonFailedJobs", a.horizonFailedJobs)
	g.GET("/system/getAuditLog", a.auditLog)
	g.POST("/system/getAuditLog", a.auditLog)

	g.GET("/plugin/types", a.pluginTypes)
	g.GET("/plugin/getPlugins", a.pluginIndex)
	g.POST("/plugin/upload", a.pluginUpload)
	g.POST("/plugin/delete", a.pluginDelete)
	g.POST("/plugin/install", a.pluginInstall)
	g.POST("/plugin/uninstall", a.pluginUninstall)
	g.POST("/plugin/enable", a.pluginSetEnabled(true))
	g.POST("/plugin/disable", a.pluginSetEnabled(false))
	g.GET("/plugin/config", a.pluginGetConfig)
	g.POST("/plugin/config", a.pluginUpdateConfig)
	g.POST("/plugin/upgrade", a.pluginUpgrade)

	g.GET("/theme/getThemes", a.themeGetAll)
	g.POST("/theme/upload", a.themeUpload)
	g.POST("/theme/delete", a.themeDelete)
	g.POST("/theme/saveThemeConfig", a.themeSaveConfig)
	g.POST("/theme/getThemeConfig", a.themeGetConfig)

	g.POST("/payment/getPaymentForm", a.paymentForm)
	g.GET("/payment/getPaymentForm", a.paymentForm)
	g.POST("/payment/uploadIcon", a.paymentIconUpload)
	g.GET("/server/manage/syncStats", a.adminSyncStats)
	g.POST("/server/manage/generateEchKey", a.adminECHKey)
	g.GET("/server/manage/generateEchKey", a.adminECHKey)

	g.GET("/config/fetch", a.configFetch)
	g.POST("/config/save", a.configSave)
	g.POST("/config/testSendMail", a.configTestSendMail)

	g.GET("/server/manage/getNodes", a.nodeFetch)
	g.POST("/server/manage/sort", a.pushAfter("node"), a.nodeSort)
	g.POST("/server/manage/save", a.pushAfter("node"), a.pushMachineAfterNode(), a.nodeSave)
	g.POST("/server/manage/update", a.pushAfter("node"), a.pushMachineAfterNode(), a.nodeUpdate)
	g.POST("/server/manage/drop", a.pushAfter("node"), a.pushMachineAfterNode(), a.nodeDrop)

	g.GET("/referral/summary", a.referralSummary)
	g.GET("/referral/records", a.referralRecords)

	g.GET("/mail/template/list", a.mailTemplateList)
	g.GET("/mail/template/get", a.mailTemplateGet)
	g.POST("/mail/template/save", a.mailTemplateSave)
	g.POST("/mail/template/reset", a.mailTemplateReset)
	g.POST("/mail/template/test", a.mailTemplateTest)

	g.GET("/traffic-reset/logs", a.trafficResetLogs)
	g.GET("/traffic-reset/stats", a.trafficResetStats)
	g.GET("/traffic-reset/user/:uid/history", a.trafficResetHistory)
	g.POST("/traffic-reset/reset-user", a.trafficResetUser)
	g.POST("/user/ban", a.adminUserBan)
	g.POST("/user/update", a.adminUserUpdate)
	g.POST("/user/destroy", a.adminUserDestroy)
	g.GET("/user/getUserInfoById", a.adminUserInfo)
	g.GET("/user/getOnlineUsers", a.adminUserOnline)
	g.POST("/user/generate", a.adminUserGenerate)
	g.POST("/user/resetSecret", a.adminUserResetSecret)
	return e
}

// adminDispatch 校验 secure_path 前缀后把请求交给内部引擎。
func (a *App) adminDispatch(c *gin.Context) {
	// GET /{secure_path}：管理端页面外壳（PHP 的 Route::get('/'.secure_path)）。
	if c.Request.Method == http.MethodGet && strings.Trim(c.Request.URL.Path, "/") == a.adminPath(a.AppKey) {
		a.adminShell(c)
		return
	}
	p := strings.TrimPrefix(c.Request.URL.Path, "/api/v2/")
	if p == c.Request.URL.Path {
		c.AbortWithStatus(404)
		return
	}
	sp, rest, _ := strings.Cut(p, "/")
	if sp == "" || sp != a.adminPath(a.AppKey) {
		c.AbortWithStatus(404)
		return
	}
	c.Request.URL.Path = "/" + rest
	a.adminEngine().HandleContext(c)
}

func (a *App) adminEngine() *gin.Engine {
	a.adminOnce.Do(func() { a.admin = a.buildAdminEngine() })
	return a.admin
}

func (a *App) adminAuthMW(c *gin.Context) {
	u := auth.UserFromBearer(a.S.DB, c.GetHeader("Authorization"))
	if u == nil || !u.IsAdmin {
		c.AbortWithStatusJSON(403, gin.H{"message": "Unauthorized"})
		return
	}
	c.Set(userKey, u)
	c.Next()
}

// pageOf 解析 current/pageSize（Coupon 等接口使用的参数名）。
func pageOf(c *gin.Context) (int, int) {
	// 同时支持查询串与 JSON / 表单 body（PHP 的 $request->input 两者都读）。
	p := requestParams(c)
	cur, _ := anyInt(p["current"])
	if cur < 1 {
		cur = 1
	}
	size, _ := anyInt(p["pageSize"])
	if size < 1 {
		size = 10
	}
	return int(cur), int(size)
}

// paginateJSON 对应 ApiResponse::paginate 的输出结构（不带 status 外壳）。
func paginateJSON(c *gin.Context, total int64, cur, size int, data any) {
	last := int((total + int64(size) - 1) / int64(size))
	if last < 1 {
		last = 1
	}
	c.JSON(200, gin.H{"total": total, "current_page": cur, "per_page": size, "last_page": last, "data": data})
}

// bodyMap 读取 JSON 或表单请求体为通用 map，保留数值类型（用 json.Number）。
func bodyMap(c *gin.Context) map[string]any {
	out := make(map[string]any, len(requestParams(c)))
	for k, v := range requestParams(c) {
		out[k] = v
	}
	return out
}

func mInt(m map[string]any, k string) (int64, bool) {
	switch v := m[k].(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, true
		}
		if f, err := v.Float64(); err == nil {
			return int64(f), true
		}
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	case float64:
		return int64(v), true
	}
	return 0, false
}

func mBool(m map[string]any, k string) (bool, bool) {
	switch v := m[k].(type) {
	case bool:
		return v, true
	case json.Number:
		i, err := v.Int64()
		return i != 0, err == nil
	case string:
		return v != "" && v != "0" && v != "false", true
	case float64:
		return v != 0, true
	}
	return false, false
}

func mStr(m map[string]any, k string) (string, bool) {
	switch v := m[k].(type) {
	case string:
		return v, true
	case nil:
		return "", false
	default:
		b, _ := json.Marshal(v)
		return string(b), true
	}
}

// ---- 套餐（PlanController） ----

func (a *App) planFetch2(c *gin.Context) {
	var plans []model.Plan
	a.S.DB.Order("sort ASC").Find(&plans)
	groupIDs := map[int64]bool{}
	for _, p := range plans {
		if p.GroupID != nil {
			groupIDs[*p.GroupID] = true
		}
	}
	groups := map[int64]gin.H{}
	if len(groupIDs) > 0 {
		ids := make([]int64, 0, len(groupIDs))
		for id := range groupIDs {
			ids = append(ids, id)
		}
		var gs []model.ServerGroup
		a.S.DB.Select("id", "name").Where("id IN ?", ids).Find(&gs)
		for _, g := range gs {
			groups[g.ID] = gin.H{"id": g.ID, "name": g.Name}
		}
	}
	// Sell 在 PHP 里没有布尔转换，输出 0/1；外层字段覆盖内嵌 Plan 的 bool 字段。
	type row struct {
		model.Plan
		Sell        int   `json:"sell"`
		Group       any   `json:"group"`
		UsersCount  int64 `json:"users_count"`
		ActiveUsers int64 `json:"active_users_count"`
	}
	out := make([]row, 0, len(plans))
	for _, p := range plans {
		r := row{Plan: p}
		if p.Sell {
			r.Sell = 1
		}
		if p.GroupID != nil {
			if g, ok := groups[*p.GroupID]; ok {
				r.Group = g
			}
		}
		a.S.DB.Model(&model.User{}).Where("plan_id = ?", p.ID).Count(&r.UsersCount)
		a.S.DB.Model(&model.User{}).Where("plan_id = ? AND (expired_at > ? OR expired_at IS NULL)", p.ID, now()).
			Count(&r.ActiveUsers)
		out = append(out, r)
	}
	resp.OK(c, out)
}

var planWritable = []string{"name", "content", "group_id", "transfer_enable", "speed_limit", "device_limit",
	"prices", "reset_traffic_method", "capacity_limit", "tags", "show", "sell", "renew"}

func (a *App) planSave(c *gin.Context) {
	m := bodyMap(c)
	upd := map[string]any{}
	for _, k := range planWritable {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch k {
		case "prices", "tags":
			b, _ := json.Marshal(v)
			upd[k] = json.RawMessage(b)
		case "show", "sell", "renew":
			b, _ := mBool(m, k)
			upd[k] = b
		default:
			upd[k] = v
		}
	}
	if id, ok := mInt(m, "id"); ok && id > 0 {
		var p model.Plan
		if err := a.S.DB.First(&p, id).Error; err != nil {
			resp.Fail(c, 400, "该订阅不存在")
			return
		}
		err := a.S.DB.Transaction(func(tx *gorm.DB) error {
			if force, _ := mBool(m, "force_update"); force {
				ut := map[string]any{}
				if g, ok := mInt(m, "group_id"); ok {
					ut["group_id"] = g
				}
				if t, ok := mInt(m, "transfer_enable"); ok {
					ut["transfer_enable"] = shop.GbToTransfer(t)
				}
				if s, ok := mInt(m, "speed_limit"); ok {
					ut["speed_limit"] = s
				}
				if d, ok := mInt(m, "device_limit"); ok {
					ut["device_limit"] = d
				}
				if len(ut) > 0 {
					if err := tx.Model(&model.User{}).Where("plan_id = ?", p.ID).Updates(ut).Error; err != nil {
						return err
					}
				}
			}
			upd["updated_at"] = now()
			return tx.Model(&p).Updates(upd).Error
		})
		if err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		resp.OK(c, true)
		return
	}
	upd["created_at"], upd["updated_at"] = now(), now()
	if err := a.S.DB.Model(&model.Plan{}).Create(upd).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) planDrop(c *gin.Context) {
	m := bodyMap(c)
	id, _ := mInt(m, "id")
	var n int64
	a.S.DB.Model(&model.Order{}).Where("plan_id = ?", id).Count(&n)
	if n > 0 {
		resp.Fail(c, 400, "该订阅下存在订单无法删除")
		return
	}
	a.S.DB.Model(&model.User{}).Where("plan_id = ?", id).Count(&n)
	if n > 0 {
		resp.Fail(c, 400, "该订阅下存在用户无法删除")
		return
	}
	res := a.S.DB.Delete(&model.Plan{}, id)
	if res.RowsAffected == 0 {
		resp.Fail(c, 400, "该订阅不存在")
		return
	}
	resp.OK(c, true)
}

func (a *App) planUpdate(c *gin.Context) {
	m := bodyMap(c)
	id, _ := mInt(m, "id")
	upd := map[string]any{}
	for _, k := range []string{"show", "renew", "sell"} {
		if v, ok := mBool(m, k); ok {
			upd[k] = v
		}
	}
	var p model.Plan
	if err := a.S.DB.First(&p, id).Error; err != nil {
		resp.Fail(c, 400, "该订阅不存在")
		return
	}
	if err := a.S.DB.Model(&p).Updates(upd).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) planSort(c *gin.Context) {
	m := bodyMap(c)
	ids, _ := m["ids"].([]any)
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		for i, v := range ids {
			id, ok := toNum(v)
			if !ok {
				return fmt.Errorf("bad id")
			}
			if err := tx.Model(&model.Plan{}).Where("id = ?", int64(id)).Update("sort", i+1).Error; err != nil {
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

// ---- 公告（NoticeController） ----

func (a *App) noticeFetchAdmin(c *gin.Context) {
	var rows []model.Notice
	a.S.DB.Order("sort ASC").Order("id DESC").Find(&rows)
	resp.OK(c, rows)
}

func (a *App) noticeSave(c *gin.Context) {
	m := bodyMap(c)
	upd := map[string]any{}
	for _, k := range []string{"title", "content", "img_url", "url"} {
		if v, ok := mStr(m, k); ok {
			upd[k] = v
		}
	}
	if v, ok := m["tags"]; ok && v != nil {
		b, _ := json.Marshal(v)
		upd["tags"] = json.RawMessage(b)
	}
	for _, k := range []string{"show", "popup"} {
		if v, ok := mBool(m, k); ok {
			upd[k] = v
		}
	}
	if id, ok := mInt(m, "id"); ok && id > 0 {
		if err := a.S.DB.Model(&model.Notice{}).Where("id = ?", id).Updates(upd).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		resp.OK(c, true)
		return
	}
	upd["created_at"], upd["updated_at"] = now(), now()
	if err := a.S.DB.Model(&model.Notice{}).Create(upd).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) noticeToggle(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if id == 0 {
		resp.Fail(c, 500, "公告ID不能为空")
		return
	}
	var n model.Notice
	if err := a.S.DB.First(&n, id).Error; err != nil {
		resp.Fail(c, 400, "公告不存在")
		return
	}
	if err := a.S.DB.Model(&n).Update("show", !n.Show).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) noticeDrop(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	if id == 0 {
		resp.Fail(c, 422, "公告ID不能为空")
		return
	}
	res := a.S.DB.Delete(&model.Notice{}, id)
	if res.RowsAffected == 0 {
		resp.Fail(c, 400, "公告不存在")
		return
	}
	resp.OK(c, true)
}

func (a *App) noticeSort(c *gin.Context) {
	ids, _ := bodyMap(c)["ids"].([]any)
	err := a.S.DB.Transaction(func(tx *gorm.DB) error {
		for i, v := range ids {
			id, ok := toNum(v)
			if !ok {
				return fmt.Errorf("bad id")
			}
			if err := tx.Model(&model.Notice{}).Where("id = ?", int64(id)).Update("sort", i+1).Error; err != nil {
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

// ---- 优惠券（CouponController） ----

var couponFilterCols = map[string]bool{"id": true, "code": true, "name": true, "type": true, "value": true,
	"show": true, "limit_use": true, "limit_use_with_user": true, "started_at": true, "ended_at": true,
	"created_at": true, "updated_at": true}

func (a *App) couponFetch(c *gin.Context) {
	cur, size := pageOf(c)
	q := a.S.DB.Model(&model.Coupon{})
	var filters []struct {
		ID    string `json:"id"`
		Value any    `json:"value"`
	}
	if raw := c.Query("filter"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &filters)
	}
	for _, f := range filters {
		if !couponFilterCols[f.ID] {
			continue
		}
		switch v := f.Value.(type) {
		case []any:
			q = q.Where(f.ID+" IN ?", v)
		default:
			q = q.Where(f.ID+" LIKE ?", fmt.Sprintf("%%%v%%", v))
		}
	}
	var total int64
	q.Count(&total)
	var rows []model.Coupon
	q.Order("created_at DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	paginateJSON(c, total, cur, size, rows)
}

func (a *App) couponGenerate(c *gin.Context) {
	m := bodyMap(c)
	count, _ := mInt(m, "generate_count")
	base := map[string]any{}
	for _, k := range []string{"name", "type", "value", "limit_use", "limit_use_with_user", "started_at", "ended_at"} {
		if v, ok := m[k]; ok && v != nil {
			base[k] = v
		}
	}
	if v, ok := mBool(m, "show"); ok {
		base["show"] = v
	}
	for _, k := range []string{"limit_plan_ids", "limit_period"} {
		if v, ok := m[k]; ok && v != nil {
			b, _ := json.Marshal(v)
			base[k] = json.RawMessage(b)
		}
	}
	ts := now()
	base["created_at"], base["updated_at"] = ts, ts

	if count > 0 {
		err := a.S.DB.Transaction(func(tx *gorm.DB) error {
			for i := int64(0); i < count; i++ {
				row := copyMap(base)
				row["code"] = randomChars(8)
				row["show"] = true
				if err := tx.Model(&model.Coupon{}).Create(row).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			resp.Fail(c, 500, "创建失败")
			return
		}
		resp.OK(c, true)
		return
	}
	if id, ok := mInt(m, "id"); ok && id > 0 {
		if code, ok := mStr(m, "code"); ok {
			base["code"] = code
		}
		if err := a.S.DB.Model(&model.Coupon{}).Where("id = ?", id).Updates(base).Error; err != nil {
			resp.Fail(c, 500, "保存失败")
			return
		}
		resp.OK(c, true)
		return
	}
	if code, ok := mStr(m, "code"); ok && code != "" {
		base["code"] = code
	} else {
		base["code"] = randomChars(8)
	}
	if err := a.S.DB.Model(&model.Coupon{}).Create(base).Error; err != nil {
		resp.Fail(c, 500, "创建失败")
		return
	}
	resp.OK(c, true)
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m)+2)
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (a *App) couponToggle(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var cp model.Coupon
	if err := a.S.DB.First(&cp, id).Error; err != nil {
		resp.Fail(c, 400, "优惠券不存在")
		return
	}
	if err := a.S.DB.Model(&cp).Update("show", !cp.Show).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) couponUpdate(c *gin.Context) {
	m := bodyMap(c)
	id, ok := mInt(m, "id")
	if !ok || id == 0 {
		resp.Fail(c, 422, "优惠券ID不能为空")
		return
	}
	upd := map[string]any{}
	if v, ok := mBool(m, "show"); ok {
		upd["show"] = v
	}
	if len(upd) == 0 {
		resp.OK(c, true)
		return
	}
	if err := a.S.DB.Model(&model.Coupon{}).Where("id = ?", id).Updates(upd).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

// ---- 用户（UserController，子集） ----

func (a *App) adminUserBan(c *gin.Context) {
	m := bodyMap(c)
	if scope, _ := mStr(m, "scope"); scope != "" {
		a.adminUserBatchBan(c, scope, m)
		return
	}
	// 兼容旧用法：按 id 切换单个用户的封禁状态。
	id, _ := mInt(m, "id")
	var u model.User
	if err := a.S.DB.First(&u, id).Error; err != nil {
		resp.Fail(c, 400, "用户不存在")
		return
	}
	if err := a.S.DB.Model(&u).Update("banned", !u.Banned).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}

// adminUserBatchBan 对应 UserController::ban：按范围批量封禁（all 全部 / filtered 当前筛选结果 / selected 已选用户）。
//
// 与 PHP 的差异：不会封禁管理员和员工账号。PHP 在 scope=all 时会把管理员一并封掉，
// 一次误点就可能让所有人（包括自己）无法登录后台。
func (a *App) adminUserBatchBan(c *gin.Context, scope string, m map[string]any) {
	if scope != "all" && scope != "filtered" && scope != "selected" {
		resp.Fail(c, 422, "scope 只能是 all、filtered 或 selected")
		return
	}
	q, err := a.adminUserScope(c, scope, idList(m["user_ids"]))
	if err != nil {
		resp.Fail(c, 422, err.Error())
		return
	}
	// 先取出命中的用户 ID，再分批更新：筛选里可以带 or 逻辑，直接给查询追加 AND 条件会改变 A OR B 的优先级，
	// 让管理员保护失效。
	var ids []int64
	if err := q.Pluck("id", &ids).Error; err != nil {
		resp.Fail(c, 500, "处理失败")
		return
	}
	const chunk = 1000
	for i := 0; i < len(ids); i += chunk {
		end := i + chunk
		if end > len(ids) {
			end = len(ids)
		}
		if err := a.S.DB.Model(&model.User{}).
			Where("id IN ? AND is_admin = 0 AND is_staff = 0", ids[i:end]).
			Update("banned", true).Error; err != nil {
			resp.Fail(c, 500, "处理失败")
			return
		}
	}
	resp.OK(c, true)
}

func (a *App) adminUserResetSecret(c *gin.Context) {
	id, _ := mInt(bodyMap(c), "id")
	var u model.User
	if err := a.S.DB.First(&u, id).Error; err != nil {
		resp.Fail(c, 400, "用户不存在")
		return
	}
	upd := map[string]any{"token": auth.RandHex(16), "uuid": uuid.NewString()}
	if err := a.S.DB.Model(&u).Updates(upd).Error; err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	resp.OK(c, true)
}
