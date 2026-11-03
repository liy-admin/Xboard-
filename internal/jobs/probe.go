package jobs

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"xboard/internal/model"
	"xboard/internal/singbox"
	"xboard/internal/sub"
)

// 节点探测，对应 app/Services/NodeProbeService.php：
//   - node_probe_mode=singbox（缺省）：起一个只含被测节点的 sing-box，经本地 socks 发真实请求测延迟，
//     不回退 TCP（TCP 通不代表节点可用）；
//   - node_probe_mode=tcp：TCP 握手取 3 次最低，UDP 类协议用 ICMP ping。
// sing-box 的定位与在线下载由 internal/singbox 负责（自定义路径 → PATH → 托管副本）。

// ProbeResult 与 PHP 返回的数组一致。
type ProbeResult struct {
	OK            bool    `json:"ok"`
	LatencyMs     *int64  `json:"latency_ms"`
	MinLatency    *int64  `json:"min_latency"`
	AvgLatency    *int64  `json:"avg_latency"`
	MedianLatency *int64  `json:"median_latency"`
	MaxLatency    *int64  `json:"max_latency"`
	Tests         int     `json:"tests"`
	SuccessTests  int     `json:"success_tests"`
	ExitIP        *string `json:"exit_ip"`
	CountryCode   *string `json:"country_code"`
	Error         *string `json:"error"`
	// Untestable 为 true 表示「没测成」（基础设施问题），不是节点不通，调用方应保留上次结果。
	Untestable bool `json:"untestable"`
}

// ProbeOptions 覆盖设置项（诊断用快速探测）。
type ProbeOptions struct {
	TestCount        int
	RequestTimeoutMs int
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func i64Ptr(v int64) *int64 { return &v }

func failResult(err string, untestable bool, tests int) ProbeResult {
	return ProbeResult{Error: &err, Untestable: untestable, Tests: tests}
}

// ProbeNode 探测一个父节点；绝不 panic，异常按「没法测」处理。
func (e *Env) ProbeNode(s model.Server, opt ProbeOptions) (res ProbeResult) {
	defer func() {
		if r := recover(); r != nil {
			res = failResult(fmt.Sprintf("探测异常：%v", r), true, 0)
			res.CountryCode = strPtr(ParseCountryFromName(s.Name))
		}
	}()
	if e.Set.Get("node_probe_mode", "singbox") == "singbox" {
		res = e.probeViaSingbox(s, opt)
		if !res.OK && res.CountryCode == nil {
			res.CountryCode = strPtr(ParseCountryFromName(s.Name))
		}
		return res
	}
	return e.probeViaTCP(s)
}

// ---- TCP / ping ----

var udpTypes = map[string]bool{"hysteria": true, "tuic": true}

func firstPort(raw string) int {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "-"); i >= 0 {
		raw = raw[:i]
	}
	n, _ := strconv.Atoi(strings.TrimSpace(raw))
	return n
}

func (e *Env) timeoutMs() int {
	t := e.Set.Int("server_latency_timeout_ms", 2000)
	if t < 200 {
		t = 200
	}
	if t > 10000 {
		t = 10000
	}
	return t
}

func tcpConnectBest(host string, port int, timeout time.Duration) (int64, bool) {
	if port <= 0 {
		return 0, false
	}
	best := int64(-1)
	for i := 0; i < 3; i++ {
		if ms, ok := tcpLatency(host, port, timeout); ok && (best < 0 || ms < best) {
			best = ms
		}
		time.Sleep(150 * time.Millisecond)
	}
	return best, best >= 0
}

var pingRe = regexp.MustCompile(`(?i)time[=<]\s*([\d.]+)\s*ms`)

func pingHost(host string, timeoutMs int) (int64, bool) {
	wait := (timeoutMs + 999) / 1000
	if wait < 1 {
		wait = 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait+2)*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(wait), host).Output()
	if err != nil {
		return 0, false
	}
	if m := pingRe.FindSubmatch(out); m != nil {
		f, _ := strconv.ParseFloat(string(m[1]), 64)
		return int64(f + 0.5), true
	}
	return 0, false
}

func (e *Env) probeViaTCP(s model.Server) ProbeResult {
	timeoutMs := e.timeoutMs()
	var ms int64
	var ok bool
	if udpTypes[s.Type] {
		ms, ok = pingHost(s.Host, timeoutMs)
	} else {
		ms, ok = tcpConnectBest(s.Host, firstPort(s.Port), time.Duration(timeoutMs)*time.Millisecond)
	}
	// 国家：先从节点名解析，再 GeoIP(host) 兜底。
	cc := ParseCountryFromName(s.Name)
	if cc == "" {
		cc = LookupCountries(context.Background(), []string{s.Host})[strings.TrimSpace(s.Host)]
	}
	r := ProbeResult{OK: ok, Tests: 1, CountryCode: strPtr(cc)}
	if ok {
		r.LatencyMs, r.MinLatency, r.AvgLatency = i64Ptr(ms), i64Ptr(ms), i64Ptr(ms)
		r.MedianLatency, r.MaxLatency = i64Ptr(ms), i64Ptr(ms)
		r.SuccessTests = 1
	} else {
		r.Error = strPtr("tcp/ping 探测失败")
	}
	return r
}

// ---- sing-box ----

// ResolveSingBox 按 PHP SingBoxBinaryService::resolve 的前两步定位可用的 sing-box。
func ResolveSingBox(custom string) (bin, version, errMsg string) {
	custom = strings.TrimSpace(custom)
	cands := []string{}
	if custom != "" && custom != "sing-box" {
		cands = append(cands, custom)
	}
	cands = append(cands, "sing-box")
	last := ""
	for _, c := range cands {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		out, err := exec.CommandContext(ctx, c, "version").CombinedOutput()
		cancel()
		text := string(out)
		if i := strings.Index(text, "sing-box version"); i >= 0 {
			line := text[i+len("sing-box version"):]
			if j := strings.IndexByte(line, '\n'); j >= 0 {
				line = line[:j]
			}
			return c, strings.TrimSpace(line), ""
		}
		if err != nil {
			last = err.Error()
		} else {
			last = "输出不包含 sing-box version"
		}
	}
	return "", "", last
}

func freeLocalPort() (int, bool) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, false
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, true
}

func tailOf(path string, n int) string {
	b, _ := os.ReadFile(path)
	r := []rune(strings.TrimSpace(string(b)))
	if len(r) > n {
		r = r[len(r)-n:]
	}
	return string(r)
}

// findProbeUser 取一个有权限访问该节点的真实用户（未封禁、未过期），与 show / 流量无关。
func (e *Env) findProbeUser(s model.Server) (*model.User, bool) {
	u, ok, _ := e.findProbeUserWhy(s)
	return u, ok
}

// placeholderProbeUser 供带上游凭据（code）的导入节点使用：这类节点的认证信息取自 code，与面板用户无关。
func placeholderProbeUser() *model.User {
	return &model.User{UUID: "00000000-0000-0000-0000-000000000000"}
}

// findProbeUserWhy 为节点挑一个用于生成出站配置的用户，失败时给出具体原因。
//
// 与 PHP 的差异：PHP 只在节点权限组里找用户，导入的节点通常没分配权限组，结果整批「无法测试」。
// 但导入节点的认证信息来自 code（上游账号 UUID / 密码），不依赖面板用户，所以此时用占位用户即可。
func (e *Env) findProbeUserWhy(s model.Server) (*model.User, bool, string) {
	hasCode := s.Code != nil && strings.TrimSpace(*s.Code) != ""
	var raw []any
	_ = json.Unmarshal(s.GroupIDs, &raw)
	var gids []int64
	for _, v := range raw {
		if n, err := strconv.ParseInt(fmt.Sprint(v), 10, 64); err == nil {
			gids = append(gids, n)
		}
	}
	if len(gids) == 0 {
		if hasCode {
			return placeholderProbeUser(), true, ""
		}
		return nil, false, "节点没有分配权限组，无法取得测试用的用户凭据（请在节点编辑里选择权限组）"
	}
	var u model.User
	err := e.DB.Where("group_id IN ? AND banned = 0 AND (expired_at IS NULL OR expired_at >= ?)", gids, e.nowTs()).
		Order("id ASC").First(&u).Error
	if err == nil {
		return &u, true, ""
	}
	if hasCode {
		return placeholderProbeUser(), true, ""
	}
	return nil, false, "节点所在权限组里没有可用用户（未封禁且未过期），无法取得测试用的用户凭据"
}

func latencyStats(l []int64) (mn, avg, median, mx int64) {
	sort.Slice(l, func(i, j int) bool { return l[i] < l[j] })
	n := len(l)
	var sum int64
	for _, v := range l {
		sum += v
	}
	if n%2 == 1 {
		median = l[n/2]
	} else {
		median = int64(float64(l[n/2-1]+l[n/2])/2 + 0.5)
	}
	return l[0], int64(float64(sum)/float64(n) + 0.5), median, l[n-1]
}

func (e *Env) probeViaSingbox(s model.Server, opt ProbeOptions) ProbeResult {
	testCount := opt.TestCount
	if testCount <= 0 {
		testCount = e.Set.Int("node_probe_test_count", 3)
	}
	if testCount < 1 {
		testCount = 1
	}
	if testCount > 10 {
		testCount = 10
	}
	fail := func(msg string, untestable bool) ProbeResult { return failResult(msg, untestable, testCount) }

	bin, _, errMsg := e.resolveSingBox(true)
	if bin == "" {
		return fail("sing-box 不可用（"+errMsg+"）：请在后台「测速设置」点击「在线下载」，或手动安装 sing-box 并填写 node_probe_singbox_path", true)
	}
	user, ok, why := e.findProbeUserWhy(s)
	if !ok {
		return fail(why, true)
	}
	ob := sub.ProbeOutbound(s, s, user)
	if ob == nil {
		return fail("无法为该节点生成 sing-box outbound（无可用用户或协议不受支持）", true)
	}
	ob["tag"] = "proxy"

	// 预解析域名为 IP：否则 DNS 查询会在 route.final=proxy 下塞进尚未连上的节点而死锁。
	orig, _ := ob["server"].(string)
	if orig == "" {
		orig = s.Host
	}
	if net.ParseIP(orig) == nil {
		if ip, ok := hostToIPv4(orig); ok {
			ob["server"] = ip
			if t, ok := ob["tls"].(map[string]any); ok {
				if n, _ := t["server_name"].(string); n == "" {
					t["server_name"] = orig
				}
			}
		}
	}

	port, ok := freeLocalPort()
	if !ok {
		return fail("无法分配本地端口", true)
	}
	cfg := map[string]any{
		"log":       map[string]any{"level": "info"},
		"dns":       map[string]any{"servers": []any{map[string]any{"type": "udp", "tag": "r", "server": e.Set.Get("node_probe_dns", "1.1.1.1")}}},
		"inbounds":  []any{map[string]any{"type": "mixed", "tag": "in", "listen": "127.0.0.1", "listen_port": port}},
		"outbounds": []any{ob},
		"route":     map[string]any{"final": "proxy"},
	}
	cfgFile, err := os.CreateTemp("", "sbprobe_*.json")
	if err != nil {
		return fail("无法写入临时配置："+err.Error(), true)
	}
	cfgPath := cfgFile.Name()
	logPath := cfgPath + ".log"
	defer os.Remove(cfgPath)
	defer os.Remove(logPath)
	b, _ := json.Marshal(cfg)
	_, _ = cfgFile.Write(b)
	_ = cfgFile.Close()

	logFile, err := os.Create(logPath)
	if err != nil {
		return fail("无法创建日志文件："+err.Error(), true)
	}
	cmd := exec.Command(bin, "run", "-c", cfgPath)
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fail(fmt.Sprintf("无法启动 sing-box（检查 node_probe_singbox_path：%s）", bin), true)
	}
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
		_ = logFile.Close()
	}()
	defer func() {
		_ = cmd.Process.Kill()
		<-exited
	}()
	alive := func() bool {
		select {
		case <-exited:
			return false
		default:
			return true
		}
	}

	// 就绪判断：轮询日志里的 "server started"，不能用 TCP 连一下（可能假阳性）。
	ready := false
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		lg, _ := os.ReadFile(logPath)
		if strings.Contains(string(lg), "server started") || strings.Contains(string(lg), "sing-box started") {
			ready = true
			break
		}
		if !alive() {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !ready {
		lg := tailOf(logPath, 400)
		state := "启动超时"
		if !alive() {
			state = "启动失败/退出"
		}
		if lg == "" {
			lg = "（无日志，可能二进制无法运行）"
		}
		return fail("sing-box "+state+"："+lg, true)
	}

	reqTimeout := opt.RequestTimeoutMs
	if reqTimeout <= 0 {
		reqTimeout = e.Set.Int("node_probe_request_timeout_ms", 10000)
	}
	if reqTimeout < 2000 {
		reqTimeout = 2000
	}
	client := &http.Client{
		Timeout: time.Duration((reqTimeout+999)/1000) * time.Second,
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(&url.URL{Scheme: "socks5h", Host: "127.0.0.1:" + strconv.Itoa(port)}),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	latURL := e.Set.Get("node_probe_latency_url", "https://www.gstatic.com/generate_204")
	var lats []int64
	lastErr := ""
	for i := 0; i < testCount; i++ {
		t0 := time.Now()
		resp, err := client.Get(latURL)
		if err != nil {
			lastErr = err.Error()
			continue
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		if resp.StatusCode < 400 {
			lats = append(lats, time.Since(t0).Milliseconds())
		} else {
			lastErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
	}
	if len(lats) == 0 {
		lg := tailOf(logPath, 400)
		// 进程已退出或日志里有 FATAL（配置 / 启动失败）才算「没法测」。
		// 不能用「日志非空」判断：info 级别的启动行和连接失败的 ERROR 行都会让日志非空，
		// 会把真正不通的节点永远当成没法测（PHP 原实现的这一判断因此形同虚设）。
		if !alive() || strings.Contains(lg, "FATAL") {
			if lg == "" {
				lg = lastErr
			}
			return fail("真连测速未完成（sing-box 异常）："+lg+" | 请求错误："+lastErr, true)
		}
		if lastErr == "" {
			lastErr = "无成功请求"
		}
		return fail("真连测速失败："+lastErr, false)
	}
	mn, avg, med, mx := latencyStats(lats)

	// 出口 IP / 国家：best-effort。
	var exitIP, cc string
	echoURL := e.Set.Get("node_probe_ip_echo_url", "http://ip-api.com/json/?fields=status,countryCode,query")
	if resp, err := client.Get(echoURL); err != nil {
		log.Printf("probe: 出口 IP 回显请求失败（%s）: %v", s.Name, err)
	} else {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		if resp.StatusCode == 200 {
			exitIP, cc = parseEcho(body)
		} else {
			log.Printf("probe: 出口 IP 回显返回 HTTP %d（%s）", resp.StatusCode, s.Name)
		}
	}
	if cc == "" && exitIP != "" {
		cc = LookupCountries(context.Background(), []string{exitIP})[exitIP]
	}
	if cc == "" {
		cc = ParseCountryFromName(s.Name)
	}
	return ProbeResult{
		OK: true, LatencyMs: i64Ptr(mn), MinLatency: i64Ptr(mn), AvgLatency: i64Ptr(avg),
		MedianLatency: i64Ptr(med), MaxLatency: i64Ptr(mx), Tests: testCount, SuccessTests: len(lats),
		ExitIP: strPtr(exitIP), CountryCode: strPtr(strings.ToUpper(cc)),
	}
}

// parseEcho 解析 ip-api 风格 JSON 或纯文本 IP。
func parseEcho(body []byte) (ip, cc string) {
	var m map[string]any
	if json.Unmarshal(body, &m) == nil {
		for _, k := range []string{"query", "ip"} {
			if v, ok := m[k].(string); ok && v != "" {
				ip = v
				break
			}
		}
		for _, k := range []string{"countryCode", "country_code", "country"} {
			if v, ok := m[k].(string); ok && len(v) == 2 {
				cc = v
				break
			}
		}
		if ip != "" {
			return ip, cc
		}
	}
	t := strings.TrimSpace(string(body))
	if net.ParseIP(t) != nil {
		return t, ""
	}
	return "", ""
}

// resolveSingBox 按后台设置定位 sing-box；allowDownload 为 true 且开启了自动下载时，缺失会在线下载。
func (e *Env) resolveSingBox(allowDownload bool) (bin, ver, errMsg string) {
	auto := allowDownload && e.Set.Bool("node_probe_singbox_auto_download", true)
	version := strings.TrimSpace(e.Set.Get("node_probe_singbox_version", singbox.DefaultVersion))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return singbox.Default().Resolve(ctx, e.Set.Get("node_probe_singbox_path", ""), version, auto)
}

// ResolveSingBoxFor 供管理端诊断使用：只定位，不触发下载。
func (e *Env) ResolveSingBoxFor() (bin, ver, errMsg string) { return e.resolveSingBox(false) }

// FindProbeUser 为节点挑一个有权使用它的用户（与测速相同的规则），失败时给出原因。
// 管理端导出分享链接时用它作为缺省用户。
func (e *Env) FindProbeUser(s model.Server) (*model.User, bool, string) {
	return e.findProbeUserWhy(s)
}
