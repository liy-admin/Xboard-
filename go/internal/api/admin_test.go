package api

import (
	"encoding/json"
	"testing"
)

func TestAdminBodyValueCoercion(t *testing.T) {
	m := map[string]any{
		"n":    json.Number("42"),
		"s":    "17",
		"f":    float64(3),
		"bool": true,
		"str":  "x",
		"b2":   "0",
	}
	if v, ok := mInt(m, "n"); !ok || v != 42 {
		t.Fatalf("json.Number: %d %v", v, ok)
	}
	if v, ok := mInt(m, "s"); !ok || v != 17 {
		t.Fatalf("string int: %d %v", v, ok)
	}
	if v, ok := mInt(m, "f"); !ok || v != 3 {
		t.Fatalf("float: %d %v", v, ok)
	}
	if _, ok := mInt(m, "str"); ok {
		t.Fatal("non numeric must not coerce")
	}
	if v, _ := mBool(m, "bool"); !v {
		t.Fatal("bool true")
	}
	if v, _ := mBool(m, "b2"); v {
		t.Fatal(`"0" must be false`)
	}
	if s, ok := mStr(m, "str"); !ok || s != "x" {
		t.Fatalf("string: %q", s)
	}
}

func TestCouponViewRendersLegacyPeriodsAndStringIDs(t *testing.T) {
	raw := json.RawMessage(`["monthly","yearly"]`)
	got := periodLegacyNames(raw)
	if len(got) != 2 || got[0] != "month_price" || got[1] != "year_price" {
		t.Fatalf("legacy periods: %v", got)
	}
	ids := idStrings(json.RawMessage(`[1,22]`))
	if len(ids) != 2 || ids[0] != "1" || ids[1] != "22" {
		t.Fatalf("ids: %v", ids)
	}
	if idStrings(nil) != nil {
		t.Fatal("empty list must be null")
	}
}
