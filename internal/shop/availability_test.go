package shop

import (
	"testing"
	"time"

	"xboard/internal/model"
)

func i64p(n int64) *int64 { return &n }

func TestIsAvailableTreatsZeroTrafficAsUnlimited(t *testing.T) {
	future := time.Now().Add(24 * time.Hour).Unix()
	past := time.Now().Add(-24 * time.Hour).Unix()
	cases := []struct {
		name string
		u    model.User
		want bool
	}{
		{"流量为 0（无限）、永久有效", model.User{TransferEnab: 0}, true},
		{"流量为 0、未过期", model.User{TransferEnab: 0, ExpiredAt: i64p(future)}, true},
		{"有流量上限、未过期", model.User{TransferEnab: 10 << 30, ExpiredAt: i64p(future)}, true},
		{"已过期", model.User{TransferEnab: 0, ExpiredAt: i64p(past)}, false},
		{"已封禁，即使无限流量", model.User{TransferEnab: 0, Banned: true}, false},
		{"已封禁且有流量", model.User{TransferEnab: 10 << 30, Banned: true}, false},
	}
	for _, c := range cases {
		if got := IsAvailable(&c.u); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestUnlimitedTraffic(t *testing.T) {
	if !UnlimitedTraffic(&model.User{TransferEnab: 0}) || UnlimitedTraffic(&model.User{TransferEnab: 1}) {
		t.Fatal("0 means unlimited, any positive limit does not")
	}
}

// 流量为 0 不等于有会员：没有套餐的用户订阅可以下载（返回空节点），但不应被当作会员。
func TestHasPlanRequiresAPlan(t *testing.T) {
	if HasPlan(&model.User{TransferEnab: 0}) {
		t.Error("a user without a plan must not count as a member just because traffic is 0")
	}
	if !HasPlan(&model.User{PlanID: i64p(3), TransferEnab: 0}) {
		t.Error("a user with a plan and unlimited traffic is a member")
	}
	if HasPlan(&model.User{PlanID: i64p(3), Banned: true}) {
		t.Error("banned users are not members")
	}
	past := time.Now().Add(-time.Hour).Unix()
	if HasPlan(&model.User{PlanID: i64p(3), ExpiredAt: i64p(past)}) {
		t.Error("expired plan is not a member")
	}
}
