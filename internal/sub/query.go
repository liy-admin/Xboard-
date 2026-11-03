package sub

import (
	"net/url"
	"strconv"
	"strings"
)

// orderedQuery 复刻 PHP 关联数组 + http_build_query 的行为：保持插入顺序、赋值覆盖原位、null 跳过。
type orderedQuery struct {
	keys []string
	vals map[string]any
}

func newQuery() *orderedQuery { return &orderedQuery{vals: map[string]any{}} }

// set 写入一个键；值为 nil 时整个键不输出（与 PHP 的 http_build_query 一致，但若键已存在则保留原位）。
func (q *orderedQuery) set(k string, v any) *orderedQuery {
	if _, ok := q.vals[k]; !ok {
		q.keys = append(q.keys, k)
	}
	q.vals[k] = v
	return q
}

func (q *orderedQuery) has(k string) bool {
	_, ok := q.vals[k]
	return ok
}

// encode 输出查询串。rfc3986 为 true 时空格编码为 %20，否则为 +（PHP 默认 RFC1738）。
func (q *orderedQuery) encode(rfc3986 bool) string {
	parts := make([]string, 0, len(q.keys))
	for _, k := range q.keys {
		v := q.vals[k]
		if v == nil {
			continue
		}
		s := queryScalar(v)
		enc := url.QueryEscape
		if rfc3986 {
			enc = rawurlencode
		}
		parts = append(parts, enc(k)+"="+enc(s))
	}
	return strings.Join(parts, "&")
}

// queryScalar 把标量转为 PHP 字符串形式：true→"1"，false→"0"。
func queryScalar(v any) string {
	switch x := v.(type) {
	case bool:
		if x {
			return "1"
		}
		return "0"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return phpFloat(x)
	}
	return Str(v)
}

// intFlag 对应 (int) $v：true→1，其它按整数解析，缺失为 0。
func intFlag(v any) int {
	if b, ok := v.(bool); ok {
		if b {
			return 1
		}
		return 0
	}
	return Int(v)
}
