package api

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/auth"
	"xboard/internal/jobs"
	"xboard/internal/model"
	"xboard/internal/resp"
	"xboard/internal/setting"
	"xboard/internal/store"
)

type App struct {
	S *store.Store
	// TrustedProxies 为可信反向代理（X-Forwarded-For 只在来自这些地址时才被采信）；为空表示不信任任何代理头。
	TrustedProxies []string
	Set            *setting.Service
	AppKey         string

	adminOnce sync.Once

	mailq     *jobs.MailQueue
	mailqOnce sync.Once

	lastBase atomic.Value

	hub     *wsHub
	hubOnce sync.Once
	admin   *gin.Engine
}

const userKey = "user"

func (a *App) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(a.rememberBase)
	if a.TrustedProxies != nil {
		_ = r.SetTrustedProxies(a.TrustedProxies)
	}
	r.GET("/healthz", a.healthz)

	// 路径与 Laravel V1/V2 保持一致，现有前端与节点端无需改动。
	for _, v := range []string{"/api/v1", "/api/v2"} {
		g := r.Group(v)

		pp := g.Group("/passport")
		pp.POST("/auth/login", a.login)
		pp.POST("/auth/loginWithMailLink", a.loginWithMailLink)
		if v == "/api/v1" {
			pp.POST("/comm/sendEmailVerify", a.sendEmailVerify)
			pp.POST("/comm/pv", a.invitePV)
			pp.POST("/auth/register", a.register)
			pp.POST("/auth/forget", a.forget)
			pp.GET("/auth/token2Login", a.token2Login)
			pp.POST("/auth/getQuickLoginUrl", a.passportQuickLoginURL)
			pp.POST("/auth/deviceTrial", a.deviceTrialClaim)
		}

		u := g.Group("/user", a.userAuth)
		u.GET("/info", a.userInfo)
		u.GET("/checkLogin", a.checkLogin)
		u.GET("/getSubscribe", a.getSubscribe)
		u.GET("/resetSecurity", a.resetSecurity)
		u.POST("/changePassword", a.changePassword)
		u.GET("/plan/fetch", a.planFetchV1)
		a.registerOrderRoutes(u)
		if v == "/api/v1" {
			a.registerUserExt(u)
		}
	}

	a.registerClientRoutes(r)
	a.registerServerV2(r)
	a.wsRegister(r)
	a.registerTelegramRoutes(r)
	a.registerBalala(r)
	r.Any("/api/v1/guest/payment/notify/:method/:uuid", a.guestPaymentNotify)
	r.GET("/api/v1/guest/server/countries", a.guestServerCountries)
	// 支付方式图标等上传文件（对应 public/uploads）。
	r.Static("/uploads", publicDir()+"/uploads")
	// 用户端 / 管理端网页外壳与静态资源（/、/assets、/theme）。
	a.registerWeb(r)
	a.registerGuestRoutes(r)
	// 后台路径由 secure_path 决定，无法静态注册，统一走 NoRoute 分发。
	r.NoRoute(a.adminDispatch)

	up := r.Group("/api/v1/server/UniProxy", a.serverAuth)
	up.GET("config", a.nodeConfig)
	up.GET("user", a.nodeUsers)
	up.POST("push", a.nodePush)
	up.POST("alive", a.nodeAlive)
	up.GET("alivelist", a.nodeAliveList)
	up.POST("status", a.nodeStatus)
	return r
}

func (a *App) userAuth(c *gin.Context) {
	u := auth.UserFromBearer(a.S.DB, c.GetHeader("Authorization"))
	if u == nil {
		resp.Fail(c, 403, "未登录或登陆已过期")
		return
	}
	c.Set(userKey, u)
	c.Next()
}

func cur(c *gin.Context) *model.User { return c.MustGet(userKey).(*model.User) }

func (a *App) adminAuth(c *gin.Context) {
	if !cur(c).IsAdmin {
		resp.Fail(c, 403, "Unauthorized")
	}
}

func now() int64 { return time.Now().Unix() }

func lower(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// mailQ 返回进程内的邮件队列（惰性创建，StartWS 时启动消费）。
func (a *App) mailQ() *jobs.MailQueue {
	a.mailqOnce.Do(func() { a.mailq = jobs.NewMailQueue(a.jobsEnv(), 2, 3) })
	return a.mailq
}
