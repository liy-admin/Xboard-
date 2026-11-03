package jobs

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
)

// fakeRunner 记录调用并按顺序返回预设结果，避免测试真的去 pip 安装。
type fakeCall struct {
	name string
	args []string
}

func withFakeRunner(t *testing.T, results []struct {
	out string
	err error
}) *[]fakeCall {
	t.Helper()
	orig := runCmd
	var calls []fakeCall
	i := 0
	runCmd = func(ctx context.Context, name string, args ...string) (string, error) {
		calls = append(calls, fakeCall{name, append([]string(nil), args...)})
		if i >= len(results) {
			return "", errors.New("unexpected extra call")
		}
		r := results[i]
		i++
		return r.out, r.err
	}
	t.Cleanup(func() { runCmd = orig })
	return &calls
}

type res = struct {
	out string
	err error
}

func TestResolvePython(t *testing.T) {
	if got := ResolvePython("python3.11"); got != "python3.11" {
		t.Errorf("custom command must be kept: %q", got)
	}
	if got := ResolvePython("/usr/bin/python3"); got != "/usr/bin/python3" {
		t.Errorf("custom path must be kept: %q", got)
	}
	want := "python3"
	if runtime.GOOS == "windows" {
		want = "python"
	}
	for _, in := range []string{"", "  ", "python"} {
		if got := ResolvePython(in); got != want {
			t.Errorf("ResolvePython(%q)=%q want %q (default 'python' is not present on most Linux hosts)", in, got, want)
		}
	}
}

func TestPipInstallRetriesOnPEP668(t *testing.T) {
	calls := withFakeRunner(t, []res{
		{"error: externally-managed-environment\n× This environment is externally managed", errors.New("exit 1")},
		{"Successfully installed ruyipage", nil},
	})
	out, err := pipInstallRuyipage(context.Background(), "python3")
	if err != nil {
		t.Fatalf("must succeed after the retry: %v", err)
	}
	if len(*calls) != 2 {
		t.Fatalf("want 2 calls, got %d", len(*calls))
	}
	if strings.Contains(strings.Join((*calls)[0].args, " "), "break-system-packages") {
		t.Error("first attempt must not use --break-system-packages")
	}
	if !strings.Contains(strings.Join((*calls)[1].args, " "), "--break-system-packages") {
		t.Errorf("retry must add --break-system-packages: %v", (*calls)[1].args)
	}
	if !strings.Contains(out, "Successfully installed") {
		t.Errorf("output must include the retry result: %q", out)
	}
}

func TestPipInstallDoesNotRetryOtherFailures(t *testing.T) {
	calls := withFakeRunner(t, []res{{"ERROR: No matching distribution found", errors.New("exit 1")}})
	if _, err := pipInstallRuyipage(context.Background(), "python3"); err == nil {
		t.Fatal("must report the failure")
	}
	if len(*calls) != 1 {
		t.Fatalf("non-PEP668 failures must not be retried with --break-system-packages, calls=%d", len(*calls))
	}
}

func TestBootstrapEnvFallsBackToNextPython(t *testing.T) {
	calls := withFakeRunner(t, []res{
		{`exec: "python3": executable file not found in $PATH`, errors.New("not found")},
		{"Successfully installed ruyipage", nil},
	})
	var log bytes.Buffer
	used, err := bootstrapEnv(context.Background(), []string{"python3", "python"}, &log)
	if err != nil || used != "python" {
		t.Fatalf("used=%q err=%v", used, err)
	}
	if (*calls)[0].name != "python3" || (*calls)[1].name != "python" {
		t.Errorf("order: %v", *calls)
	}
	l := log.String()
	if !strings.Contains(l, "python3 失败") || !strings.Contains(l, "安装完成 (python)") {
		t.Errorf("log must record both attempts:\n%s", l)
	}
}

func TestBootstrapEnvReportsFailureWhenAllFail(t *testing.T) {
	withFakeRunner(t, []res{{"x", errors.New("a")}, {"y", errors.New("b")}})
	var log bytes.Buffer
	if _, err := bootstrapEnv(context.Background(), []string{"python3", "python"}, &log); err == nil {
		t.Fatal("must fail when every candidate fails")
	}
}

func TestBootstrapCandidates(t *testing.T) {
	got := bootstrapCandidates("")
	if runtime.GOOS == "windows" {
		if strings.Join(got, ",") != "python" {
			t.Errorf("windows: %v", got)
		}
	} else if strings.Join(got, ",") != "python3,python" {
		t.Errorf("linux default must try python3 first: %v", got)
	}
	if got := bootstrapCandidates("/opt/py/bin/python3"); got[0] != "/opt/py/bin/python3" {
		t.Errorf("custom command must come first: %v", got)
	}
}
