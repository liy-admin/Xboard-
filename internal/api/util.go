package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"math/big"
	"net/url"
	"strings"
)

// bgCtx 供不依赖请求上下文的 Redis 调用使用。
func bgCtx() context.Context { return context.Background() }

// randIndex 返回 [0,n) 的安全随机下标（对应 Arr::random）。
func randIndex(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}

// phpURLEncode 对应 PHP urlencode：空格为 +，其余与 RFC3986 相比保留 ~ 之外的编码差异。
func phpURLEncode(s string) string {
	return url.QueryEscape(s)
}

// stringReader 包装字符串为 io.Reader，仅供测试使用。
func stringReader(s string) io.Reader { return strings.NewReader(s) }

// subscribePath 对应 route('client.subscribe')：/{subscribe_path}/{token}。
func (a *App) subscribePath(token string) string {
	p := strings.Trim(a.Set.Get("subscribe_path", "s"), "/")
	return "/" + p + "/" + token
}

// subscribeURL 对应 Helper::getSubscribeUrl：domain 非空时使用 https://domain，否则从 subscribe_url 列表随机选取，
// 没有配置时回退到 app_url。
func (a *App) subscribeURL(token, domain string) string {
	return a.buildSubURL(a.subscribePath(token), domain)
}

// encSubscribeURL 对应 Helper::getEncryptedSubscribeUrl（路径 /{subscribe_path}-enc/{token}）。
func (a *App) encSubscribeURL(token string) string {
	p := strings.Trim(a.Set.Get("subscribe_path", "s"), "/")
	return a.buildSubURL("/"+p+"-enc/"+token, "")
}

// buildSubURL 按 subscribe_url 配置（逗号分隔，随机选取）或 app_url 拼接路径；domain 非空时固定使用 https://domain。
func (a *App) buildSubURL(path, domain string) string {
	if domain != "" {
		return "https://" + strings.TrimRight(domain, "/") + path
	}
	var picks []string
	for _, s := range strings.Split(a.Set.Get("subscribe_url", ""), ",") {
		if s = strings.TrimSpace(s); s != "" {
			picks = append(picks, s)
		}
	}
	base := a.Set.Get("app_url", "")
	if len(picks) > 0 {
		base = picks[randIndex(len(picks))]
	}
	if base == "" {
		// 对应 PHP 的 url($path)：没有配置站点地址时，用最近一次请求的 scheme://host。
		if v, ok := a.lastBase.Load().(string); ok {
			base = v
		}
	}
	return strings.TrimRight(base, "/") + path
}

// newEncKey 生成账号级加密密钥（32 字节随机数的 base64），对应 base64_encode(random_bytes(32))。
func newEncKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// orderedGroups 保持分组首次出现的顺序序列化为 JSON 对象（对应 PHP Collection::groupBy 的键顺序）。
type orderedGroups struct {
	keys []string
	vals map[string][]gin.H
}

func (g *orderedGroups) add(k string, v gin.H) {
	if _, ok := g.vals[k]; !ok {
		g.keys = append(g.keys, k)
	}
	g.vals[k] = append(g.vals[k], v)
}

func (g *orderedGroups) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range g.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		vb, err := json.Marshal(g.vals[k])
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// rememberBase 记录最近一次请求的 scheme://host，供没有请求上下文的地方（邮件、Telegram）拼完整链接。
func (a *App) rememberBase(c *gin.Context) {
	scheme := "http"
	if c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	if host := c.Request.Host; host != "" {
		a.lastBase.Store(scheme + "://" + host)
	}
	c.Next()
}
