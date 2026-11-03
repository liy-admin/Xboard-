package jobs

import (
	"testing"
	"time"
)

func TestScheduleDueEveryMinute(t *testing.T) {
	s := schedule{Name: "x", Every: time.Minute}
	base := time.Date(2026, 10, 15, 12, 30, 0, 0, time.Local)
	ok, k1 := s.due(base)
	if !ok {
		t.Fatal("minute job should fire at second 0")
	}
	if ok2, _ := s.due(base.Add(30 * time.Second)); ok2 {
		t.Fatal("should not fire mid-minute")
	}
	if ok3, k3 := s.due(base.Add(time.Minute)); !ok3 || k3 == k1 {
		t.Fatal("next minute should fire with a new event key")
	}
}

func TestScheduleDueDaily(t *testing.T) {
	s := schedule{Name: "x", DailyAt: "00:00"}
	midnight := time.Date(2026, 10, 15, 0, 0, 0, 0, time.Local)
	if ok, _ := s.due(midnight); !ok {
		t.Fatal("daily job should fire at 00:00")
	}
	if ok, _ := s.due(midnight.Add(time.Hour)); ok {
		t.Fatal("daily job should not fire at 01:00")
	}
}

func TestDefaultSchedulesAreRunnable(t *testing.T) {
	for _, sc := range DefaultSchedules {
		if !IsRunnable(sc.Name) {
			t.Errorf("scheduled job %s has no implementation", sc.Name)
		}
	}
}
