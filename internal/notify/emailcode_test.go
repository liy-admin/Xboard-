package notify

import (
	"context"
	"strconv"
	"testing"
	"time"
)

type memStore struct {
	m map[string]string
	n map[string]int64
}

func (s *memStore) Get(_ context.Context, k string) (string, bool, error) {
	v, ok := s.m[k]
	return v, ok, nil
}

func (s *memStore) Set(_ context.Context, k, v string, _ time.Duration) error {
	s.m[k] = v
	return nil
}

func (s *memStore) Incr(_ context.Context, k string, _ time.Duration) (int64, error) {
	s.n[k]++
	return s.n[k], nil
}

func (s *memStore) Del(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(s.m, k)
		delete(s.n, k)
	}
	return nil
}

func TestEmailCodeIssueVerifyAndLimit(t *testing.T) {
	st := &memStore{m: map[string]string{}, n: map[string]int64{}}
	e := &EmailCode{Store: st}
	ctx := context.Background()
	code, err := e.Issue(ctx, " User@Example.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if n, _ := strconv.Atoi(code); n < 100000 || n > 999999 {
		t.Fatalf("code not 6 digits: %s", code)
	}
	if ok, _ := e.Verify(ctx, "user@example.com", "000000"); ok {
		t.Fatal("wrong code accepted")
	}
	if ok, _ := e.Verify(ctx, "user@example.com", code); !ok {
		t.Fatal("right code rejected")
	}
	// 超过 5 次后验证码被删除，之后即使输入正确也失败。
	for i := 0; i < 6; i++ {
		e.Verify(ctx, "user@example.com", "bad")
	}
	if ok, _ := e.Verify(ctx, "user@example.com", code); ok {
		t.Fatal("should be locked out after too many attempts")
	}
}
