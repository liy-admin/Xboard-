package api

import (
	"reflect"
	"testing"
)

func TestNormalizeIDList(t *testing.T) {
	got := normalizeIDList("group_ids", []any{"39", float64(2), "x"})
	want := []any{int64(39), float64(2), "x"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	if v := normalizeIDList("tags", []any{"1"}); !reflect.DeepEqual(v, []any{"1"}) {
		t.Fatalf("non-id keys must be untouched: %#v", v)
	}
}
