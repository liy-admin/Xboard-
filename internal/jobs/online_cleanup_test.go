package jobs

import (
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// last_online_at 是 MySQL timestamp 列。传 Unix 整数会被当成 YYYYMMDDHHMMSS 数字去比较，
// 条件永远不成立，过期的在线数从不清零；必须传时间值。
func TestCleanupOnlineStatusComparesWithTime(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/x", SkipInitializeWithVersion: true}),
		&gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	var vars []any
	var sql string
	if err := db.Callback().Raw().After("gorm:raw").Register("capture", func(tx *gorm.DB) {
		vars, sql = tx.Statement.Vars, tx.Statement.SQL.String()
	}); err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	e := &Env{DB: db, Now: func() time.Time { return fixed }}
	if err := e.CleanupOnlineStatus(); err != nil {
		t.Fatal(err)
	}
	if len(vars) != 1 {
		t.Fatalf("want 1 bound parameter, got %v (sql %s)", vars, sql)
	}
	cutoff, ok := vars[0].(time.Time)
	if !ok {
		t.Fatalf("cutoff must be time.Time, got %T", vars[0])
	}
	if want := fixed.Add(-300 * time.Second); !cutoff.Equal(want) {
		t.Fatalf("cutoff %v, want %v (设备在 Redis 里只保留 300 秒)", cutoff, want)
	}
}
