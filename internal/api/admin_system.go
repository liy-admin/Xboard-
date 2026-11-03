package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"xboard/internal/web"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/resp"
)

// 管理端系统类接口：审计日志、队列统计、系统状态、更新、插件与主题管理、支付表单与图标上传。
// 说明见 PROGRESS.md：插件与主题的 PHP 代码不会在 Go 版中执行，这里只管理记录、配置与文件。

// ---- 审计日志 ----

var sensitiveKeys = []string{"password", "token", "secret", "key", "api_key"}

// auditAction 对应 RequestLog::resolveAction：路径（去掉 api/vN/<secure_path>/ 前缀）中间段用 _ 连接，最后一段为方法。
func auditAction(rest string) string {
	p := strings.ReplaceAll(strings.Trim(rest, "/"), "-", "_")
	segs := strings.Split(p, "/")
	method := segs[len(segs)-1]
	return strings.Join(segs[:len(segs)-1], "_") + "." + method
}

// auditMW 记录管理端的 POST 操作（响应完成后写入 v2_admin_audit_log）。
func (a *App) auditMW(c *gin.Context) {
	params := requestParams(c) // 先于处理器读取，避免处理器消费掉请求体后审计数据为空
	c.Next()
	if c.Request.Method != http.MethodPost {
		return
	}
	admin, ok := c.Get(userKey)
	if !ok {
		return
	}
	u := admin.(*model.User)
	data := map[string]any{}
	for k, v := range params {
		skip := false
		for _, s := range sensitiveKeys {
			if strings.EqualFold(k, s) {
				skip = true
			}
		}
		if !skip {
			data[k] = v
		}
	}
	b, _ := json.Marshal(data)
	ts := now()
	a.S.DB.Exec("INSERT INTO v2_admin_audit_log (admin_id, action, method, uri, request_data, ip, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		u.ID, auditAction(c.Request.URL.Path), c.Request.Method, c.Request.URL.RequestURI(), string(b), c.ClientIP(), ts, ts)
}

// auditLog 对应 SystemController::getAuditLog。
func (a *App) auditLog(c *gin.Context) {
	cur, _ := strconv.Atoi(c.DefaultQuery("current", "1"))
	if cur < 1 {
		cur = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if size < 10 {
		size = 10
	}
	type row struct {
		ID          int64  `json:"id"`
		AdminID     int64  `json:"admin_id"`
		Action      string `json:"action"`
		Method      string `json:"method"`
		URI         string `json:"uri"`
		RequestData string `json:"request_data"`
		IP          string `json:"ip"`
		CreatedAt   int64  `json:"created_at"`
	}
	q := a.S.DB.Table("v2_admin_audit_log")
	if v := c.Query("action"); v != "" {
		q = q.Where("action = ?", v)
	}
	if v := c.Query("admin_id"); v != "" {
		q = q.Where("admin_id = ?", v)
	}
	if kw := c.Query("keyword"); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("(uri LIKE ? OR request_data LIKE ?)", like, like)
	}
	var total int64
	q.Count(&total)
	var rows []row
	q.Order("id DESC").Offset((cur - 1) * size).Limit(size).Find(&rows)
	ids := map[int64]bool{}
	for _, r := range rows {
		ids[r.AdminID] = true
	}
	list := make([]int64, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	emails := map[int64]string{}
	if len(list) > 0 {
		var us []model.User
		a.S.DB.Select("id", "email").Where("id IN ?", list).Find(&us)
		for _, u := range us {
			emails[u.ID] = u.Email
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var admin any
		if e, ok := emails[r.AdminID]; ok {
			admin = gin.H{"id": r.AdminID, "email": e}
		}
		out = append(out, gin.H{"id": r.ID, "admin_id": r.AdminID, "action": r.Action, "method": r.Method,
			"uri": r.URI, "request_data": r.RequestData, "ip": r.IP, "created_at": r.CreatedAt, "admin": admin})
	}
	c.JSON(200, gin.H{"data": out, "total": total})
}

// ---- 队列 ----

func (a *App) queueStats(c *gin.Context) {
	st := a.mailQ().Stats()
	resp.OK(c, gin.H{
		"failedJobs": st.Failed, "jobsPerMinute": st.JobsPerMinute, "pausedMasters": 0,
		"periods":   gin.H{"failedJobs": 10080, "recentJobs": 60},
		"processes": st.Workers, "queueWithMaxRuntime": nil, "queueWithMaxThroughput": nil,
		"recentJobs": st.Processed, "status": st.Running,
		"wait": []gin.H{},
	})
}

func (a *App) queueWorkload(c *gin.Context) {
	st := a.mailQ().Stats()
	resp.OK(c, []gin.H{{"name": "mail", "length": st.Pending, "wait": 0, "processes": st.Workers, "failed": st.Failed}})
}

func (a *App) horizonFailedJobs(c *gin.Context) {
	failed := a.mailQ().Failed()
	c.JSON(200, gin.H{"data": failed, "total": len(failed)})
}

// ---- 系统状态 ----

func (a *App) systemStatus(c *gin.Context) {
	var lastRun any
	schedOK := false
	if raw, err := a.S.Redis.Get(bgCtx(), "SCHEDULE_LAST_CHECK_AT").Result(); err == nil {
		if t, err := strconv.ParseInt(raw, 10, 64); err == nil {
			lastRun = t
			schedOK = now()-t <= 120
		}
	}
	resp.OK(c, gin.H{"schedule": schedOK, "horizon": a.mailQ().Stats().Running, "schedule_last_runtime": lastRun})
}

// ---- 插件 ----

// pluginDirs 为插件目录（XBOARD_PLUGINS_DIR，多个用逗号分隔，默认 plugins-core 与 plugins）。
func pluginDirs() []string {
	raw := os.Getenv("XBOARD_PLUGINS_DIR")
	if raw == "" {
		return []string{"plugins-core", "plugins"}
	}
	return strings.Split(raw, ",")
}

// pluginUploadDir 返回上传插件的落盘目录：第一个「非核心」目录，缺省 plugins。
// 之前取的是 pluginDirs()[0]，默认值里排第一的是 plugins-core，导致用户上传的插件被当成受保护的核心插件
// （无法删除）。PHP 的上传目录是 base_path('plugins')。
func pluginUploadDir() string {
	for _, d := range pluginDirs() {
		if filepath.Base(filepath.Clean(d)) != "plugins-core" {
			return d
		}
	}
	return "plugins"
}

type pluginManifest struct {
	Name        string         `json:"name"`
	Code        string         `json:"code"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Author      string         `json:"author"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	AdminMenus  any            `json:"admin_menus"`
	AdminCrud   any            `json:"admin_crud"`
	dir         string
}

// scanPlugins 扫描插件目录中的 config.json，按 code 去重（先出现者优先）。
func scanPlugins() map[string]pluginManifest {
	out := map[string]pluginManifest{}
	for _, d := range pluginDirs() {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(d, e.Name())
			raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
			if err != nil {
				continue
			}
			var m pluginManifest
			if json.Unmarshal(raw, &m) != nil || m.Code == "" {
				continue
			}
			if _, ok := out[m.Code]; ok {
				continue
			}
			m.dir = dir
			out[m.Code] = m
		}
	}
	// 磁盘上没有对应目录时，用二进制内嵌的核心插件清单兜底（单文件部署也能看到 / 安装核心插件）。
	for _, dir := range web.DirNames("plugins-core", true) {
		raw, err := web.ReadFile("plugins-core/" + dir + "/config.json")
		if err != nil {
			continue
		}
		var m pluginManifest
		if json.Unmarshal(raw, &m) != nil || m.Code == "" {
			continue
		}
		if _, ok := out[m.Code]; ok {
			continue
		}
		m.dir = "embed:plugins-core/" + dir
		out[m.Code] = m
	}
	return out
}

// readme 读取 README.md（若存在）。
func (m pluginManifest) readme() string {
	for _, f := range []string{"README.md", "readme.md"} {
		if strings.HasPrefix(m.dir, "embed:") {
			if b, err := web.ReadFile(strings.TrimPrefix(m.dir, "embed:") + "/" + f); err == nil {
				return string(b)
			}
			continue
		}
		if b, err := os.ReadFile(filepath.Join(m.dir, f)); err == nil {
			return string(b)
		}
	}
	return ""
}

// pluginRow 对应 v2_plugins 的一行（配置为 JSON）。
type pluginRow struct {
	ID        int64
	Code      string
	Version   string
	IsEnabled bool
	Config    string
	Type      string
}

func (a *App) pluginRowOf(code string) (pluginRow, bool) {
	var r pluginRow
	err := a.S.DB.Raw("SELECT id, code, version, is_enabled, config, type FROM v2_plugins WHERE code = ?", code).Scan(&r).Error
	return r, err == nil && r.ID != 0
}

// isCoreManifest 对应 PluginManager::isCorePlugin：位于核心插件目录（plugins-core，磁盘或内嵌）的插件受保护，不可删除。
func isCoreManifest(m pluginManifest) bool {
	d := filepath.ToSlash(m.dir)
	return strings.HasPrefix(d, "embed:plugins-core/") || strings.Contains(d, "/plugins-core/") || strings.HasPrefix(d, "plugins-core/")
}

// pluginConfigSchema 对应 PluginConfigService::getConfig：返回「字段定义 + 当前值」，
// 配置弹窗据此渲染标签、类型、选项。未保存的字段取 default，保存过（含 false / 空串）的取已存值。
func (a *App) pluginConfigSchema(m pluginManifest, row pluginRow, installed bool) map[string]any {
	saved := map[string]any{}
	if installed && row.Config != "" {
		_ = json.Unmarshal([]byte(row.Config), &saved)
	}
	out := map[string]any{}
	for k, v := range m.Config {
		field := map[string]any{"label": "", "placeholder": "", "description": "", "options": []any{}}
		if item, ok := v.(map[string]any); ok {
			for kk, vv := range item {
				field[kk] = vv
			}
		} else {
			field["default"] = v
		}
		val := field["default"]
		if sv, ok := saved[k]; ok && sv != nil {
			val = sv
		}
		field["value"] = val
		out[k] = field
	}
	return out
}

// pluginConfigForList 对应 getPlugins 的 config：已安装返回带值的定义，未安装返回 config.json 里的原始定义。
func (a *App) pluginConfigForList(m pluginManifest, row pluginRow, installed bool) any {
	if installed {
		return a.pluginConfigSchema(m, row, true)
	}
	if m.Config == nil {
		return map[string]any{}
	}
	return m.Config
}

// pluginConfigOf 合并插件默认值与已保存的配置。
func (a *App) pluginConfigOf(m pluginManifest, row pluginRow, installed bool) map[string]any {
	out := map[string]any{}
	for k, v := range m.Config {
		if spec, ok := v.(map[string]any); ok {
			out[k] = spec["default"]
		}
	}
	if installed && row.Config != "" {
		var saved map[string]any
		if json.Unmarshal([]byte(row.Config), &saved) == nil {
			for k, v := range saved {
				out[k] = v
			}
		}
	}
	return out
}

func (a *App) pluginTypes(c *gin.Context) {
	resp.OK(c, []gin.H{
		{"value": "feature", "label": "功能", "description": "提供功能扩展的插件，如Telegram登录、邮件通知等", "icon": "🔧"},
		{"value": "payment", "label": "支付方式", "description": "提供支付接口的插件，如支付宝、微信支付等", "icon": "💳"},
	})
}

func (a *App) pluginIndex(c *gin.Context) {
	typeFilter := c.Query("type")
	installed := map[string]pluginRow{}
	var rows []pluginRow
	a.S.DB.Raw("SELECT id, code, version, is_enabled, config, type FROM v2_plugins").Scan(&rows)
	for _, r := range rows {
		installed[r.Code] = r
	}
	manifests := scanPlugins()
	codes := make([]string, 0, len(manifests))
	for code := range manifests {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	out := []gin.H{}
	for _, code := range codes {
		m := manifests[code]
		pType := m.Type
		if pType == "" {
			pType = "feature"
		}
		if typeFilter != "" && pType != typeFilter {
			continue
		}
		row, isInstalled := installed[code]
		needUpgrade := isInstalled && m.Version != "" && row.Version != "" && versionCmp(m.Version, row.Version) > 0
		out = append(out, gin.H{
			"code": m.Code, "name": m.Name, "version": m.Version, "description": m.Description, "author": m.Author,
			"type": pType, "is_installed": isInstalled, "is_enabled": isInstalled && row.IsEnabled,
			"is_protected": isCoreManifest(m), "can_be_deleted": !isCoreManifest(m),
			"config": a.pluginConfigForList(m, row, isInstalled), "readme": m.readme(),
			"need_upgrade": needUpgrade, "admin_menus": m.AdminMenus, "admin_crud": m.AdminCrud,
		})
	}
	c.JSON(200, gin.H{"data": out})
}

func (a *App) pluginRequireCode(c *gin.Context) (string, pluginManifest, bool) {
	code, _ := mStr(requestParams(c), "code")
	if code == "" {
		resp.Fail(c, 422, "code 不能为空")
		return "", pluginManifest{}, false
	}
	m, ok := scanPlugins()[code]
	if !ok {
		resp.Fail(c, 400, "插件不存在")
		return "", pluginManifest{}, false
	}
	return code, m, true
}

func (a *App) pluginInstall(c *gin.Context) {
	code, m, ok := a.pluginRequireCode(c)
	if !ok {
		return
	}
	if _, has := a.pluginRowOf(code); has {
		resp.Fail(c, 400, "插件安装失败：已安装")
		return
	}
	cfg, _ := json.Marshal(a.pluginConfigOf(m, pluginRow{}, false))
	err := a.S.DB.Exec("INSERT INTO v2_plugins (name, code, type, version, is_enabled, config, installed_at, created_at, updated_at) VALUES (?, ?, ?, ?, 0, ?, NOW(), NOW(), NOW())",
		m.Name, code, pluginTypeOf(m), m.Version, string(cfg)).Error
	if err != nil {
		resp.Fail(c, 400, "插件安装失败："+err.Error())
		return
	}
	// 对应 Plugin::install → bootstrapEnv：BalalaSub 安装时后台拉起 `pip install ruyiPage`（不阻塞、失败只记日志）。
	if code == "balala_sub" {
		a.jobsEnv().BalalaBootstrapEnv()
	}
	c.JSON(200, gin.H{"message": "插件安装成功"})
}

func pluginTypeOf(m pluginManifest) string {
	if m.Type == "" {
		return "feature"
	}
	return m.Type
}

func (a *App) pluginUninstall(c *gin.Context) {
	code, _, ok := a.pluginRequireCode(c)
	if !ok {
		return
	}
	if r, has := a.pluginRowOf(code); has && r.IsEnabled {
		c.JSON(400, gin.H{"message": "请先禁用插件后再卸载"})
		return
	}
	a.S.DB.Exec("DELETE FROM v2_plugins WHERE code = ?", code)
	c.JSON(200, gin.H{"message": "插件卸载成功"})
}

func (a *App) pluginUpgrade(c *gin.Context) {
	code, m, ok := a.pluginRequireCode(c)
	if !ok {
		return
	}
	if _, has := a.pluginRowOf(code); !has {
		c.JSON(400, gin.H{"message": "插件升级失败：未安装"})
		return
	}
	a.S.DB.Exec("UPDATE v2_plugins SET version = ?, updated_at = NOW() WHERE code = ?", m.Version, code)
	c.JSON(200, gin.H{"message": "插件升级成功"})
}

func (a *App) pluginSetEnabled(enable bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, _, ok := a.pluginRequireCode(c)
		if !ok {
			return
		}
		row, has := a.pluginRowOf(code)
		if !has {
			c.JSON(400, gin.H{"message": map[bool]string{true: "插件启用失败：未安装", false: "插件禁用失败：未安装"}[enable]})
			return
		}
		_ = row
		a.S.DB.Exec("UPDATE v2_plugins SET is_enabled = ?, updated_at = NOW() WHERE code = ?", enable, code)
		msg := "插件禁用成功"
		if enable {
			msg = "插件启用成功"
		}
		c.JSON(200, gin.H{"message": msg})
	}
}

func (a *App) pluginGetConfig(c *gin.Context) {
	code, m, ok := a.pluginRequireCode(c)
	if !ok {
		return
	}
	row, has := a.pluginRowOf(code)
	resp.OK(c, a.pluginConfigSchema(m, row, has))
}

func (a *App) pluginUpdateConfig(c *gin.Context) {
	p := requestParams(c)
	code, _ := p["code"].(string)
	cfg, ok := p["config"].(map[string]any)
	if code == "" || !ok {
		resp.Fail(c, 422, "参数有误")
		return
	}
	if _, has := a.pluginRowOf(code); !has {
		resp.Fail(c, 400, "插件未安装")
		return
	}
	b, _ := json.Marshal(cfg)
	a.S.DB.Exec("UPDATE v2_plugins SET config = ?, updated_at = NOW() WHERE code = ?", string(b), code)
	c.JSON(200, gin.H{"message": "配置更新成功"})
}

// pluginUpload 解压上传的插件包到插件目录（仅存放文件，PHP 代码不会执行）。
func (a *App) pluginUpload(c *gin.Context) {
	data, ok := readUpload(c, "file", 50<<20, "zip")
	if !ok {
		return
	}
	dest := pluginUploadDir()
	name, err := unzipSafe(data, dest)
	if err != nil {
		c.JSON(400, gin.H{"message": "插件上传失败：" + err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "插件上传成功", "data": name})
}

func (a *App) pluginDelete(c *gin.Context) {
	code, m, ok := a.pluginRequireCode(c)
	if !ok {
		return
	}
	if isCorePlugin(code) {
		c.JSON(403, gin.H{"message": "该插件为系统核心插件，不允许删除"})
		return
	}
	a.S.DB.Exec("DELETE FROM v2_plugins WHERE code = ?", code)
	if err := os.RemoveAll(m.dir); err != nil {
		c.JSON(400, gin.H{"message": "插件删除失败：" + err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "插件删除成功"})
}

// isCorePlugin 与 PluginManager::isCorePlugin 一致：plugins-core 中的插件为核心插件。
func isCorePlugin(code string) bool {
	m, ok := scanPlugins()[code]
	return ok && isCoreManifest(m)
}

// ---- 主题 ----

func themeDir() string {
	if d := os.Getenv("XBOARD_THEME_DIR"); d != "" {
		return d
	}
	return "theme"
}

type themeEntry struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
	Path   string         `json:"-"`
}

// themeList 扫描主题目录：需要包含 config.json。
func themeList() []themeEntry {
	var out []themeEntry
	seen := map[string]bool{}
	if entries, err := os.ReadDir(themeDir()); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(themeDir(), e.Name(), "config.json"))
			if err != nil {
				continue
			}
			var cfg map[string]any
			_ = json.Unmarshal(raw, &cfg)
			out = append(out, themeEntry{Name: e.Name(), Config: cfg, Path: filepath.Join(themeDir(), e.Name())})
			seen[e.Name()] = true
		}
	}
	// 内嵌主题兜底：磁盘上没有 theme/ 目录时仍能列出默认主题。
	for _, n := range web.DirNames("theme", true) {
		if seen[n] {
			continue
		}
		if raw, err := web.ReadFile("theme/" + n + "/config.json"); err == nil {
			var cfg map[string]any
			_ = json.Unmarshal(raw, &cfg)
			out = append(out, themeEntry{Name: n, Config: cfg, Path: "embed:theme/" + n})
		}
	}
	return out
}

var themeNameRe = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)

// systemThemes 对应 ThemeService::SYSTEM_THEMES：内置主题不可删除。
var systemThemes = map[string]bool{"Xboard": true, "v2board": true}

// themeGetAll 对应 ThemeController::getThemes：themes 是「主题名 -> 主题信息」的对象
// （config.json 的 name/description/version/images/configs 平铺，再加 can_delete / is_system），
// 前端按 Object.entries 遍历并以键作为主题名。
func (a *App) themeGetAll(c *gin.Context) {
	active := a.Set.Get("frontend_theme", "Xboard")
	themes := map[string]any{}
	for _, t := range themeList() {
		info := map[string]any{}
		for k, v := range t.Config {
			info[k] = v
		}
		if _, ok := info["name"]; !ok {
			info["name"] = t.Name
		}
		system := systemThemes[t.Name]
		info["is_system"] = system
		info["can_delete"] = !system && t.Name != active
		themes[t.Name] = info
	}
	resp.OK(c, gin.H{"themes": themes, "active": active})
}

// themeDefaults 取主题 config.json 里各配置项的 default_value（缺省为空串）。
func themeDefaults(name string) (map[string]any, bool) {
	for _, t := range themeList() {
		if t.Name != name {
			continue
		}
		out := map[string]any{}
		if cfgs, ok := t.Config["configs"].([]any); ok {
			for _, it := range cfgs {
				f, _ := it.(map[string]any)
				fn, _ := f["field_name"].(string)
				if fn == "" {
					continue
				}
				if dv, ok := f["default_value"]; ok && dv != nil {
					out[fn] = dv
				} else {
					out[fn] = ""
				}
			}
		}
		return out, true
	}
	return nil, false
}

func (a *App) themeDelete(c *gin.Context) {
	name, _ := mStr(requestParams(c), "name")
	if !themeNameRe.MatchString(name) {
		resp.Fail(c, 400, "无效的主题名称")
		return
	}
	if name == "Xboard" || name == "v2board" {
		resp.Fail(c, 400, "系统主题不允许删除")
		return
	}
	if err := os.RemoveAll(filepath.Join(themeDir(), name)); err != nil {
		resp.Fail(c, 500, "删除失败")
		return
	}
	resp.OK(c, true)
}

func (a *App) themeUpload(c *gin.Context) {
	data, ok := readUpload(c, "file", 10<<20, "zip")
	if !ok {
		return
	}
	if _, err := unzipSafe(data, themeDir()); err != nil {
		resp.Fail(c, 500, "主题上传失败："+err.Error())
		return
	}
	resp.OK(c, true)
}

func (a *App) themeGetConfig(c *gin.Context) {
	name := c.Query("name")
	if !themeNameRe.MatchString(name) {
		resp.Fail(c, 400, "无效的主题名称")
		return
	}
	raw := a.Set.Get("theme_"+name, "")
	var saved map[string]any
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &saved)
	}
	// 对应 ThemeService::getConfig：从未保存过时按默认值初始化并持久化，保证配置页有初始值。
	if saved == nil {
		if defaults, ok := themeDefaults(name); ok {
			if b, err := json.Marshal(defaults); err == nil {
				_ = a.settingUpsert("theme_"+name, string(b))
				a.Set.Invalidate()
			}
			saved = defaults
		}
	}
	resp.OK(c, saved)
}

func (a *App) themeSaveConfig(c *gin.Context) {
	p := requestParams(c)
	name, _ := p["name"].(string)
	if !themeNameRe.MatchString(name) {
		resp.Fail(c, 400, "无效的主题名称")
		return
	}
	cfg, _ := json.Marshal(p["config"])
	if err := a.settingUpsert("theme_"+name, string(cfg)); err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	a.Set.Invalidate()
	resp.OK(c, true)
}

// readUpload 读取 multipart 表单中的文件，校验大小与扩展名。
func readUpload(c *gin.Context, field string, maxBytes int64, ext string) ([]byte, bool) {
	fh, err := c.FormFile(field)
	if err != nil {
		resp.Fail(c, 422, "请选择文件")
		return nil, false
	}
	if fh.Size > maxBytes {
		resp.Fail(c, 422, "文件大小超出限制")
		return nil, false
	}
	if !strings.HasSuffix(strings.ToLower(fh.Filename), "."+ext) {
		resp.Fail(c, 422, "文件类型不正确")
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		resp.Fail(c, 500, "读取失败")
		return nil, false
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil || int64(len(b)) > maxBytes {
		resp.Fail(c, 422, "文件读取失败")
		return nil, false
	}
	return b, true
}

// unzipSafe 解压 zip 到目标目录，拒绝路径穿越。返回第一层目录名。
func unzipSafe(data []byte, dest string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	root := ""
	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return "", fmt.Errorf("压缩包路径不合法")
		}
		if root == "" {
			root = strings.SplitN(name, string(filepath.Separator), 2)[0]
		}
		target := filepath.Join(dest, name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return "", err
		}
		_, err = io.Copy(out, io.LimitReader(rc, 50<<20))
		out.Close()
		rc.Close()
		if err != nil {
			return "", err
		}
	}
	return root, nil
}

// ---- 支付：表单与图标 ----

// paymentForm 对应 getPaymentForm：返回网关的配置表单（键为字段名）。
func (a *App) paymentForm(c *gin.Context) {
	name := c.Query("payment")
	if name == "" {
		name, _ = mStr(requestParams(c), "payment")
	}
	fields, ok := payGatewayForms[name]
	if !ok {
		// 网关名与插件目录名大小写不同（EPay / Epay、MGate / Mgate…），按不区分大小写匹配。
		for k, v := range payGatewayForms {
			if strings.EqualFold(k, name) {
				fields, ok = v, true
				break
			}
		}
	}
	if !ok {
		resp.Fail(c, 400, "支付方式不存在或未启用")
		return
	}
	// 对应 PaymentService::form()：带 id 时用该支付方式已保存的配置填充 value，
	// 否则前端编辑表单是空的，再点提交会把原有密钥覆盖成空。
	stored := map[string]any{}
	if id, ok := mInt(requestParams(c), "id"); ok && id > 0 {
		var p model.Payment
		if a.S.DB.First(&p, id).Error == nil {
			_ = json.Unmarshal([]byte(p.Config), &stored)
		}
	}
	resp.OK(c, buildPaymentForm(fields, stored))
}

// paymentIconUpload 对应 uploadIcon：仅接受 jpg/png/gif/webp，最大 2MB，文件名随机，类型由内容嗅探决定。
func (a *App) paymentIconUpload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 422, "请选择图标文件")
		return
	}
	if fh.Size > 2<<20 {
		resp.Fail(c, 422, "文件大小不能超过2MB")
		return
	}
	f, err := fh.Open()
	if err != nil {
		resp.Fail(c, 500, "上传失败")
		return
	}
	defer f.Close()
	b, _ := io.ReadAll(io.LimitReader(f, 2<<20+1))
	extMap := map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/gif": "gif", "image/webp": "webp"}
	ext, ok := extMap[http.DetectContentType(b)]
	if !ok {
		resp.Fail(c, 422, "文件必须是 jpg/jpeg/png/gif/webp 格式")
		return
	}
	dir := filepath.Join(publicDir(), "uploads", "payment")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		resp.Fail(c, 500, "上传目录无写入权限")
		return
	}
	name := auth.RandHex(16) + "." + ext
	if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
		resp.Fail(c, 500, "上传失败")
		return
	}
	resp.OK(c, gin.H{"url": a.subscribeRoot() + "/uploads/payment/" + name})
}

func publicDir() string {
	if d := os.Getenv("XBOARD_PUBLIC_DIR"); d != "" {
		return d
	}
	return "public"
}

// versionCmp 比较点分版本号（数字段），返回 -1/0/1。
func versionCmp(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
