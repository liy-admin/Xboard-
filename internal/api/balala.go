package api

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"xboard/internal/resp"
)

// BalalaSub 插件的管理端接口，对应 plugins/BalalaSub/Controllers/SubController.php。
// 路由前缀 /api/v1/balala-sub，全部需要管理员身份（PHP 中为 admin 中间件）。
// upload-firefox 支持 .zip 与 .tar.gz / .tgz；.tar.xz 依赖外部 xz，Go 版不支持。

func (a *App) registerBalala(r *gin.Engine) {
	g := r.Group("/api/v1/balala-sub", a.adminAuthMW)
	g.GET("/latest", a.balalaLatest)
	g.POST("/refresh", a.balalaRefresh)
	g.GET("/doctor", a.balalaDoctor)
	g.POST("/upload-firefox", a.balalaUploadFirefox)
}

func (a *App) balalaEnabled(c *gin.Context) bool {
	if !a.pluginEnabled("balala_sub") {
		resp.Fail(c, 400, "插件未启用")
		return false
	}
	return true
}

// balalaLatest GET /latest：读取最近一次抓取的缓存。
func (a *App) balalaLatest(c *gin.Context) {
	if !a.balalaEnabled(c) {
		return
	}
	raw, ok := a.jobsEnv().BalalaLatest(c.Request.Context())
	if !ok {
		resp.Fail(c, 404, "暂无缓存，请先执行 xboard balala-sub:fetch")
		return
	}
	var data any
	if json.Unmarshal([]byte(raw), &data) != nil {
		resp.Fail(c, 500, "缓存数据损坏")
		return
	}
	resp.OK(c, data)
}

// balalaRefresh POST /refresh：立即抓取一次。
func (a *App) balalaRefresh(c *gin.Context) {
	if !a.balalaEnabled(c) {
		return
	}
	env := a.jobsEnv()
	url, err := env.BalalaFetch(c.Request.Context(), env.LoadBalalaConfig())
	if err != nil {
		resp.Fail(c, 500, err.Error())
		return
	}
	resp.OK(c, gin.H{"success": true, "subscribe_url": url})
}

// balalaDoctor GET /doctor：环境自检，full=1 时附带一次真实抓取。
func (a *App) balalaDoctor(c *gin.Context) {
	if !a.balalaEnabled(c) {
		return
	}
	full := c.Query("full")
	env := a.jobsEnv()
	resp.OK(c, env.BalalaDoctor(c.Request.Context(), env.LoadBalalaConfig(), full == "1" || full == "true"))
}

// balalaUploadFirefox POST /upload-firefox：解压 Firefox 内核，定位可执行文件并写入插件配置 chrome_path。
func (a *App) balalaUploadFirefox(c *gin.Context) {
	if !a.balalaEnabled(c) {
		return
	}
	const maxSize = 512 << 20
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
	fh, err := c.FormFile("file")
	if err != nil {
		resp.Fail(c, 422, "请上传 file 字段")
		return
	}
	if fh.Size > maxSize {
		resp.Fail(c, 422, "文件超过 512MB")
		return
	}
	name := strings.ToLower(fh.Filename)
	isZip := strings.HasSuffix(name, ".zip")
	isTarGz := strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tgz")
	isTarOther := strings.HasSuffix(name, ".tar.xz") || strings.HasSuffix(name, ".tar.bz2") || strings.HasSuffix(name, ".tar")
	if !isZip && !isTarGz && !isTarOther {
		resp.Fail(c, 400, "请上传 .zip 或 .tar.xz/.tar.gz 格式的 Firefox 内核包")
		return
	}
	f, err := fh.Open()
	if err != nil {
		resp.Fail(c, 500, "读取上传文件失败")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		resp.Fail(c, 500, "读取上传文件失败")
		return
	}

	target := filepath.Join("storage", "app", "balala_sub", "firefox")
	if err := os.RemoveAll(target); err != nil {
		resp.Fail(c, 500, "清理旧内核失败："+err.Error())
		return
	}
	if err := os.MkdirAll(target, 0o775); err != nil {
		resp.Fail(c, 500, "创建目录失败："+err.Error())
		return
	}
	if isZip {
		if _, err := unzipSafe(data, target); err != nil {
			resp.Fail(c, 400, "压缩包无法打开，可能已损坏："+err.Error())
			return
		}
	} else if isTarGz {
		if err := untarGzSafe(data, target); err != nil {
			resp.Fail(c, 400, "tar.gz 解压失败："+err.Error())
			return
		}
	} else if err := extractWithSystemTar(data, name, target); err != nil {
		resp.Fail(c, 500, "tar 解压失败："+err.Error())
		return
	}

	exe := locateFirefox(target)
	if exe == "" {
		resp.Fail(c, 400, "解压后未找到 firefox.exe / firefox 可执行文件")
		return
	}
	if err := a.setPluginConfigKey("balala_sub", "chrome_path", exe); err != nil {
		resp.Fail(c, 500, "保存配置失败："+err.Error())
		return
	}
	resp.OK(c, gin.H{"path": exe, "message": "Firefox 内核已上传并设为抓取使用"})
}

// setPluginConfigKey 读取插件配置 JSON，写入单个键后回写（与 PHP 的 updateConfig 合并语义一致）。
func (a *App) setPluginConfigKey(code, key string, value any) error {
	var raw string
	_ = a.S.DB.Raw("SELECT config FROM v2_plugins WHERE code = ?", code).Row().Scan(&raw)
	cur := map[string]any{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &cur)
	}
	cur[key] = value
	b, err := json.Marshal(cur)
	if err != nil {
		return err
	}
	return a.S.DB.Exec("UPDATE v2_plugins SET config = ?, updated_at = NOW() WHERE code = ?", string(b), code).Error
}

// extractWithSystemTar 把 .tar.xz / .tar.bz2 / .tar 交给系统 tar（xz 需要 xz-utils），与 PHP 行为一致。
func extractWithSystemTar(data []byte, name, dest string) error {
	tmp, err := os.CreateTemp("", "balala-firefox-*"+filepath.Ext(name))
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tar", "-xf", tmp.Name(), "-C", dest).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// untarGzSafe 解压 tar.gz，拒绝绝对路径与 .. 穿越。
func untarGzSafe(data []byte, dest string) error {
	gz, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	root, _ := filepath.Abs(dest)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		clean := strings.ReplaceAll(h.Name, "\\", "/")
		if strings.HasPrefix(clean, "/") || strings.Contains(clean, "../") || clean == ".." {
			return fmt.Errorf("非法路径：%s", h.Name)
		}
		p := filepath.Join(root, filepath.FromSlash(clean))
		if p != root && !strings.HasPrefix(p, root+string(os.PathSeparator)) {
			return fmt.Errorf("非法路径：%s", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(p, 0o775); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(p), 0o775); err != nil {
				return err
			}
			out, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs0755(h.FileInfo().Mode()))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

// fs0755 保留压缩包中的可执行位，缺省 0644。
func fs0755(m os.FileMode) os.FileMode {
	if m&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

// locateFirefox 递归查找 firefox.exe（优先）或 firefox，返回绝对路径。
func locateFirefox(dir string) string {
	var found string
	var fallback string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		switch info.Name() {
		case "firefox.exe":
			if found == "" {
				found = p
			}
		case "firefox":
			if fallback == "" {
				fallback = p
			}
		}
		return nil
	})
	if found == "" {
		found = fallback
	}
	if found == "" {
		return ""
	}
	abs, err := filepath.Abs(found)
	if err != nil {
		return found
	}
	return abs
}
