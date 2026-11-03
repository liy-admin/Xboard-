// Package singbox 负责定位与在线下载 sing-box 可执行文件，对应 PHP 的 SingBoxBinaryService。
//
// 解析顺序：自定义路径 → PATH 中的 sing-box → 托管副本（storage/app/singbox/<版本>/sing-box）。
// 托管副本不存在时从官方 SagerNet GitHub 发布页按本机系统与架构下载、解压并用 `sing-box version` 校验。
package singbox

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	DefaultVersion = "1.14.2"
	defaultBase    = "https://github.com/SagerNet/sing-box/releases/download"
	maxArchiveSize = 200 << 20 // 发布包大小上限
	maxBinarySize  = 300 << 20 // 解压后二进制大小上限
)

// versionRe 限制版本号形态：它会拼进下载地址和磁盘路径，必须防止路径穿越与 URL 注入。
var versionRe = regexp.MustCompile(`^\d+\.\d+\.\d+(?:[-.][0-9A-Za-z]+)*$`)

// ValidVersion 校验版本号（不含前缀 v）。
func ValidVersion(v string) bool { return versionRe.MatchString(v) }

// Status 是下载进度快照，供管理端轮询。
type Status struct {
	State      string `json:"state"` // idle | downloading | done | error
	Version    string `json:"version"`
	Message    string `json:"message"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
	Path       string `json:"path"`
	StartedAt  int64  `json:"started_at"`
	FinishedAt int64  `json:"finished_at"`
}

// Manager 管理托管副本。零值不可用，请用 New 或 Default。
type Manager struct {
	StorageDir string
	BaseURL    string
	HTTP       *http.Client

	mu     sync.Mutex
	status Status
	dl     sync.Mutex // 同一时间只允许一个下载
}

// New 创建管理器；baseURL 为空时使用官方发布页。
func New(storageDir, baseURL string) *Manager {
	if baseURL == "" {
		baseURL = defaultBase
	}
	return &Manager{StorageDir: storageDir, BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP: &http.Client{Timeout: 10 * time.Minute}, status: Status{State: "idle"}}
}

var (
	defOnce sync.Once
	def     *Manager
)

// Default 返回进程级管理器。存储目录与发布页可用环境变量覆盖：
// XBOARD_STORAGE_DIR（默认 storage）、XBOARD_SINGBOX_DOWNLOAD_BASE（镜像或内网地址）。
func Default() *Manager {
	defOnce.Do(func() {
		dir := os.Getenv("XBOARD_STORAGE_DIR")
		if dir == "" {
			dir = "storage"
		}
		def = New(dir, os.Getenv("XBOARD_SINGBOX_DOWNLOAD_BASE"))
	})
	return def
}

// platform 返回发布包名里的 os / arch / 扩展名。
func platform() (osName, arch, ext string, err error) {
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	case "arm":
		arch = "armv7"
	default:
		return "", "", "", fmt.Errorf("不支持的架构: %s", runtime.GOARCH)
	}
	switch runtime.GOOS {
	case "linux", "darwin":
		return runtime.GOOS, arch, "tar.gz", nil
	case "windows":
		return "windows", arch, "zip", nil
	}
	return "", "", "", fmt.Errorf("不支持的系统: %s", runtime.GOOS)
}

func binName() string {
	if runtime.GOOS == "windows" {
		return "sing-box.exe"
	}
	return "sing-box"
}

// AssetName 返回发布包文件名，如 sing-box-1.14.2-linux-amd64.tar.gz。
func AssetName(version string) (string, error) {
	osName, arch, ext, err := platform()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("sing-box-%s-%s-%s.%s", version, osName, arch, ext), nil
}

// DownloadURL 返回某版本在当前平台的下载地址。
func (m *Manager) DownloadURL(version string) (string, error) {
	if !ValidVersion(version) {
		return "", fmt.Errorf("版本号格式不正确: %q", version)
	}
	asset, err := AssetName(version)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/v%s/%s", m.BaseURL, version, asset), nil
}

// BinPath 返回托管副本的路径。
func (m *Manager) BinPath(version string) string {
	return filepath.Join(m.StorageDir, "app", "singbox", version, binName())
}

// Works 运行 `<bin> version`，可用时返回版本号。
func Works(bin string) (version string, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, _ := exec.CommandContext(ctx, bin, "version").CombinedOutput()
	text := string(out)
	i := strings.Index(text, "sing-box version")
	if i < 0 {
		return "", false
	}
	line := text[i+len("sing-box version"):]
	if j := strings.IndexByte(line, '\n'); j >= 0 {
		line = line[:j]
	}
	return strings.TrimSpace(line), true
}

// Status 返回当前下载状态。
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *Manager) setStatus(f func(*Status)) {
	m.mu.Lock()
	f(&m.status)
	m.mu.Unlock()
}

// Start 异步开始下载；已有下载在进行时直接返回当前状态。
func (m *Manager) Start(version string) (Status, error) {
	if !ValidVersion(version) {
		return Status{}, fmt.Errorf("版本号格式不正确: %q", version)
	}
	m.mu.Lock()
	if m.status.State == "downloading" {
		s := m.status
		m.mu.Unlock()
		return s, nil
	}
	m.status = Status{State: "downloading", Version: version, Message: "准备下载", StartedAt: time.Now().Unix()}
	s := m.status
	m.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		_, err := m.Ensure(ctx, version)
		m.setStatus(func(st *Status) {
			st.FinishedAt = time.Now().Unix()
			if err != nil {
				st.State, st.Message = "error", err.Error()
				return
			}
			st.State, st.Message, st.Path = "done", "下载完成，sing-box 已就绪", m.BinPath(version)
		})
	}()
	return s, nil
}

// Ensure 确保托管副本存在并可运行，返回其路径。并发调用会排队，后到者直接复用先到者的结果。
func (m *Manager) Ensure(ctx context.Context, version string) (string, error) {
	if !ValidVersion(version) {
		return "", fmt.Errorf("版本号格式不正确: %q", version)
	}
	dest := m.BinPath(version)
	if _, ok := Works(dest); ok {
		return dest, nil
	}
	m.dl.Lock()
	defer m.dl.Unlock()
	if _, ok := Works(dest); ok { // 排队期间别的调用已下载好
		return dest, nil
	}
	if err := m.download(ctx, version, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func (m *Manager) progress(msg string, downloaded, total int64) {
	m.setStatus(func(st *Status) {
		if st.State == "downloading" {
			st.Message, st.Downloaded, st.Total = msg, downloaded, total
		}
	})
}

func (m *Manager) download(ctx context.Context, version, dest string) error {
	url, err := m.DownloadURL(version)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	asset, _ := AssetName(version)
	tmp := filepath.Join(dir, asset+".part")
	defer os.Remove(tmp)

	m.progress("正在下载 "+asset, 0, 0)
	if err := m.fetch(ctx, url, tmp); err != nil {
		return err
	}
	m.progress("正在解压", 0, 0)
	extracted := dest + ".new"
	defer os.Remove(extracted)
	if strings.HasSuffix(asset, ".zip") {
		err = extractZip(tmp, binName(), extracted)
	} else {
		err = extractTarGz(tmp, binName(), extracted)
	}
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(extracted, 0o755); err != nil {
			return err
		}
	}
	// 先校验再替换：损坏或不匹配的二进制不会覆盖已有可用副本。
	got, ok := Works(extracted)
	if !ok {
		return errors.New("下载的 sing-box 无法运行（可能与本机系统或架构不匹配）")
	}
	if !strings.Contains(got, version) {
		return fmt.Errorf("版本号不匹配：期望 %s，实际 %s", version, got)
	}
	_ = os.Remove(dest)
	if err := os.Rename(extracted, dest); err != nil {
		return fmt.Errorf("安装失败: %w", err)
	}
	return nil
}

func (m *Manager) fetch(ctx context.Context, url, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "xboard-go")
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("下载失败: 发布页没有该版本或该平台的安装包 (HTTP 404)")
		}
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	total := resp.ContentLength
	if total > maxArchiveSize {
		return errors.New("安装包超过大小上限")
	}
	var done int64
	buf := make([]byte, 64<<10)
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			done += int64(n)
			if done > maxArchiveSize {
				return errors.New("安装包超过大小上限")
			}
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			if time.Since(last) > 300*time.Millisecond {
				m.progress("正在下载", done, total)
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("下载中断: %w", rerr)
		}
	}
	if done < 1024 {
		return errors.New("下载内容过小，可能不是有效的安装包")
	}
	m.progress("下载完成", done, total)
	return nil
}

// extractTarGz 只提取名为 name 的单个文件到 out；不展开其他条目，因此不存在路径穿越。
func extractTarGz(archive, name, out string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return errors.New("安装包中没有找到 sing-box 可执行文件")
		}
		if err != nil {
			return fmt.Errorf("解压失败: %w", err)
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == name {
			return writeLimited(out, tr)
		}
	}
}

func extractZip(archive, name, out string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}
	defer zr.Close()
	for _, zf := range zr.File {
		if !zf.FileInfo().IsDir() && filepath.Base(zf.Name) == name {
			rc, err := zf.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			return writeLimited(out, rc)
		}
	}
	return errors.New("安装包中没有找到 sing-box 可执行文件")
}

func writeLimited(out string, r io.Reader) error {
	f, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(r, maxBinarySize+1))
	if err != nil {
		return err
	}
	if n > maxBinarySize {
		return errors.New("解压后的文件超过大小上限")
	}
	return nil
}

// Resolve 按「自定义路径 → PATH → 托管副本（必要时自动下载）」的顺序找可用的 sing-box。
// auto 为 true 且没有可用副本时，会在调用线程里下载（探测任务中使用，ctx 控制超时）。
func (m *Manager) Resolve(ctx context.Context, custom, version string, auto bool) (bin, ver, errMsg string) {
	custom = strings.TrimSpace(custom)
	last := ""
	try := func(c string) bool {
		if v, ok := Works(c); ok {
			bin, ver = c, v
			return true
		}
		last = "无法运行 " + c
		return false
	}
	if custom != "" && custom != "sing-box" && try(custom) {
		return bin, ver, ""
	}
	if try("sing-box") {
		return bin, ver, ""
	}
	if version == "" {
		version = DefaultVersion
	}
	if !ValidVersion(version) {
		return "", "", fmt.Sprintf("版本号格式不正确: %q", version)
	}
	if p := m.BinPath(version); fileExists(p) && try(p) {
		return bin, ver, ""
	}
	if !auto {
		return "", "", last + "；自动下载已关闭"
	}
	p, err := m.Ensure(ctx, version)
	if err != nil {
		return "", "", err.Error()
	}
	v, _ := Works(p)
	return p, v, ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
