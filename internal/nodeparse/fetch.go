package nodeparse

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// FetchSubscription 下载订阅内容并尝试 base64 解码。仅允许公网 http(s) 地址（防止 SSRF），与 PHP assertPublicUrl 一致。
func FetchSubscription(raw string) (string, error) {
	return fetchWithUA(raw, uaGeneric)
}

// 订阅服务端按 User-Agent 决定输出格式：通用客户端拿到 base64 分享链接列表，Clash 系客户端拿到 YAML。
// 两种格式包含的节点类型可能不同（例如 hysteria2 只出现在 clash.meta 的 YAML 里；老版 Clash UA 拿不到），所以导入时两种都取。
const (
	uaGeneric = "v2rayN/6.0"
	uaClash   = "clash.meta"
)

func fetchWithUA(raw, ua string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("订阅 URL 必须使用 http 或 https 协议")
	}
	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("订阅 URL 缺少主机名")
	}
	// 仅用于本机联调：XBOARD_ALLOW_PRIVATE_SUBSCRIPTION=1 时放开私有地址限制（生产环境不要设置）。
	allowPrivate := os.Getenv("XBOARD_ALLOW_PRIVATE_SUBSCRIPTION") == "1"
	if ips, err := net.LookupIP(host); err == nil && !allowPrivate {
		for _, ip := range ips {
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
				return "", fmt.Errorf("订阅 URL 不允许指向私有或保留地址")
			}
		}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, raw, nil)
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取订阅失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("获取订阅失败: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return "", err
	}
	return TryDecodeBase64(string(body)), nil
}

// parseBody 按内容格式选择解析方式：Clash YAML 或分享链接文本。
func parseBody(body string) Result {
	if LooksLikeClashYAML(body) {
		return ParseClashYAML(body)
	}
	return ParseText(body)
}

func serverKey(s Server) string {
	return fmt.Sprintf("%s|%s|%v|%s", s.Type, s.Host, s.Port, s.Name)
}

// ParseSubscription 下载订阅并解析。会分别以通用客户端和 Clash 客户端的身份各取一次并合并去重，
// 任意一种格式失败不影响另一种；两种都失败才返回错误。
func ParseSubscription(raw string) (Result, error) {
	var merged Result
	seen := map[string]bool{}
	var firstErr error
	got := false
	for _, ua := range []string{uaGeneric, uaClash} {
		body, err := fetchWithUA(raw, ua)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		got = true
		r := parseBody(TryDecodeBase64(body))
		for _, s := range r.Servers {
			if k := serverKey(s); !seen[k] {
				seen[k] = true
				merged.Servers = append(merged.Servers, s)
			}
		}
		// 另一种格式能解析出节点时，这一种里"无法识别"的行不再报错，避免重复噪声。
		merged.Errors = append(merged.Errors, r.Errors...)
	}
	if !got {
		return Result{}, firstErr
	}
	if len(merged.Servers) > 0 {
		merged.Errors = dedupeErrors(merged.Errors)
	}
	return merged, nil
}

func dedupeErrors(errs []ParseError) []ParseError {
	seen := map[string]bool{}
	out := errs[:0:0]
	for _, e := range errs {
		k := fmt.Sprintf("%d|%s|%s", e.Line, e.Content, e.Error)
		if !seen[k] {
			seen[k] = true
			out = append(out, e)
		}
	}
	return out
}
