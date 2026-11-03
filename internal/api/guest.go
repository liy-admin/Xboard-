package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/notify"
	"xboard/internal/resp"
	"xboard/internal/sub"
)

// 免登录的公共接口（V1 GuestRoute）、客户端 App 接口（V1 ClientRoute）与邮件链接登录。

var defaultEmailSuffixes = []string{"gmail.com", "qq.com", "163.com", "yahoo.com", "sina.com", "126.com",
	"outlook.com", "yeah.net", "foxmail.com"}

// emailSuffixList 对应 Helper::getEmailSuffix：设置里的白名单后缀，缺省为常见邮箱。
func (a *App) emailSuffixList() []string {
	raw := strings.TrimSpace(a.Set.Get("email_whitelist_suffix", ""))
	if raw == "" {
		return defaultEmailSuffixes
	}
	var arr []string
	if json.Unmarshal([]byte(raw), &arr) == nil && len(arr) > 0 {
		return arr
	}
	var out []string
	for _, s := range strings.Split(raw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return defaultEmailSuffixes
	}
	return out
}

func (a *App) registerGuestRoutes(r *gin.Engine) {
	g := r.Group("/api/v1/guest")
	g.GET("/plan/fetch", a.guestPlanFetch)
	g.GET("/comm/config", a.guestCommConfig)
	g.GET("/notice/fetch", a.noticeFetch)
	g.GET("/website/config", a.guestWebsiteConfig)
	g.GET("/discover/config", a.guestDiscoverConfig)
	g.GET("/anti_loss/config", a.guestAntiLossConfig)
	g.GET("/ad-banner/config", a.guestAdBannerConfig)
	g.GET("/client/routing", a.guestClientRouting)

	cl := r.Group("/api/v1/client", a.clientAuth)
	cl.GET("/app/getConfig", a.clientAppConfig)
	cl.GET("/app/getVersion", a.clientAppVersion)
}

// guestCommConfig 对应 Guest\CommController::config。
func (a *App) guestCommConfig(c *gin.Context) {
	suffix := any(0)
	if a.Set.Bool("email_whitelist_enable", false) {
		suffix = a.emailSuffixList()
	}
	webLogin := boolInt(a.Set.Bool("user_web_login_enable", true))
	resp.OK(c, gin.H{
		"tos_url":                      a.settingOrNil("tos_url"),
		"is_email_verify":              boolInt(a.Set.Bool("email_verify", false)),
		"is_invite_force":              boolInt(a.Set.Bool("invite_force", false)),
		"email_whitelist_suffix":       suffix,
		"is_captcha":                   boolInt(a.Set.Bool("captcha_enable", false)),
		"captcha_type":                 a.Set.Get("captcha_type", "recaptcha"),
		"recaptcha_site_key":           a.settingOrNil("recaptcha_site_key"),
		"recaptcha_v3_site_key":        a.settingOrNil("recaptcha_v3_site_key"),
		"recaptcha_v3_score_threshold": a.Set.Float("recaptcha_v3_score_threshold", 0.5),
		"turnstile_site_key":           a.settingOrNil("turnstile_site_key"),
		"app_description":              a.settingOrNil("app_description"),
		"app_url":                      a.settingOrNil("app_url"),
		"logo":                         a.settingOrNil("logo"),
		"is_web_login_enable":          webLogin,
		"is_recaptcha":                 boolInt(a.Set.Bool("captcha_enable", false)),
	})
}

// guestPlanFetch 对应 Guest\PlanController::fetch：展示中且在售、仍有名额的套餐。
func (a *App) guestPlanFetch(c *gin.Context) {
	sys := a.systemResetMethod()
	s := a.shop()
	var plans []model.Plan
	a.S.DB.Where("`show` = ? AND sell = ?", true, true).Order("sort").Find(&plans)
	out := make([]gin.H, 0, len(plans))
	for i := range plans {
		if s.HasCapacity(&plans[i]) {
			out = append(out, planResource(&plans[i], sys))
		}
	}
	resp.OK(c, out)
}

func (a *App) settingJSONList(key string) []map[string]any {
	raw := a.Set.Get(key, "")
	var arr []map[string]any
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &arr)
	}
	return arr
}

func str(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

// guestWebsiteConfig 对应 Guest\WebsiteController::config。
func (a *App) guestWebsiteConfig(c *gin.Context) {
	stats := []gin.H{}
	for _, it := range a.settingJSONList("website_hero_stats") {
		v, l := str(it["value"]), str(it["label"])
		if v == "" && l == "" {
			continue
		}
		stats = append(stats, gin.H{"value": v, "label": l})
	}
	resp.OK(c, gin.H{
		"brand_name":             a.Set.Get("website_brand_name", ""),
		"brand_tagline":          a.Set.Get("website_brand_tagline", ""),
		"brand_description":      a.Set.Get("website_brand_description", ""),
		"announcement_enabled":   boolInt(a.Set.Bool("website_announcement_enabled", false)),
		"announcement_text":      a.Set.Get("website_announcement_text", ""),
		"announcement_link_text": a.Set.Get("website_announcement_link_text", ""),
		"announcement_link_href": a.Set.Get("website_announcement_link_href", ""),
		"hero_title":             a.Set.Get("website_hero_title", ""),
		"hero_subtitle":          a.Set.Get("website_hero_subtitle", ""),
		"hero_stats":             stats,
		"cta_title":              a.Set.Get("website_cta_title", ""),
		"cta_subtitle":           a.Set.Get("website_cta_subtitle", ""),
		"footer_contact":         a.Set.Get("website_footer_contact", ""),
		"downloads": gin.H{
			"windows": a.Set.Get("windows_download_url", ""),
			"macos":   a.Set.Get("macos_download_url", ""),
			"android": a.Set.Get("android_download_url", ""),
			"ios":     a.Set.Get("ios_download_url", ""),
		},
	})
}

// guestDiscoverConfig 对应 Guest\DiscoverController::config。
func (a *App) guestDiscoverConfig(c *gin.Context) {
	resp.OK(c, gin.H{
		"connected_page_url": a.settingOrNil("discover_connected_page_url"),
		"prompt_image_url":   a.settingOrNil("discover_prompt_image_url"),
		"prompt_badge_text":  a.settingOrNil("discover_prompt_badge_text"),
		"prompt_body":        a.settingOrNil("discover_prompt_body"),
		"prompt_button_text": a.settingOrNil("discover_prompt_button_text"),
		"prompt_cancel_text": a.settingOrNil("discover_prompt_cancel_text"),
		"nav_page_url":       a.settingOrNil("discover_nav_page_url"),
	})
}

// guestAntiLossConfig 对应 Guest\AntiLossController::config：没有链接或协议不在白名单的社媒条目会被丢弃。
func (a *App) guestAntiLossConfig(c *gin.Context) {
	socials := []gin.H{}
	for _, it := range a.settingJSONList("anti_loss_socials") {
		u := str(it["url"])
		if u == "" {
			continue
		}
		scheme := ""
		if pu, err := url.Parse(u); err == nil {
			scheme = strings.ToLower(pu.Scheme)
		}
		if scheme != "http" && scheme != "https" && scheme != "tg" && scheme != "" {
			continue
		}
		socials = append(socials, gin.H{"name": str(it["name"]), "url": u, "icon_url": str(it["icon_url"])})
	}
	resp.OK(c, gin.H{
		"qr_data":          a.Set.Get("anti_loss_qr_data", ""),
		"qr_image_url":     a.Set.Get("anti_loss_qr_image_url", ""),
		"qr_logo_url":      a.Set.Get("anti_loss_qr_logo_url", ""),
		"title":            a.Set.Get("anti_loss_title", ""),
		"body":             a.Set.Get("anti_loss_body", ""),
		"save_button_text": a.Set.Get("anti_loss_save_button_text", ""),
		"saved_toast":      a.Set.Get("anti_loss_saved_toast", ""),
		"follow_text":      a.Set.Get("anti_loss_follow_text", ""),
		"socials":          socials,
	})
}

// guestAdBannerConfig 对应 Guest\AdBannerController::config。
func (a *App) guestAdBannerConfig(c *gin.Context) {
	resp.OK(c, gin.H{
		"enabled":           a.Set.Bool("ad_banner_enabled", false),
		"image_url":         a.Set.Get("ad_banner_image_url", ""),
		"click_url":         a.Set.Get("ad_banner_click_url", ""),
		"countdown_seconds": a.Set.Int("ad_banner_countdown_seconds", 5),
	})
}

// clientAppVersion 对应 Client\AppController::getVersion。
func (a *App) clientAppVersion(c *gin.Context) {
	ua := c.GetHeader("User-Agent")
	if strings.Contains(ua, "tidalab/4.0.0") || strings.Contains(ua, "tunnelab/4.0.0") {
		if strings.Contains(ua, "Win64") {
			resp.OK(c, gin.H{"version": a.settingOrNil("windows_version"), "download_url": a.settingOrNil("windows_download_url")})
		} else {
			resp.OK(c, gin.H{"version": a.settingOrNil("macos_version"), "download_url": a.settingOrNil("macos_download_url")})
		}
		return
	}
	data := gin.H{}
	for _, p := range []string{"windows", "macos", "android"} {
		data[p+"_version"] = a.settingOrNil(p + "_version")
		data[p+"_download_url"] = a.settingOrNil(p + "_download_url")
		data[p+"_force_update"] = a.Set.Bool(p+"_force_update", false)
	}
	for _, key := range []string{"update_title", "update_message", "update_button", "update_changelog"} {
		if v := a.Set.Get(key, ""); v != "" {
			data[key] = v
		}
		for _, p := range []string{"windows", "macos", "android", "ios"} {
			if v := a.Set.Get(p+"_"+key, ""); v != "" {
				data[p+"_"+key] = v
			}
		}
	}
	resp.OK(c, data)
}

// clientAppConfig 对应 Client\AppController::getConfig：用 app.clash.yaml 模板生成该用户可用节点的 Clash 配置。
// 只带节点本身——不加「剩余流量/到期」信息节点、协议前缀与订阅域名直连规则（与 PHP 一致）。
func (a *App) clientAppConfig(c *gin.Context) {
	u := cur(c)
	if !userAvailable(u) {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(""))
		return
	}
	servers, err := sub.AvailableServers(c, a.S.DB, a.S.Redis, u)
	if err != nil {
		resp.Fail(c, 500, "生成配置失败")
		return
	}
	ctx := &sub.Ctx{
		User: u, Servers: servers, AppName: a.Set.Get("app_name", "XBoard"),
		Template: func(string) string { return sub.AppClashTemplate() },
	}
	out, err := sub.BuildClash(ctx, sub.ClashKindClash)
	if err != nil {
		resp.Fail(c, 500, "生成配置失败")
		return
	}
	c.Data(http.StatusOK, "text/yaml; charset=utf-8", out.Body)
}

// loginWithMailLink 对应 AuthController::loginWithMailLink：向邮箱发送一次性登录链接。
func (a *App) loginWithMailLink(c *gin.Context) {
	var in struct {
		Email    string `json:"email" form:"email" binding:"required,email"`
		Redirect string `json:"redirect" form:"redirect"`
	}
	if err := c.ShouldBind(&in); err != nil {
		resp.Fail(c, 422, "邮箱格式不正确")
		return
	}
	if !a.Set.Bool("login_with_mail_link_enable", false) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	email := notify.Normalize(in.Email)
	if !a.Set.Bool("user_web_login_enable", true) {
		var u model.User
		if a.S.DB.Where("email = ?", email).First(&u).Error == nil && !u.IsAdmin {
			resp.Fail(c, 403, "Web login is not allowed for this account, please sign in from the app")
			return
		}
	}
	flag := "LAST_SEND_LOGIN_WITH_MAIL_LINK_TIMESTAMP_" + email
	if n, _ := a.S.Redis.Exists(bgCtx(), flag).Result(); n > 0 {
		resp.Fail(c, 429, "Sending frequently, please try again later")
		return
	}
	var u model.User
	if a.S.DB.Where("email = ?", email).First(&u).Error != nil {
		resp.OK(c, true) // 成功但用户不存在，保护用户隐私
		return
	}
	code := auth.RandHex(16)
	a.S.Redis.Set(bgCtx(), "TEMP_TOKEN_"+code, u.ID, 5*time.Minute)
	a.S.Redis.Set(bgCtx(), flag, time.Now().Unix(), time.Minute)
	redirect := in.Redirect
	if redirect == "" {
		redirect = "dashboard"
	}
	path := "/#/login?verify=" + code + "&redirect=" + url.PathEscape(redirect)
	base := strings.TrimRight(a.Set.Get("app_url", ""), "/")
	if base == "" {
		scheme := "http"
		if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		base = scheme + "://" + c.Request.Host
	}
	app := a.Set.Get("app_name", "Lets VPN")
	a.mailQ().Enqueue(jobs.MailTask{To: u.Email, Subject: "Login to " + app, Template: "mailLogin",
		Values: map[string]any{"name": app, "link": base + path, "url": a.Set.Get("app_url", "")}})
	resp.OK(c, true)
}
