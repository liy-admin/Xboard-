package schema

import (
	"strings"
	"testing"
)

func TestEmbeddedSchemaIsComplete(t *testing.T) {
	ddl, err := files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(ddl), "CREATE TABLE IF NOT EXISTS"); n < 30 {
		t.Fatalf("expected the full table set, got %d CREATE TABLE statements", n)
	}
	// 幂等：不允许出现不带 IF NOT EXISTS 的建表。
	if strings.Contains(string(ddl), "CREATE TABLE `") {
		t.Fatal("found CREATE TABLE without IF NOT EXISTS")
	}
	for _, table := range []string{"v2_user", "v2_server", "v2_order", "v2_plan", "v2_payment", "v2_settings", "v2_plugins"} {
		if !strings.Contains(string(ddl), "`"+table+"`") {
			t.Errorf("missing table %s", table)
		}
	}
	seed, err := files.ReadFile("seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(seed), "`v2_subscribe_templates`") {
		t.Error("seed lacks subscribe templates")
	}
}

func TestJSONColumnConversionForMySQL(t *testing.T) {
	in := "`conditions` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL COMMENT '使用条件' CHECK (json_valid(`conditions`)),"
	got := jsonColumn.ReplaceAllString(in, "json$1$2")
	want := "`conditions` json DEFAULT NULL COMMENT '使用条件',"
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
	notNull := "`x` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`x`)),"
	if got := jsonColumn.ReplaceAllString(notNull, "json$1$2"); got != "`x` json NOT NULL," {
		t.Fatalf("not null: %q", got)
	}
}

func TestSchemaHasDeviceTrialLog(t *testing.T) {
	ddl, err := files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ddl), "CREATE TABLE IF NOT EXISTS `v2_device_trial_log`") {
		t.Fatal("schema lacks v2_device_trial_log")
	}
	// 补列清单里的列必须同时出现在 schema.sql（全新库建表）中，保证新老库结构一致。
	for _, col := range addedColumns {
		if !strings.Contains(string(ddl), "`"+col.Column+"`") {
			t.Errorf("schema.sql lacks column %s.%s", col.Table, col.Column)
		}
	}
}
