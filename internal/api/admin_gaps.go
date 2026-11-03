package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"xboard/internal/web"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/model"
	"xboard/internal/notify"
	"xboard/internal/resp"
)

// 补齐与 PHP AdminRoute 之间缺失的管理端接口，以及请求参数的统一读取。

// paramsMW 把 JSON / 表单 body 里的标量参数镜像到查询串（已有同名查询参数优先）。
// PHP 的 $request->input() 同时读取 query 与 body；前端的列表接口用 POST 提交 body，
// 而许多处理函数直接读 c.Query，经此中间件后两种来源都能生效。数组 / 对象（filter、sort 等）不镜像，
// 它们通过 requestParams 读取。
func (a *App) paramsMW(c *gin.Context) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		q := c.Request.URL.Query()
		changed := false
		for k, v := range requestParams(c) {
			if _, exists := q[k]; exists {
				continue
			}
			switch x := v.(type) {
			case string:
				q.Set(k, x)
				changed = true
			case bool:
				if x {
					q.Set(k, "1")
				} else {
					q.Set(k, "0")
				}
				changed = true
			case float64, int, int64, json.Number:
				q.Set(k, fmt.Sprint(x))
				changed = true
			}
		}
		if changed {
			c.Request.URL.RawQuery = q.Encode()
		}
	}
	c.Next()
}

func listDirNames(dir string, onlyDirs bool) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if onlyDirs && !e.IsDir() {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)
	return out
}

// configGetEmailTemplate 对应 ConfigController::getEmailTemplate：列出 resources/views/mail 下的模板目录名。
func (a *App) configGetEmailTemplate(c *gin.Context) {
	base := os.Getenv("XBOARD_VIEWS_DIR")
	if base == "" {
		base = filepath.Join("resources", "views")
	}
	names := listDirNames(filepath.Join(base, "mail"), false)
	if names == nil {
		names = []string{"classic", "default"}
	}
	resp.OK(c, names)
}

// configGetThemeTemplate 对应 getThemeTemplate：列出 public/theme 下的主题目录名。
func (a *App) configGetThemeTemplate(c *gin.Context) {
	names := listDirNames(filepath.Join(publicDir(), "theme"), false)
	if names == nil {
		names = web.DirNames("theme", true)
	}
	if names == nil {
		names = []string{}
	}
	resp.OK(c, names)
}

func (a *App) telegramWebhookBase() string {
	if u := strings.TrimSpace(a.Set.Get("telegram_webhook_url", "")); u != "" {
		return strings.TrimRight(u, "/")
	}
	if u := strings.TrimSpace(a.Set.Get("app_url", "")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return ""
}

// configSetTelegramWebhook 对应 setTelegramWebhook：验证 token、设置 webhook 并注册命令菜单。
func (a *App) configSetTelegramWebhook(c *gin.Context) {
	m := bodyMap(c)
	base := a.telegramWebhookBase()
	if base == "" {
		resp.Fail(c, 422, "Telegram Webhook地址未配置")
		return
	}
	hook := base
	if !strings.Contains(hook, "/api/v1/guest/telegram/webhook") {
		hook += "/api/v1/guest/telegram/webhook"
	}
	reqToken, _ := mStr(m, "telegram_bot_token")
	tokenForSig := a.Set.Get("telegram_bot_token", reqToken)
	sum := md5.Sum([]byte(tokenForSig))
	hook += "?" + url.Values{"access_token": {hex.EncodeToString(sum[:])}}.Encode()

	cl := notify.NewTelegram(reqToken)
	if _, err := cl.GetMe(); err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	if _, err := cl.SetWebhook(hook); err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	a.registerBotCommands(cl)
	resp.OK(c, gin.H{"success": true, "webhook_url": hook, "webhook_base_url": base})
}

// configUploadDiscoverHTML 对应 uploadDiscoverHtml：保存为 .txt（避免浏览器直接执行脚本），并记录访问地址。
func (a *App) configUploadDiscoverHTML(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 422, "请选择HTML文件")
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".html" && ext != ".htm" && ext != ".txt" {
		resp.Fail(c, 422, "文件必须是 .html 格式")
		return
	}
	if fh.Size > 2<<20 {
		resp.Fail(c, 422, "文件大小不能超过2MB")
		return
	}
	dir := filepath.Join(publicDir(), "uploads", "discover")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		resp.Fail(c, 500, "上传目录无写入权限")
		return
	}
	name := auth.RandHex(16) + ".txt"
	if err := c.SaveUploadedFile(fh, filepath.Join(dir, name)); err != nil {
		resp.Fail(c, 500, "保存失败")
		return
	}
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	base := scheme + "://" + c.Request.Host
	// 配置了站点地址（通常是 https 域名）就用它：客户端的 WebView 拒绝加载 http:// 页面（ERR_CLEARTEXT_NOT_PERMITTED）。
	if u := strings.TrimRight(strings.TrimSpace(a.Set.Get("app_url", "")), "/"); strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		base = u
	}
	pub := base + "/uploads/discover/" + name
	_ = a.settingUpsert("discover_connected_page_url", pub)
	a.Set.Invalidate()
	resp.OK(c, gin.H{"url": pub})
}

// couponDrop 对应 CouponController::drop。
func (a *App) couponDrop(c *gin.Context) {
	id, ok := mInt(bodyMap(c), "id")
	if !ok {
		resp.Fail(c, 422, "优惠券ID不能为空")
		return
	}
	var cp model.Coupon
	if a.S.DB.First(&cp, id).Error != nil {
		resp.Fail(c, 400, "优惠券不存在")
		return
	}
	if err := a.S.DB.Delete(&cp).Error; err != nil {
		resp.Fail(c, 500, "删除失败")
		return
	}
	resp.OK(c, true)
}

// ---- BalalaSub 经后台统一通道的入口（ManageController::plugin*）----

func (a *App) pluginSubscription(c *gin.Context) {
	raw, ok := a.jobsEnv().BalalaLatest(c.Request.Context())
	var data struct {
		SubscribeURL  string `json:"subscribe_url"`
		Subscriptions any    `json:"subscriptions"`
		FetchedAt     any    `json:"fetched_at"`
	}
	if !ok || json.Unmarshal([]byte(raw), &data) != nil || data.SubscribeURL == "" {
		resp.Fail(c, 404, "暂无可用订阅链接，请先在 Balala 插件中执行抓取（xboard balala-sub:fetch）")
		return
	}
	if data.Subscriptions == nil {
		data.Subscriptions = []any{}
	}
	resp.OK(c, gin.H{"subscribe_url": data.SubscribeURL, "subscriptions": data.Subscriptions, "fetched_at": data.FetchedAt})
}

func (a *App) pluginDoctor(c *gin.Context) {
	if !a.pluginEnabled("balala_sub") {
		resp.Fail(c, 400, "Balala 插件未安装或未启用")
		return
	}
	full := requestParams(c)["full"]
	isFull := full == true || full == "1" || full == "true"
	env := a.jobsEnv()
	resp.OK(c, env.BalalaDoctor(c.Request.Context(), env.LoadBalalaConfig(), isFull))
}

func (a *App) pluginSetup(c *gin.Context) {
	if !a.pluginEnabled("balala_sub") {
		resp.Fail(c, 400, "Balala 插件未安装或未启用")
		return
	}
	env := a.jobsEnv()
	py, out, err := env.BalalaSetup(c.Request.Context(), env.LoadBalalaConfig())
	if err != nil {
		resp.Fail(c, 500, fmt.Sprintf("安装失败(python=%s)：%s", py, strings.TrimSpace(out)))
		return
	}
	resp.OK(c, gin.H{"message": fmt.Sprintf("环境已安装完成（python=%s）。现在可点「开始自检」验证。", py)})
}

func (a *App) pluginImport(c *gin.Context) {
	if !a.pluginEnabled("balala_sub") {
		resp.Fail(c, 400, "Balala 插件未安装或未启用")
		return
	}
	env := a.jobsEnv()
	if err := env.BalalaImport(true); err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	msg := "导入完成"
	if raw, ok, _ := env.Redis.Get(c.Request.Context(), "plugin:balala_sub:import_status"); ok {
		var st struct {
			Message string `json:"message"`
		}
		if json.Unmarshal([]byte(raw), &st) == nil && st.Message != "" {
			msg = st.Message
		}
	}
	resp.OK(c, gin.H{"message": msg})
}
