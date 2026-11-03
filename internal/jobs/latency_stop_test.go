package jobs

import (
	"testing"
	"time"
)

func TestRequestLatencyStopOnlyWhenRunning(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	e := &Env{Redis: newMemRedis(), Now: func() time.Time { return now }}

	// 没有测速在跑：不应留下停止标记（否则会把下一轮测速误停）
	if e.RequestLatencyStop() {
		t.Fatal("nothing is running")
	}
	if e.LatencyStopRequested() {
		t.Fatal("no stop flag may be left behind when nothing is running")
	}

	// 正在跑：写入标记
	e.writeLatencyStatus(LatencyStatus{Running: true, Total: 10, Done: 3})
	if !e.RequestLatencyStop() || !e.LatencyStopRequested() {
		t.Fatal("stop must be recorded while a probe is running")
	}

	// 新一轮开始时清掉遗留标记
	e.clearLatencyStop()
	if e.LatencyStopRequested() {
		t.Fatal("flag must be cleared")
	}
}

func TestRequestLatencyStopIgnoresStaleStatus(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	e := &Env{Redis: newMemRedis(), Now: func() time.Time { return now }}
	e.writeLatencyStatus(LatencyStatus{Running: true})
	// 状态停在 running 但超过 2 分钟没更新：进程多半已退出，不算在跑
	now = now.Add(3 * time.Minute)
	if e.RequestLatencyStop() {
		t.Fatal("a stale running status must not count as running")
	}
}
