package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// 订阅抓取插件 balala_sub：调用插件自带的 Python 脚本（DrissionPage 驱动浏览器登录 balala.io）。
// 对应 plugins/BalalaSub/Commands/FetchCommand.php。

const balalaCacheKey = "plugin:balala_sub:latest"

// BalalaConfig 为插件配置（已合并默认值）。
type BalalaConfig struct {
	SiteURL    string
	Email      string
	Password   string
	Cookie     string
	PythonBin  string
	ChromePath string
	Headless   bool
	Timeout    int
	CacheTTL   int
}

// balalaScriptPath 返回抓取脚本路径（XBOARD_BALALA_SCRIPT 可覆盖）。
func balalaScriptPath() string {
	if p := os.Getenv("XBOARD_BALALA_SCRIPT"); p != "" {
		return p
	}
	for _, d := range []string{"plugins", "plugins-core"} {
		p := filepath.Join(d, "BalalaSub", "python", "fetch_sub.py")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join("plugins", "BalalaSub", "python", "fetch_sub.py")
}

// LoadBalalaConfig 从 plugins 表读取 balala_sub 的配置，缺省值与 config.json 一致。
func (e *Env) LoadBalalaConfig() BalalaConfig {
	cfg := BalalaConfig{SiteURL: "https://balala.io", PythonBin: ResolvePython(""), Headless: true, Timeout: 120, CacheTTL: 86400}
	var raw string
	if e.DB.Raw("SELECT config FROM v2_plugins WHERE code = ?", "balala_sub").Row().Scan(&raw) != nil || raw == "" {
		return cfg
	}
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return cfg
	}
	str := func(k string) string {
		if v, ok := m[k]; ok && v != nil {
			return fmt.Sprint(v)
		}
		return ""
	}
	if v := str("site_url"); v != "" {
		cfg.SiteURL = v
	}
	cfg.Email, cfg.Password, cfg.Cookie = str("email"), str("password"), str("cookie")
	// 默认的 "python" 在 Linux 上通常不存在，统一经 ResolvePython 解析（与 PHP 的 EnvDoctor 一致）。
	cfg.PythonBin = ResolvePython(str("python_bin"))
	cfg.ChromePath = str("chrome_path")
	if v, ok := m["headless"]; ok {
		cfg.Headless = balalaTruthy(v)
	}
	if v, err := strconv.Atoi(str("timeout")); err == nil && v > 0 {
		cfg.Timeout = v
	}
	if v, err := strconv.Atoi(str("cache_ttl")); err == nil && v > 0 {
		cfg.CacheTTL = v
	}
	return cfg
}

// BalalaResult 为抓取脚本的输出：主订阅链接与按客户端区分的订阅列表。
type BalalaResult struct {
	SubscribeURL  string
	Subscriptions []map[string]any
}

// BalalaFetch 执行抓取并返回主订阅链接；成功结果写入 Redis 缓存。
func (e *Env) BalalaFetch(ctx context.Context, cfg BalalaConfig) (string, error) {
	r, err := e.BalalaFetchFull(ctx, cfg)
	return r.SubscribeURL, err
}

// BalalaFetchFull 同 BalalaFetch，但同时返回各客户端的订阅列表（自动导入按客户端选链接）。
func (e *Env) BalalaFetchFull(ctx context.Context, cfg BalalaConfig) (BalalaResult, error) {
	var zero BalalaResult
	if cfg.Cookie == "" && (cfg.Email == "" || cfg.Password == "") {
		return zero, fmt.Errorf("缺少登录凭据：请填写「会话 Cookie」，或「登录邮箱 + 密码」")
	}
	script := balalaScriptPath()
	if _, err := os.Stat(script); err != nil {
		return zero, fmt.Errorf("找不到抓取脚本：%s", script)
	}
	outDir := filepath.Join("storage", "app", "balala_sub")
	_ = os.MkdirAll(outDir, 0o775)
	outFile := filepath.Join(outDir, fmt.Sprintf("fetch_result_%d.json", time.Now().UnixNano()))
	defer os.Remove(outFile)

	headless := "0"
	if cfg.Headless {
		headless = "1"
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Timeout+30)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, cfg.PythonBin, script)
	cmd.Env = append(os.Environ(),
		"BALALA_URL="+strings.TrimRight(cfg.SiteURL, "/"),
		"BALALA_EMAIL="+cfg.Email,
		"BALALA_PASSWORD="+cfg.Password,
		"BALALA_COOKIE="+cfg.Cookie,
		"BALALA_HEADLESS="+headless,
		"BALALA_TIMEOUT="+strconv.Itoa(cfg.Timeout),
		"BALALA_CHROME_PATH="+cfg.ChromePath,
		"BALALA_OUTFILE="+outFile,
		"PYTHONIOENCODING=utf-8",
	)
	stdout, runErr := cmd.CombinedOutput()
	raw, readErr := os.ReadFile(outFile)
	body := strings.TrimSpace(string(raw))
	if readErr != nil || body == "" {
		body = strings.TrimSpace(string(stdout))
	}
	var res struct {
		Success       bool             `json:"success"`
		SubscribeURL  string           `json:"subscribe_url"`
		Subscriptions []map[string]any `json:"subscriptions"`
		Error         string           `json:"error"`
	}
	if json.Unmarshal([]byte(body), &res) != nil {
		if runErr != nil {
			return zero, fmt.Errorf("调用 Python 失败：%v", runErr)
		}
		return zero, fmt.Errorf("抓取结果无法解析")
	}
	if !res.Success || res.SubscribeURL == "" {
		if res.Error != "" {
			return zero, fmt.Errorf("%s", res.Error)
		}
		return zero, fmt.Errorf("抓取失败")
	}
	if e.Redis != nil {
		b, _ := json.Marshal(map[string]any{"success": true, "subscribe_url": res.SubscribeURL,
			"subscriptions": res.Subscriptions, "fetched_at": time.Now().Unix()})
		_ = e.Redis.Set(ctx, balalaCacheKey, string(b), time.Duration(cfg.CacheTTL)*time.Second)
	}
	return BalalaResult{SubscribeURL: res.SubscribeURL, Subscriptions: res.Subscriptions}, nil
}

// balalaTruthy 与 PHP 的 (bool) 转换一致：字符串 "0"、"" 为假。
func balalaTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != "" && x != "0"
	case float64:
		return x != 0
	default:
		return true
	}
}

// BalalaLatest 返回最近一次抓取的缓存结果。
func (e *Env) BalalaLatest(ctx context.Context) (string, bool) {
	if e.Redis == nil {
		return "", false
	}
	raw, ok, err := e.Redis.Get(ctx, balalaCacheKey)
	if err != nil || !ok {
		return "", false
	}
	return raw, true
}

// resolveFirefox 与 EnvDoctor::resolveFirefox 一致：配置的 chrome_path 优先，其次已上传内核目录里的 firefox。
func resolveFirefox(cfg BalalaConfig) string {
	if cfg.ChromePath != "" {
		return cfg.ChromePath
	}
	var found string
	_ = filepath.Walk(filepath.Join("storage", "app", "balala_sub", "firefox"), func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && (info.Name() == "firefox" || info.Name() == "firefox.exe") && found == "" {
			found = p
		}
		return nil
	})
	return found
}

func probeCmd(ctx context.Context, name string, args ...string) (bool, string) {
	c, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(c, name, args...).CombinedOutput()
	return err == nil, strings.TrimSpace(string(out))
}

// BalalaDoctor 对应 EnvDoctor::run：逐项检查抓取环境，full 时再做一次真实抓取。
// 返回 {ok, checks:[{name, ok, detail}]}。
func (e *Env) BalalaDoctor(ctx context.Context, cfg BalalaConfig, full bool) map[string]any {
	type check struct {
		Name   string `json:"name"`
		OK     bool   `json:"ok"`
		Detail string `json:"detail"`
	}
	var checks []check
	py := cfg.PythonBin
	pyOK, pyOut := probeCmd(ctx, py, "--version")
	if pyOK {
		checks = append(checks, check{"Python", true, pyOut + " (" + py + ")"})
	} else {
		checks = append(checks, check{"Python", false, "无法运行 " + py + "：" + pyOut})
	}
	if pyOK {
		ok, out := probeCmd(ctx, py, "-c", `import ruyipage,sys; sys.stdout.write(getattr(ruyipage,"__version__","?"))`)
		d := "未安装，请先点「安装环境」"
		if ok {
			d = "已安装 " + out
		}
		checks = append(checks, check{"ruyiPage", ok, d})
	} else {
		checks = append(checks, check{"ruyiPage", false, "Python 不可用，跳过"})
	}
	fx := resolveFirefox(cfg)
	_, statErr := os.Stat(fx)
	fxExists := fx != "" && statErr == nil
	if fxExists {
		checks = append(checks, check{"Firefox 路径", true, fx})
	} else {
		checks = append(checks, check{"Firefox 路径", false, "未找到 Firefox，可在配置上传内核或填写路径"})
	}
	if !full {
		if fxExists {
			ok, out := probeCmd(ctx, fx, "--version")
			d := out
			if !ok {
				d = "启动失败（多半缺运行库）：" + out
			}
			checks = append(checks, check{"Firefox 可启动", ok, d})
		} else {
			checks = append(checks, check{"Firefox 可启动", false, "无 Firefox，跳过"})
		}
	}
	hasCookie := strings.TrimSpace(cfg.Cookie) != ""
	hasPwd := strings.TrimSpace(cfg.Email) != "" && strings.TrimSpace(cfg.Password) != ""
	switch {
	case hasCookie:
		checks = append(checks, check{"登录凭据", true, "已配置 Cookie"})
	case hasPwd:
		checks = append(checks, check{"登录凭据", true, "已配置邮箱+密码"})
	default:
		checks = append(checks, check{"登录凭据", false, "缺少邮箱+密码或 Cookie"})
	}
	if full {
		if r, err := e.BalalaFetchFull(ctx, cfg); err != nil {
			checks = append(checks, check{"真实抓取", false, err.Error()})
		} else {
			checks = append(checks, check{"真实抓取", true, "已获取订阅：" + r.SubscribeURL})
		}
	}
	all := true
	for _, c := range checks {
		all = all && c.OK
	}
	return map[string]any{"ok": all, "checks": checks}
}
