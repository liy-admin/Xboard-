package jobs

import (
	"strconv"
)

// fmtSscan 解析十进制整数字符串。
func fmtSscan(s string, out *int64) (int, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	*out = v
	return 1, nil
}

// jsonInt 返回整数的 JSON 文本形式，供 JSON_CONTAINS 使用。
func jsonInt(v int64) string { return strconv.FormatInt(v, 10) }
