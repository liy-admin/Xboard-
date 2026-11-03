package jobs

import (
	"testing"
	"time"

	"xboard/internal/setting"
)

func TestNormalizeCleanupDays(t *testing.T) {
	cases := map[int]int{
		-5: DefaultCleanupDays, 0: DefaultCleanupDays, 1: 1, 7: 7, 30: 30, 3650: 3650, 3651: MaxCleanupDays, 99999: MaxCleanupDays,
	}
	for in, want := range cases {
		if got := NormalizeCleanupDays(in); got != want {
			t.Errorf("NormalizeCleanupDays(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestCleanupCutoffUsesWholeDays(t *testing.T) {
	now := time.Date(2026, 10, 19, 3, 30, 0, 0, time.UTC)
	got := cleanupCutoff(now, 30)
	want := now.Add(-30 * 24 * time.Hour).Unix()
	if got != want {
		t.Fatalf("cutoff %d want %d", got, want)
	}
	if cleanupCutoff(now, 1) >= now.Unix() {
		t.Fatal("cutoff must be in the past")
	}
}

func TestCleanupSettingsDefaultsAreSafe(t *testing.T) {
	// 没有任何设置：删除数据必须由管理员主动打开，缺省关闭；天数缺省 30。
	e := &Env{Set: setting.NewFromMap(nil)}
	if e.CleanupEnabled() {
		t.Fatal("auto cleanup must be OFF by default")
	}
	if e.CleanupDays() != DefaultCleanupDays {
		t.Fatalf("default days = %d", e.CleanupDays())
	}
	// PHP 风格：保存 false 时写入空串，仍是「关闭」而不是回落到缺省。
	e = &Env{Set: setting.NewFromMap(map[string]string{SettingCleanupEnable: "", SettingCleanupDays: "7"})}
	if e.CleanupEnabled() || e.CleanupDays() != 7 {
		t.Fatalf("enabled=%v days=%d", e.CleanupEnabled(), e.CleanupDays())
	}
	e = &Env{Set: setting.NewFromMap(map[string]string{SettingCleanupEnable: "1", SettingCleanupDays: "0"})}
	if !e.CleanupEnabled() || e.CleanupDays() != DefaultCleanupDays {
		t.Fatalf("invalid stored days must fall back: enabled=%v days=%d", e.CleanupEnabled(), e.CleanupDays())
	}
}

func TestScheduledCleanupIsNoopWhenDisabled(t *testing.T) {
	// 关闭时调度入口必须在碰数据库之前返回（DB 为 nil，若误删会直接 panic）。
	e := &Env{Set: setting.NewFromMap(nil), Now: time.Now}
	if err := e.ScheduledCleanupCancelled(); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupJobIsSchedulerOnlyAndScheduled(t *testing.T) {
	const name = "cleanup:cancelled-orders"
	if !IsRunnable(name) {
		t.Fatal("scheduler must be able to run it")
	}
	if IsCLI(name) {
		t.Fatal("new features have no command-line entry; admin button + scheduler only")
	}
	found := false
	for _, s := range DefaultSchedules {
		if s.Name == name {
			found = true
			if s.DailyAt != CleanupAt {
				t.Errorf("scheduled at %q, want %q", s.DailyAt, CleanupAt)
			}
		}
	}
	if !found {
		t.Fatal("not in DefaultSchedules")
	}
	if _, err := time.Parse("15:04", CleanupAt); err != nil {
		t.Fatalf("CleanupAt must be HH:MM: %v", err)
	}
}
