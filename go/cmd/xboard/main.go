// xboard 是面板后端的单一可执行文件。
//
//	xboard [serve]            启动 HTTP 服务；默认同时运行内置调度器（XBOARD_SCHEDULER=0 可关闭）；启动前自动建库建表（XBOARD_AUTO_MIGRATE=0 可关闭）
//	xboard schedule           仅运行内置调度器（多实例部署时可只在一台机器上开）
//	xboard migrate            建库建表（幂等；已有数据不动）
//	xboard install [邮箱]     migrate + 创建管理员（已有管理员时跳过），并打印后台地址
//	xboard admin:create 邮箱 [密码]   创建管理员；邮箱已存在则提升为管理员
//	xboard keygen             生成一个 APP_KEY
//	xboard version            打印版本
//	xboard <job>              立即执行一次定时任务，例如 check:order、reset:traffic
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/api"
	"xboard/internal/config"
	"xboard/internal/jobs"
	"xboard/internal/schema"
	"xboard/internal/setting"
	"xboard/internal/setup"
	"xboard/internal/shop"
	"xboard/internal/store"
)

// version 由构建脚本通过 -ldflags "-X main.version=..." 注入。
var version = "dev"

func usage() {
	fmt.Fprintln(os.Stderr, "用法: xboard <命令>")
	fmt.Fprintln(os.Stderr, "命令:")
	for _, l := range []string{
		"serve              启动 HTTP 服务（默认）",
		"schedule           仅运行调度器",
		"migrate            建库建表（幂等）",
		"install [邮箱]     迁移并创建管理员",
		"admin:create 邮箱 [密码]",
		"keygen             生成 APP_KEY",
		"version            打印版本",
	} {
		fmt.Fprintln(os.Stderr, "  ", l)
	}
	fmt.Fprintln(os.Stderr, "任务（立即执行一次）:")
	for _, name := range jobs.Names() {
		fmt.Fprintln(os.Stderr, "  ", name)
	}
	os.Exit(2)
}

func main() {
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "version", "--version", "-v":
		fmt.Println("xboard-go", version)
		return
	case "keygen":
		fmt.Println(setup.NewAppKey())
		return
	case "help", "--help", "-h":
		usage()
	}
	if os.Getenv("XBOARD_DEBUG") != "1" {
		gin.SetMode(gin.ReleaseMode)
	}

	known := map[string]bool{"serve": true, "schedule": true, "migrate": true, "install": true, "admin:create": true}
	if !known[cmd] && !jobs.IsCLI(cmd) {
		usage()
	}

	cfg := config.Load()

	// 迁移不要求库已存在，必须在打开连接之前处理。
	// serve / schedule 启动时也自动建库建表（幂等，已有数据不动）；XBOARD_AUTO_MIGRATE=0 可关闭。
	auto := (cmd == "serve" || cmd == "schedule") && os.Getenv("XBOARD_AUTO_MIGRATE") != "0"
	if cmd == "migrate" || cmd == "install" || auto {
		res, err := schema.Migrate(cfg)
		if err != nil {
			log.Fatalf("migrate: %v", err)
		}
		state := "已是最新（未改动）"
		if res.Created > 0 {
			state = fmt.Sprintf("新建 %d 张表", res.Created)
		}
		fmt.Printf("数据库 %s 就绪：共 %d 张表，%s\n", cfg.DBName, res.Tables, state)
		if cmd == "migrate" {
			return
		}
		if auto && res.Fresh {
			fmt.Println("全新数据库：请执行 xboard admin:create 邮箱 创建管理员。")
		}
	}

	st, err := store.Open(cfg)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	set := setting.New(st.DB)

	switch cmd {
	case "install":
		runInstall(cfg, st, set)
		return
	case "admin:create":
		if len(os.Args) < 3 {
			log.Fatal("用法: xboard admin:create 邮箱 [密码]")
		}
		pw := ""
		if len(os.Args) > 3 {
			pw = os.Args[3]
		}
		printAdmin(cfg, st, set, os.Args[2], pw)
		return
	}

	env := &jobs.Env{
		DB:    st.DB,
		Redis: jobs.NewRedisAdapter(st.Redis),
		Set:   set,
		Shop:  &shop.Service{DB: st.DB, Set: set},
		Push:  jobs.NewRedisPusher(st.Redis),
		Now:   time.Now,
	}

	switch cmd {
	case "serve":
		if cfg.AppKey == "" {
			log.Printf("警告：APP_KEY 未设置。后台路径、加密订阅密钥等依赖它，请先执行 xboard keygen 并写入 .env")
		}
		if os.Getenv("XBOARD_SCHEDULER") != "0" {
			go runScheduler(st, env)
		}
		app := &api.App{S: st, Set: set, AppKey: cfg.AppKey, TrustedProxies: cfg.TrustedProxies}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		app.StartWS(ctx)
		app.SyncTelegramCommands()
		log.Printf("xboard-go %s listening on %s (timezone %s)", version, cfg.Listen, time.Local)
		log.Printf("管理后台: /%s", app.AdminPath())
		// 收到 SIGINT/SIGTERM 后优雅退出：停止接收新连接，给在途请求最多 10 秒。
		srv := &http.Server{Addr: cfg.Listen, Handler: app.Router(), ReadHeaderTimeout: 15 * time.Second}
		go func() {
			<-ctx.Done()
			shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := srv.Shutdown(shCtx); err != nil {
				log.Printf("shutdown: %v", err)
			}
		}()
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		log.Printf("xboard-go stopped")
	case "schedule":
		runScheduler(st, env)
	default:
		if err := env.Run(cmd); err != nil {
			log.Fatalf("%s: %v", cmd, err)
		}
	}
}

func runInstall(cfg *config.Config, st *store.Store, set *setting.Service) {
	email := os.Getenv("ADMIN_EMAIL")
	if email == "" {
		email = os.Getenv("ADMIN_ACCOUNT") // 与 PHP xboard:install 的环境变量名一致
	}
	if len(os.Args) > 2 {
		email = os.Args[2]
	}
	if setup.HasAdmin(st.DB) && email == "" {
		fmt.Println("已存在管理员，跳过创建。")
		printAdminURL(cfg, st, set)
		return
	}
	if email == "" {
		email = "admin@demo.com"
	}
	printAdmin(cfg, st, set, email, os.Getenv("ADMIN_PASSWORD"))
}

func printAdmin(cfg *config.Config, st *store.Store, set *setting.Service, email, pw string) {
	final, created, err := setup.CreateAdmin(st.DB, email, pw)
	if err != nil {
		log.Fatal(err)
	}
	switch {
	case created:
		fmt.Printf("管理员已创建：\n  账号: %s\n  密码: %s\n（请登录后立即修改密码）\n", email, final)
	case final != "":
		fmt.Printf("管理员 %s 已存在：已提升为管理员并重置密码：%s\n", email, final)
	default:
		fmt.Printf("管理员 %s 已存在：已提升为管理员（密码未改）\n", email)
	}
	printAdminURL(cfg, st, set)
}

func printAdminURL(cfg *config.Config, st *store.Store, set *setting.Service) {
	app := &api.App{S: st, Set: set, AppKey: cfg.AppKey}
	fmt.Printf("后台地址: http(s)://你的站点/%s\n", app.AdminPath())
	if set.Get("app_url", "") == "" {
		fmt.Println("提示：登录后台后请先在「系统配置 → 站点」填写站点地址（支付回调、订阅链接、邮件链接都依赖它）。")
	}
}

// runScheduler 阻塞运行内置调度器，收到 SIGINT/SIGTERM 后退出。
func runScheduler(st *store.Store, env *jobs.Env) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("scheduler started")
	jobs.NewScheduler(env, st.Redis).Run(ctx)
}
