package jobs

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BalalaSub 的抓取环境：解析 Python、安装 ruyiPage（对应 EnvDoctor::resolvePython、SetupCommand、Plugin::bootstrapEnv）。

// runCmd 执行外部命令并返回合并输出；测试里替换成假的，避免真的去 pip 安装。
var runCmd = func(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// ResolvePython 对应 EnvDoctor::resolvePython：配置里填了自定义命令就用它；
// 没填或是默认的 "python" 时，Linux / macOS 优先 python3（多数发行版没有 python 命令），Windows 用 python。
func ResolvePython(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" && configured != "python" {
		return configured
	}
	if runtime.GOOS == "windows" {
		return "python"
	}
	return "python3"
}

// pipInstallRuyipage 在指定 Python 里 pip 安装 ruyiPage，遇到 PEP 668（externally-managed-environment）时
// 自动加 --break-system-packages 重试。返回命令输出与错误。
func pipInstallRuyipage(ctx context.Context, python string) (string, error) {
	base := []string{"-m", "pip", "install", "--upgrade", "ruyiPage"}
	out, err := runCmd(ctx, python, base...)
	if err == nil {
		return out, nil
	}
	low := strings.ToLower(out)
	if strings.Contains(low, "externally-managed") || strings.Contains(low, "break-system-packages") {
		out2, err2 := runCmd(ctx, python, append(append([]string{}, base...), "--break-system-packages")...)
		return out + "\n--- 检测到 externally-managed，改用 --break-system-packages 重试 ---\n" + out2, err2
	}
	return out, err
}

// BalalaSetup 对应 balala-sub:setup：在抓取使用的 Python 里安装 ruyiPage。
func (e *Env) BalalaSetup(ctx context.Context, cfg BalalaConfig) (python, output string, err error) {
	python = ResolvePython(cfg.PythonBin)
	c, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	output, err = pipInstallRuyipage(c, python)
	return python, output, err
}

// bootstrapCandidates 对应 Plugin::bootstrapEnv 的尝试顺序：Linux 先 python3 再 python，Windows 只用 python。
// 管理员在配置里指定了自定义命令时优先用它。
func bootstrapCandidates(configured string) []string {
	var out []string
	if c := strings.TrimSpace(configured); c != "" && c != "python" {
		out = append(out, c)
	}
	if runtime.GOOS == "windows" {
		return appendUnique(out, "python")
	}
	return appendUnique(appendUnique(out, "python3"), "python")
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// bootstrapEnv 依次尝试候选 Python，第一个安装成功的即返回；全部过程写入 log。
func bootstrapEnv(ctx context.Context, candidates []string, log io.Writer) (string, error) {
	var last error
	for _, py := range candidates {
		fmt.Fprintf(log, "[%s] 使用 %s 安装 ruyiPage\n", time.Now().Format("2006-01-02 15:04:05"), py)
		out, err := pipInstallRuyipage(ctx, py)
		fmt.Fprintln(log, strings.TrimSpace(out))
		if err == nil {
			fmt.Fprintf(log, "[%s] 安装完成 (%s)\n", time.Now().Format("2006-01-02 15:04:05"), py)
			return py, nil
		}
		fmt.Fprintf(log, "[%s] %s 失败: %v\n", time.Now().Format("2006-01-02 15:04:05"), py, err)
		last = err
	}
	if last == nil {
		last = fmt.Errorf("没有可用的 Python 候选")
	}
	return "", last
}

// BalalaSetupLogPath 返回环境安装日志路径，与 PHP 的 storage/logs/balala_sub_setup.log 一致。
func BalalaSetupLogPath() string {
	dir := os.Getenv("XBOARD_STORAGE_DIR")
	if dir == "" {
		dir = "storage"
	}
	return filepath.Join(dir, "logs", "balala_sub_setup.log")
}

// BalalaBootstrapEnv 在后台安装抓取环境：插件「安装」时调用，不阻塞请求，失败只写日志，不影响插件安装
// （对应 Plugin::install → bootstrapEnv）。返回的 channel 在结束时关闭，便于测试等待。
func (e *Env) BalalaBootstrapEnv() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		logPath := BalalaSetupLogPath()
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
			return
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return
		}
		defer f.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		_, _ = bootstrapEnv(ctx, bootstrapCandidates(e.LoadBalalaConfig().PythonBin), f)
	}()
	return done
}
