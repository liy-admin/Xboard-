package api

import (
	"strings"
	"testing"
)

func TestDeviceKeywordFilterSQL(t *testing.T) {
	a, _ := dryRunApp(t)
	sqlOf := func(kw string) (string, []any) {
		var rows []deviceRow
		stmt := deviceKeywordFilter(a.deviceBase(), kw).Select(deviceRowSelect).Scan(&rows).Statement
		return stmt.SQL.String(), stmt.Vars
	}

	sql, vars := sqlOf("")
	if strings.Contains(sql, "LIKE") || len(vars) != 0 {
		t.Errorf("empty keyword should not filter: %s %v", sql, vars)
	}
	for _, want := range []string{"FROM v2_device_binding AS b", "LEFT JOIN v2_user u ON u.id = b.user_id", "v2_device_trial_log"} {
		if !strings.Contains(sql, want) {
			t.Errorf("sql missing %q: %s", want, sql)
		}
	}

	_, vars = sqlOf("abc")
	if len(vars) != 3 || vars[0] != "%abc%" {
		t.Errorf("text keyword vars: %v", vars)
	}
	sql, vars = sqlOf("123456789")
	if !strings.Contains(sql, "b.user_id = ?") || len(vars) != 4 || vars[3] != int64(123456789) {
		t.Errorf("numeric keyword should also match user id: %s %v", sql, vars)
	}
}

func TestParamString(t *testing.T) {
	p := map[string]any{"a": " x ", "n": nil, "i": 1}
	if paramString(p, "a") != "x" || paramString(p, "n") != "" || paramString(p, "missing") != "" || paramString(p, "i") != "1" {
		t.Error("paramString mismatch")
	}
}
