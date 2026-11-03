package jobs

import (
	"reflect"
	"testing"
)

func TestCommissionShares(t *testing.T) {
	if got := CommissionShares(false, 10, 5, 2); !reflect.DeepEqual(got, []int{100}) {
		t.Fatalf("no distribution: %v", got)
	}
	if got := CommissionShares(true, 10, 5, 2); !reflect.DeepEqual(got, []int{10, 5, 2}) {
		t.Fatalf("distribution: %v", got)
	}
}

func TestReferralExpiry(t *testing.T) {
	past, future := int64(100), int64(5000)
	if got := ReferralExpiry(nil, 1000, 60); got != 1060 {
		t.Fatalf("nil: %d", got)
	}
	if got := ReferralExpiry(&past, 1000, 60); got != 1060 {
		t.Fatalf("expired restarts from now: %d", got)
	}
	if got := ReferralExpiry(&future, 1000, 60); got != 5060 {
		t.Fatalf("future extends: %d", got)
	}
}

func TestRoundCents(t *testing.T) {
	cases := map[float64]int64{0.4: 0, 0.5: 1, 2.5: 3, 9.99: 10}
	for in, want := range cases {
		if got := roundCents(in); got != want {
			t.Errorf("%v: want %d got %d", in, want, got)
		}
	}
}
