package api

import (
	"encoding/json"
	"testing"
)

func TestAnyIntAcceptsJSONNumberStringAndFloat(t *testing.T) {
	var m map[string]any
	dec := json.NewDecoder(stringReader(`{"a": 12, "b": "34", "c": 5.9, "d": "x"}`))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]int64{"a": 12, "b": 34, "c": 5} {
		if got, ok := anyInt(m[k]); !ok || got != want {
			t.Errorf("%s: want %d got %d ok=%v", k, want, got, ok)
		}
	}
	if _, ok := anyInt(m["d"]); ok {
		t.Error("non-numeric string should not parse")
	}
}

func TestPick2DefaultsToZero(t *testing.T) {
	got := pick2(map[string]any{"mem": map[string]any{"total": 10}}, "mem")
	if got["total"].(int64) != 10 || got["used"].(int64) != 0 {
		t.Fatalf("unexpected %v", got)
	}
	empty := pick2(map[string]any{}, "missing")
	if empty["total"].(int64) != 0 {
		t.Fatalf("missing key should be zero: %v", empty)
	}
}
