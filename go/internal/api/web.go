package api

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"xboard/internal/web"
)

// 网页外壳：对应 routes/web.php 与 resources/views/admin.blade.php、theme/*/dashboard.blade.php。
//   - GET /                 用户端 SPA 外壳（主题 umi.js）
//   - GET /{secure_path}    管理端 SPA 外壳（Vite 构建产物，按 manifest.json 取入口）
//   - /assets、/theme        静态文件（public/assets、public/theme）

const goVersion = "1.0.0-go"

// jsString 输出可安全嵌入 <script> 的 JS 字符串字面量（json.Marshal 默认转义 < > &）。
func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (a *App) registerWeb(r *gin.Engine) {
	r.GET("/", a.userSite)
	r.GET("/assets/*filepath", serveStatic("assets"))
	r.GET("/theme/*filepath", serveStatic("theme"))
}

// serveStatic 提供 /assets 与 /theme 下的静态文件：磁盘（public/）优先，其次二进制内嵌的副本。
// 这样既可以用磁盘文件覆盖 / 追加主题，也能在只有一个二进制时完整工作。
func serveStatic(prefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rel := strings.TrimPrefix(pathpkg.Clean("/"+c.Param("filepath")), "/")
		if rel == "" || strings.Contains(rel, "..") {
			c.Status(http.StatusNotFound)
			return
		}
		disk := filepath.Join(publicDir(), prefix, filepath.FromSlash(rel))
		if st, err := os.Stat(disk); err == nil && !st.IsDir() {
			c.File(disk)
			return
		}
		raw, err := web.ReadFile(prefix + "/" + rel)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		ctype := mime.TypeByExtension(filepath.Ext(rel))
		if ctype == "" {
			ctype = http.DetectContentType(raw)
		}
		c.Header("Cache-Control", "public, max-age=3600")
		c.Data(http.StatusOK, ctype, raw)
	}
}

// themeConfigValues 读取主题配置：先取 config.json 中字段的默认值，再用已保存的设置覆盖。
func (a *App) themeConfigValues(name string) map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile(filepath.Join(themeDir(), name, "config.json"))
	if err != nil {
		raw, err = web.ReadFile("theme/" + name + "/config.json")
	}
	if err == nil {
		var m struct {
			Configs []struct {
				FieldName    string `json:"field_name"`
				DefaultValue any    `json:"default_value"`
			} `json:"configs"`
		}
		if json.Unmarshal(raw, &m) == nil {
			for _, f := range m.Configs {
				if f.DefaultValue != nil {
					out[f.FieldName] = fmt.Sprint(f.DefaultValue)
				}
			}
		}
	}
	if saved := a.Set.Get("theme_"+name, ""); saved != "" {
		var m map[string]any
		if json.Unmarshal([]byte(saved), &m) == nil {
			for k, v := range m {
				if v != nil {
					out[k] = fmt.Sprint(v)
				}
			}
		}
	}
	return out
}

// copyDir 把主题目录复制到 public/theme（PHP 在首次访问时做同样的初始化）。
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

// userSite 对应 Route::get('/')：渲染主题的 dashboard 外壳。
func (a *App) userSite(c *gin.Context) {
	if !a.Set.Bool("user_web_login_enable", true) {
		c.String(http.StatusNotFound, "404 Not Found")
		return
	}
	theme := a.Set.Get("frontend_theme", "Xboard")
	_, onDisk := os.Stat(filepath.Join(themeDir(), theme))
	if onDisk != nil && !web.Exists("theme/"+theme+"/config.json") {
		theme = "Xboard"
		_, onDisk = os.Stat(filepath.Join(themeDir(), theme))
	}
	// 磁盘上有该主题且 public/theme 里还没有时，复制一份（与 PHP 行为一致）；仅内嵌时直接由内嵌副本提供。
	if onDisk == nil {
		pub := filepath.Join(publicDir(), "theme", theme)
		if _, err := os.Stat(pub); err != nil {
			if err := copyDir(filepath.Join(themeDir(), theme), pub); err != nil {
				c.String(http.StatusInternalServerError, "主题初始化失败")
				return
			}
		}
	}
	cfg := a.themeConfigValues(theme)
	title := a.Set.Get("app_name", "Lets VPN")
	page := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,minimum-scale=1,user-scalable=no" />
  <title>%s</title>
  <script type="module" crossorigin src="/theme/%s/assets/umi.js"></script>
</head>
<body>
  <script>
    window.routerBase = "/";
    window.settings = {
      title: %s,
      assets_path: %s,
      theme: { color: %s },
      version: %s,
      background_url: %s,
      description: %s,
      i18n: ['zh-CN','en-US','ja-JP','vi-VN','ko-KR','zh-TW','fa-IR'],
      logo: %s
    }
  </script>
  <div id="app"></div>
  %s
</body>
</html>`,
		html.EscapeString(title), html.EscapeString(theme),
		jsString(title), jsString("/theme/"+theme+"/assets"),
		jsString(firstNonEmpty(cfg["theme_color"], "default")),
		jsString(goVersion), jsString(cfg["background_url"]),
		jsString(a.Set.Get("app_description", "")), jsString(a.Set.Get("logo", "")),
		cfg["custom_html"])
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

type manifestChunk struct {
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	Imports []string `json:"imports"`
	IsEntry bool     `json:"isEntry"`
}

// adminShell 对应 admin.blade.php：从 Vite manifest 收集入口脚本与样式。
func (a *App) adminShell(c *gin.Context) {
	title := a.Set.Get("app_name", "Lets VPN")
	base := filepath.Join(publicDir(), "assets", "admin")
	var scripts, styles []string
	raw, err := os.ReadFile(filepath.Join(base, "manifest.json"))
	if err != nil {
		raw, err = web.ReadFile("assets/admin/manifest.json")
	}
	if err == nil {
		var m map[string]manifestChunk
		if json.Unmarshal(raw, &m) == nil {
			seen := map[string]bool{}
			var walk func(name string)
			walk = func(name string) {
				ch, ok := m[name]
				if !ok || seen[name] {
					return
				}
				seen[name] = true
				styles = append(styles, ch.CSS...)
				for _, imp := range ch.Imports {
					walk(imp)
				}
				if ch.IsEntry && ch.File != "" {
					scripts = append(scripts, ch.File)
				}
			}
			walk("index.html")
		}
	}
	locales, _ := filepath.Glob(filepath.Join(base, "locales", "*.js"))
	if len(locales) == 0 {
		for _, n := range web.DirNames("assets/admin/locales", false) {
			if strings.HasSuffix(n, ".js") {
				locales = append(locales, n)
			}
		}
	}
	sort.Strings(locales)

	var head strings.Builder
	for _, css := range styles {
		fmt.Fprintf(&head, "    <link rel=\"stylesheet\" crossorigin href=\"/assets/admin/%s\" />\n", html.EscapeString(css))
	}
	for _, l := range locales {
		fmt.Fprintf(&head, "    <script src=\"/assets/admin/locales/%s\"></script>\n", html.EscapeString(filepath.Base(l)))
	}
	for _, js := range scripts {
		fmt.Fprintf(&head, "    <script type=\"module\" crossorigin src=\"/assets/admin/%s\"></script>\n", html.EscapeString(js))
	}
	if len(scripts) == 0 {
		head.WriteString("    <!-- 未找到 public/assets/admin/manifest.json：请先构建管理端（admin-src） -->\n")
	}

	page := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>%s</title>
  <script>
    window.settings = {
      base_url: "/",
      title: %s,
      version: %s,
      logo: %s,
      secure_path: %s,
    };
  </script>
%s</head>
<body>
  <div id="root"></div>
</body>
</html>`,
		html.EscapeString(title), jsString(title), jsString(goVersion),
		jsString(a.Set.Get("logo", "")), jsString(a.adminPath(a.AppKey)), head.String())
	// 外壳引用带哈希的入口文件，升级后必须重新取，否则浏览器继续加载旧版前端
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(page))
}

// healthz 供负载均衡 / 容器探活：数据库与 Redis 都可用返回 200，否则 503。
func (a *App) healthz(c *gin.Context) {
	status := http.StatusOK
	out := gin.H{"status": "ok", "version": goVersion}
	if sqlDB, err := a.S.DB.DB(); err != nil || sqlDB.Ping() != nil {
		out["database"], status = "down", http.StatusServiceUnavailable
	} else {
		out["database"] = "up"
	}
	if err := a.S.Redis.Ping(c.Request.Context()).Err(); err != nil {
		out["redis"], status = "down", http.StatusServiceUnavailable
	} else {
		out["redis"] = "up"
	}
	if status != http.StatusOK {
		out["status"] = "degraded"
	}
	c.JSON(status, out)
}
