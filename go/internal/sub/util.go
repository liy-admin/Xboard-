// Package sub 实现客户端订阅生成（对应 app/Protocols 与 ClientController::doSubscribe）。
package sub

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Map 即 PHP 里的关联数组；协议配置用它以保持与 data_get 同样的宽松取值语义。
type Map = map[string]any

// Get 等价 data_get($m, "a.b.c")；缺失返回 nil。
func Get(m any, path string) any {
	cur := m
	for _, seg := range strings.Split(path, ".") {
		switch t := cur.(type) {
		case Map:
			v, ok := t[seg]
			if !ok {
				return nil
			}
			cur = v
		case []any:
			i, err := strconv.Atoi(seg)
			if err != nil || i < 0 || i >= len(t) {
				return nil
			}
			cur = t[i]
		default:
			return nil
		}
	}
	return cur
}

func Str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		if t {
			return "1"
		}
		return ""
	}
	return fmt.Sprint(v)
}

func Int(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	case bool:
		if t {
			return 1
		}
	}
	return 0
}

// Truthy 复刻 PHP 的真值判断（0、""、"0"、null、空数组均为假）。
func Truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	case string:
		return t != "" && t != "0"
	case json.Number:
		// 管理端请求体用 UseNumber 解析；0 / 0.0 为假。
		f, err := t.Float64()
		return err != nil || f != 0
	case []any:
		return len(t) > 0
	case Map:
		return len(t) > 0
	}
	return true
}

func md5hex(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }

func wrapIPv6(a string) string {
	if ip := net.ParseIP(a); ip != nil && strings.Contains(a, ":") {
		return "[" + a + "]"
	}
	return a
}

// rawurlencode 对应 PHP rawurlencode（空格为 %20）。
func rawurlencode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// httpBuildQuery 对应 http_build_query：按插入顺序编码，null 跳过，bool 为 1/0。
type kv struct{ k, v string }
type Query []kv

func (q *Query) Add(k string, v any) {
	if v == nil {
		return
	}
	s := ""
	switch t := v.(type) {
	case bool:
		if t {
			s = "1"
		} else {
			s = "0"
		}
	default:
		s = Str(v)
	}
	*q = append(*q, kv{k, s})
}

func (q Query) Encode() string {
	parts := make([]string, len(q))
	for i, e := range q {
		parts[i] = url.QueryEscape(e.k) + "=" + url.QueryEscape(e.v)
	}
	return strings.Join(parts, "&")
}

// versionCompare 简化版 PHP version_compare：按 . 分段比较数字，返回 -1/0/1。
func versionCompare(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			y, _ = strconv.Atoi(pb[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func randPick(v any, def []any) any {
	arr, ok := v.([]any)
	if !ok || len(arr) == 0 {
		arr = def
	}
	if len(arr) == 0 {
		return nil
	}
	return arr[rand.Intn(len(arr))]
}

func tlsFingerprint(utls any) string {
	if m, ok := utls.(Map); ok {
		if !Truthy(m["enabled"]) {
			return ""
		}
		fp := Str(m["fingerprint"])
		if fp == "" {
			fp = "chrome"
		}
		if fp != "random" {
			return fp
		}
	}
	fps := []string{"chrome", "firefox", "safari", "ios", "edge", "qq"}
	return fps[rand.Intn(len(fps))]
}

func sortedKeys(m Map) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func trimToNull(v any) any {
	s := strings.TrimSpace(Str(v))
	if s == "" {
		return nil
	}
	return s
}

// compact 去掉值为 nil 的键，等价 array_filter(..., fn($v) => !is_null($v))。
func compact(m Map) Map {
	for k, v := range m {
		if v == nil {
			delete(m, k)
		}
	}
	return m
}
