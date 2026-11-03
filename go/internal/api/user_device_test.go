package api

import "testing"

func TestNormalizeDeviceIP(t *testing.T) {
	cases := map[string]string{
		"1.2.3.4:5678":      "1.2.3.4",
		"[2001:db8::1]:443": "2001:db8::1",
		"2001:db8::1":       "2001:db8::1",
	}
	for in, want := range cases {
		if got := normalizeDeviceIP(in); got != want {
			t.Errorf("%s: want %s got %s", in, want, got)
		}
	}
}

func TestParseUserDevicesFiltersExpired(t *testing.T) {
	now := int64(10000)
	data := map[string]string{
		"3:1.1.1.1": "9999",
		"3:2.2.2.2": "9000", // 超过 TTL
		"bad":       "9999",
		"4:3.3.3.3": "x",
	}
	got := parseUserDevices(data, now)
	if len(got) != 1 || got[0].NodeID != 3 || got[0].IP != "1.1.1.1" {
		t.Fatalf("unexpected devices: %+v", got)
	}
}
