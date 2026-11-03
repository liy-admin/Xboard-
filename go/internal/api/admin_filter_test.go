package api

import (
	"reflect"
	"testing"
)

func TestFilterCondition(t *testing.T) {
	cases := []struct {
		name  string
		col   string
		value any
		sql   string
		args  []any
	}{
		{"like default", "email", "foo", "email LIKE ?", []any{"%foo%"}},
		{"number is like", "id", float64(5), "id LIKE ?", []any{"%5%"}},
		{"eq numeric", "plan_id", "eq:3", "plan_id = ?", []any{int64(3)}},
		{"gt float", "balance", "gt:1.5", "balance > ?", []any{1.5}},
		{"lte", "expired_at", "lte:100", "expired_at <= ?", []any{int64(100)}},
		{"like op", "email", "like:abc", "email LIKE ?", []any{"%abc%"}},
		{"unknown op falls back to like", "email", "zzz:abc", "email LIKE ?", []any{"%abc%"}},
		{"null", "plan_id", "null:", "plan_id IS NULL", nil},
		{"notnull", "plan_id", "notnull:", "plan_id IS NOT NULL", nil},
		{"string eq", "uuid", "eq:abc-def", "uuid = ?", []any{"abc-def"}},
	}
	for _, c := range cases {
		sql, args := filterCondition(c.col, c.value)
		if sql != c.sql || !reflect.DeepEqual(args, c.args) {
			t.Errorf("%s: got (%q, %#v) want (%q, %#v)", c.name, sql, args, c.sql, c.args)
		}
	}
	// 数组 → IN
	sql, args := filterCondition("id", []any{float64(1), float64(2)})
	if sql != "id IN ?" || len(args) != 1 {
		t.Errorf("array: %q %#v", sql, args)
	}
}

func TestDecodeListAcceptsArrayAndJSONString(t *testing.T) {
	var a, b []filterItem
	decodeList([]any{map[string]any{"id": "email", "value": "x"}}, &a)
	decodeList(`[{"id":"email","value":"x","logic":"or"}]`, &b)
	if len(a) != 1 || a[0].ID != "email" {
		t.Fatalf("array form: %#v", a)
	}
	if len(b) != 1 || b[0].Logic != "or" {
		t.Fatalf("string form: %#v", b)
	}
}

func TestSafeIdent(t *testing.T) {
	for in, want := range map[string]bool{"email": true, "total_used": true, "a;drop": false, "": false, "x y": false} {
		if safeIdent(in) != want {
			t.Errorf("safeIdent(%q) != %v", in, want)
		}
	}
}
