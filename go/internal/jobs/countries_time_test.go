package jobs

import (
	"testing"
	"time"

	"xboard/internal/setting"
)

func TestNormalizeHHMM(t *testing.T) {
	for in, want := range map[string]string{"04:00": "04:00", "4:05": "04:05", " 23:59 ": "23:59", "00:00": "00:00"} {
		if got, ok := NormalizeHHMM(in); !ok || got != want {
			t.Errorf("%q => %q %v, want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "24:00", "12:60", "noon", "12", "12:3x", "4pm"} {
		if got, ok := NormalizeHHMM(bad); ok {
			t.Errorf("%q must be rejected, got %q", bad, got)
		}
	}
}

func TestCountryRefreshTimeDefault(t *testing.T) {
	e := &Env{Set: setting.New(nil)}
	if got := e.CountryRefreshTime(); got != DefaultCountryRefreshTime {
		t.Fatalf("default time: %q", got)
	}
}

// 设置里的时刻改变后，调度器下一次检查就按新时刻触发（无需重启）。
func TestScheduleResolveUsesConfiguredTime(t *testing.T) {
	sc := schedule{Name: "x", DailyAt: "04:00", TimeFn: func(*Env) string { return "13:37" }}
	r := sc.resolve(&Env{})
	at := func(h, m int) time.Time { return time.Date(2026, 10, 16, h, m, 0, 0, time.Local) }
	if ok, _ := r.due(at(13, 37)); !ok {
		t.Error("must fire at the configured time")
	}
	if ok, _ := r.due(at(4, 0)); ok {
		t.Error("must not fire at the old default time once reconfigured")
	}
	// 没有 TimeFn 的任务不受影响
	fixed := schedule{Name: "y", DailyAt: "00:10"}.resolve(&Env{})
	if ok, _ := fixed.due(at(0, 10)); !ok {
		t.Error("fixed schedule must keep its time")
	}
}

// 新功能只做后台按钮 + 调度器，不提供命令行入口。
func TestCountryRefreshHasNoCLI(t *testing.T) {
	if !IsRunnable("refresh:countries") {
		t.Fatal("scheduler must still be able to run it")
	}
	if IsCLI("refresh:countries") {
		t.Fatal("refresh:countries must not be exposed as a command")
	}
	for _, n := range Names() {
		if n == "refresh:countries" {
			t.Fatal("must not be listed among CLI jobs")
		}
	}
	if !IsCLI("check:order") {
		t.Fatal("existing PHP-compatible commands stay available")
	}
}
