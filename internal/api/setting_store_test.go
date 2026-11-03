package api

import (
	"encoding/json"
	"testing"
)

func TestSettingStoredValueEncodesTypedCollectionsAsJSON(t *testing.T) {
	// 回归：保存订阅预设传入 []map[string]any，曾被写成 "[map[...]]" 而读不回来。
	list := []map[string]any{{"id": "a", "name": "机场 A", "group_ids": []int{1, 2}, "rate": 1.5}}
	v, err := settingStoredValue(list)
	if err != nil {
		t.Fatal(err)
	}
	var back []map[string]any
	if err := json.Unmarshal([]byte(v.(string)), &back); err != nil || len(back) != 1 || back[0]["name"] != "机场 A" {
		t.Fatalf("stored %q not valid JSON round-trip: %v", v, err)
	}
	for _, in := range []any{[]string{"a", "b"}, []int64{1, 2}, map[string]int{"x": 1}} {
		s, _ := settingStoredValue(in)
		if !json.Valid([]byte(s.(string))) {
			t.Errorf("%T must be stored as JSON, got %q", in, s)
		}
	}
}

func TestSettingStoredValueScalars(t *testing.T) {
	cases := []struct {
		in   any
		want any
	}{{nil, nil}, {true, "1"}, {false, ""}, {"x", "x"}, {42, "42"}, {1.5, "1.5"}, {json.RawMessage(`{"a":1}`), `{"a":1}`}}
	for _, c := range cases {
		got, err := settingStoredValue(c.in)
		if err != nil || got != c.want {
			t.Errorf("%v => %v (%v), want %v", c.in, got, err, c.want)
		}
	}
}

// 与线上损坏数据同形（两条预设，名称带空格的情况另测）。
const corruptedSubs = `[map[group_ids:[] id:61c6c076-37a1-4403-9e36-9b163560e61b interval_hours:24 last_run_at:<nil> name:1 rate:1 refetch_balala:false schedule_enabled:false show:1 tag:1 url:https://sub.example.com/s/TOKEN1] map[group_ids:[3 4] id:dce8287d-e6d6-477a-b113-3c51de1431e6 interval_hours:6 last_run_at:2026-10-16 03:00:00 name:机场 A 备用 rate:1.5 refetch_balala:true schedule_enabled:true show:0 tag:HK 线路 url:https://sub.example.com/s/TOKEN2]]`

func TestRecoverFmtSubscriptions(t *testing.T) {
	got, ok := recoverFmtSubscriptions(corruptedSubs)
	if !ok || len(got) != 2 {
		t.Fatalf("ok=%v len=%d", ok, len(got))
	}
	a, b := got[0], got[1]
	if a["name"] != "1" || a["url"] != "https://sub.example.com/s/TOKEN1" || a["tag"] != "1" ||
		a["interval_hours"] != 24 || a["schedule_enabled"] != false || a["last_run_at"] != nil || a["rate"] != 1.0 {
		t.Errorf("first entry: %+v", a)
	}
	if b["name"] != "机场 A 备用" || b["tag"] != "HK 线路" || b["last_run_at"] != "2026-10-16 03:00:00" ||
		b["rate"] != 1.5 || b["interval_hours"] != 6 || b["schedule_enabled"] != true || b["refetch_balala"] != true || b["show"] != 0 {
		t.Errorf("second entry (spaces in values): %+v", b)
	}
	if g, _ := b["group_ids"].([]int); len(g) != 2 || g[0] != 3 || g[1] != 4 {
		t.Errorf("group_ids: %v", b["group_ids"])
	}
	// 还原结果必须能被正常序列化并读回
	enc, _ := settingStoredValue(got)
	var back []map[string]any
	if err := json.Unmarshal([]byte(enc.(string)), &back); err != nil || len(back) != 2 {
		t.Fatalf("recovered list not JSON round-trippable: %v", err)
	}
}

func TestRecoverFmtSubscriptionsRejectsOtherFormats(t *testing.T) {
	for _, raw := range []string{"", "[]", `[{"id":"x"}]`, "[map[foo:bar]]", "garbage"} {
		if _, ok := recoverFmtSubscriptions(raw); ok {
			t.Errorf("%q must not be treated as recoverable", raw)
		}
	}
}
