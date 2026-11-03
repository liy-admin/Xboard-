package shop

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 网关共用的签名与参数工具，行为对齐 PHP 版插件（ksort + 拼接 + md5/hmac）。

// cfgStr 读取网关配置项，值统一转为字符串（后台保存的是 JSON，数字也可能是字符串）。
func cfgStr(cfg map[string]any, key string) string {
	v, ok := cfg[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "1"
		}
		return "0"
	}
	return fmt.Sprint(v)
}

// yuan 把分换算为元并格式化，对应 PHP 中 total_amount / 100 的浮点输出。
func yuan(cents int64) string {
	return strconv.FormatFloat(float64(cents)/100, 'f', -1, 64)
}

// yuan2 对应 sprintf('%.2f', total/100)。
func yuan2(cents int64) string {
	return strconv.FormatFloat(float64(cents)/100, 'f', 2, 64)
}

// rawSortedQuery 对应 PHP 的 ksort + stripslashes(urldecode(http_build_query(...)))：
// 按键排序后拼接为未编码的 k=v&k=v。
func rawSortedQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	return strings.Join(parts, "&")
}

// encodedSortedQuery 对应 PHP 的 ksort + http_build_query（值做 URL 编码）。
func encodedSortedQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	v := url.Values{}
	for _, k := range keys {
		v.Set(k, params[k])
	}
	return v.Encode()
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func hmacHex(h func() hash.Hash, msg, key string) string {
	m := hmac.New(h, []byte(key))
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func hmacSHA256Hex(msg, key string) string { return hmacHex(sha256.New, msg, key) }

func hmacSHA512Hex(msg, key string) string { return hmacHex(sha512.New, msg, key) }

// secureEqual 常量时间比较，对应 hash_equals。
func secureEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// httpClient 为网关请求统一的超时设置（PHP 版 curl 超时为 30~300 秒，这里取 30 秒）。
var httpClient = &http.Client{Timeout: 30 * time.Second}

// newestTimestamp 返回当前时间（用于签名参数）。
func newestTimestamp() string { return time.Now().Format("2006-01-02 15:04:05") }
