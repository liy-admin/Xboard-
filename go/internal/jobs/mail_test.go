package jobs

import "testing"

func TestMailTrafficWarnRange(t *testing.T) {
	cases := []struct {
		name     string
		u, d, tr int64
		want     bool
	}{
		{"80 percent warns", 40, 40, 100, true},
		{"99 percent warns", 50, 49, 100, true},
		{"100 percent does not", 50, 50, 100, false},
		{"20 percent does not", 10, 10, 100, false},
		{"no usage does not", 0, 0, 100, false},
		{"no quota does not", 1, 1, 0, false},
	}
	for _, c := range cases {
		if got := mailTrafficWarn(c.u, c.d, c.tr); got != c.want {
			t.Errorf("%s: want %v got %v", c.name, c.want, got)
		}
	}
}

func TestDefaultSchedulesIncludeNewJobs(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range DefaultSchedules {
		seen[s.Name] = true
	}
	for _, n := range []string{"send:remindMail", "xboard:statistics"} {
		if !seen[n] || !IsRunnable(n) {
			t.Errorf("%s must be scheduled and runnable", n)
		}
	}
}
