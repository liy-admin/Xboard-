package notify

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// CodeStore 抽象验证码存储（生产环境为 Redis），便于测试替换。
type CodeStore interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	// Incr 原子自增，首次创建时设置过期时间，返回自增后的值。
	Incr(ctx context.Context, key string, ttl time.Duration) (int64, error)
	Del(ctx context.Context, keys ...string) error
}

const (
	emailCodeTTL     = 300 * time.Second
	emailMaxAttempts = 5
)

// EmailCode 对应 EmailCodeService：6 位数字验证码，5 分钟有效，最多校验 5 次。
type EmailCode struct {
	Store CodeStore
}

// Normalize 对应 EmailCodeService::normalize。
func Normalize(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func codeKey(email string) string { return "EMAIL_VERIFY_CODE_" + Normalize(email) }
func failKey(email string) string { return "EMAIL_VERIFY_ATTEMPTS_" + Normalize(email) }

// Issue 生成并保存验证码，返回明文供邮件发送。
func (e *EmailCode) Issue(ctx context.Context, email string) (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := 100000 + binary.BigEndian.Uint32(b[:])%900000
	code := fmt.Sprint(n)
	if err := e.Store.Set(ctx, codeKey(email), code, emailCodeTTL); err != nil {
		return "", err
	}
	return code, nil
}

// Verify 校验验证码。尝试次数超过上限后删除验证码，之后即使输入正确也失败。
func (e *EmailCode) Verify(ctx context.Context, email, code string) (bool, error) {
	cached, ok, err := e.Store.Get(ctx, codeKey(email))
	if err != nil || !ok {
		return false, err
	}
	attempts, err := e.Store.Incr(ctx, failKey(email), emailCodeTTL)
	if err != nil {
		return false, err
	}
	if attempts > emailMaxAttempts {
		_ = e.Store.Del(ctx, codeKey(email))
		return false, nil
	}
	return subtle.ConstantTimeCompare([]byte(cached), []byte(code)) == 1, nil
}

// Forget 删除验证码与尝试计数。
func (e *EmailCode) Forget(ctx context.Context, email string) error {
	return e.Store.Del(ctx, codeKey(email), failKey(email))
}
