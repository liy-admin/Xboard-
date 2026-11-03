package api

import (
	"encoding/json"
	"testing"

	"xboard/internal/model"
)

func TestPlanRawMatchesPHPModelFields(t *testing.T) {
	content := "原始 {{transfer}} 描述"
	cap := int64(0)
	p := &model.Plan{ID: 1, Name: "青铜", Show: true, Sell: false, Renew: true, Content: &content,
		Prices: json.RawMessage(`{"monthly":9.9,"onetime":null}`), Tags: json.RawMessage(`["热门"]`),
		CapacityLimit: &cap, CreatedAt: 10, UpdatedAt: 20}
	b, _ := json.Marshal(planRaw(p))
	var got map[string]any
	_ = json.Unmarshal(b, &got)

	want := []string{"id", "group_id", "transfer_enable", "name", "speed_limit", "show", "sort", "renew", "content",
		"prices", "reset_traffic_method", "capacity_limit", "created_at", "updated_at", "sell", "device_limit", "tags"}
	if len(got) != len(want) {
		t.Errorf("field count %d, want %d: %v", len(got), len(want), got)
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing PHP field %q", k)
		}
	}
	// PHP 原始模型不带旧版价格字段
	for _, k := range []string{"month_price", "week_price", "year_price", "reset_price"} {
		if _, ok := got[k]; ok {
			t.Errorf("legacy field %q must not appear in the raw model", k)
		}
	}
	if got["content"] != content {
		t.Errorf("content must stay raw (no template substitution): %v", got["content"])
	}
	if got["capacity_limit"] != float64(0) {
		t.Errorf("capacity_limit must stay raw, got %v", got["capacity_limit"])
	}
	if got["sell"] != float64(0) || got["show"] != true || got["renew"] != true {
		t.Errorf("sell is 0/1 (no cast), show/renew are bool: %v %v %v", got["sell"], got["show"], got["renew"])
	}
	if pr, _ := got["prices"].(map[string]any); pr["monthly"] != 9.9 {
		t.Errorf("prices object: %v", got["prices"])
	}
}
