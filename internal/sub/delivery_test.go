package sub

import (
	"testing"
	"time"
)

func TestNodeAvailableProbeDisabledIgnoresAge(t *testing.T) {
	now := time.Now().Unix()
	old := now - 3600
	reachable := Map{"latency_ms": int64(80), "last_latency_at": old}
	failed := Map{"last_latency_at": old}
	never := Map{}
	// 测速开启：一小时前的结果已过期 → 不可用；从未测过 → 不可用
	if NodeAvailable(reachable, now, 600) || NodeAvailable(never, now, 600) {
		t.Fatal("probe enabled: stale or never-probed nodes must not count as available")
	}
	// 测速关闭：以最后一次结论为准；从未测过视为可用；测过不通仍不可用
	if !NodeAvailable(reachable, now, NoExpiry) || !NodeAvailable(never, now, NoExpiry) || NodeAvailable(failed, now, NoExpiry) {
		t.Fatal("probe disabled: keep the last verdict, treat never-probed as available")
	}
	if !NodeAvailable(Map{"is_online": 1}, now, 600) {
		t.Fatal("online agent is always available")
	}
}

func TestSelectForDeliveryFallsBackWhenNothingAvailable(t *testing.T) {
	now := time.Now().Unix()
	servers := []Map{{"id": int64(1), "country_code": "HK", "last_latency_at": now - 10}, {"id": int64(2), "country_code": "JP", "last_latency_at": now - 10}}
	if got := SelectForDelivery(servers, true, 600, false, nil); len(got) != 2 {
		t.Fatalf("all unavailable: must fall back to the full list, got %d", len(got))
	}
	if got := SelectForDelivery(servers, false, 600, false, nil); len(got) != 2 {
		t.Fatalf("no dedup: deliver as is, got %d", len(got))
	}
}
