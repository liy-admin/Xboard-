package singbox

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 把测试程序自身当作假的 sing-box：被以 `version` 参数启动且带标记环境变量时，打印版本号后退出。
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "version" && os.Getenv("FAKE_SINGBOX_VERSION") != "" {
		fmt.Printf("sing-box version %s\n\nEnvironment: go\n", os.Getenv("FAKE_SINGBOX_VERSION"))
		os.Exit(0)
	}
	os.Setenv("FAKE_SINGBOX_VERSION", "1.14.2")
	os.Exit(m.Run())
}

func selfBytes(t *testing.T) []byte {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// buildArchive 按当前平台的发布包格式（tar.gz / zip）打包，目录结构与官方一致。
func buildArchive(t *testing.T, version string, bin []byte) []byte {
	t.Helper()
	asset, err := AssetName(version)
	if err != nil {
		t.Skip(err)
	}
	inner := strings.TrimSuffix(strings.TrimSuffix(asset, ".tar.gz"), ".zip") + "/" + binName()
	var buf bytes.Buffer
	if strings.HasSuffix(asset, ".zip") {
		zw := zip.NewWriter(&buf)
		w, _ := zw.Create(inner)
		w.Write(bin)
		other, _ := zw.Create(strings.TrimSuffix(asset, ".zip") + "/LICENSE")
		other.Write([]byte("license"))
		zw.Close()
		return buf.Bytes()
	}
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: inner, Mode: 0o755, Size: int64(len(bin)), Typeflag: tar.TypeReg})
	tw.Write(bin)
	tw.WriteHeader(&tar.Header{Name: strings.Split(inner, "/")[0] + "/LICENSE", Mode: 0o644, Size: 7, Typeflag: tar.TypeReg})
	tw.Write([]byte("license"))
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func releaseServer(t *testing.T, version string, payload []byte, hits *int32) *httptest.Server {
	t.Helper()
	asset, err := AssetName(version)
	if err != nil {
		t.Skip(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.URL.Path != "/v"+version+"/"+asset {
			http.NotFound(w, r)
			return
		}
		w.Write(payload)
	}))
}

func TestValidVersion(t *testing.T) {
	for _, ok := range []string{"1.14.2", "1.12.0", "1.13.0-rc.1"} {
		if !ValidVersion(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "1.2", "../1.0.0", "1.0.0/../x", "v1.0.0", "1.0.0 ", "1.0.0;rm", "latest"} {
		if ValidVersion(bad) {
			t.Errorf("%q must be rejected (used in URL and path)", bad)
		}
	}
}

func TestEnsureDownloadsOnceAndCaches(t *testing.T) {
	var hits int32
	srv := releaseServer(t, "1.14.2", buildArchive(t, "1.14.2", selfBytes(t)), &hits)
	defer srv.Close()
	m := New(t.TempDir(), srv.URL)

	p, err := m.Ensure(context.Background(), "1.14.2")
	if err != nil {
		t.Fatal(err)
	}
	if p != m.BinPath("1.14.2") {
		t.Fatalf("path %s", p)
	}
	if v, ok := Works(p); !ok || v != "1.14.2" {
		t.Fatalf("installed binary not runnable: %q %v", v, ok)
	}
	if _, err := m.Ensure(context.Background(), "1.14.2"); err != nil {
		t.Fatal(err)
	}
	if h := atomic.LoadInt32(&hits); h != 1 {
		t.Fatalf("second Ensure must reuse the cached copy, server hits=%d", h)
	}
	// 临时文件必须清理干净，只剩下二进制
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("leftover files: %v", names)
	}
}

func TestEnsureRejectsVersionMismatch(t *testing.T) {
	var hits int32
	// 服务器上 9.9.9 的包里实际装的是 1.14.2 的程序
	srv := releaseServer(t, "9.9.9", buildArchive(t, "9.9.9", selfBytes(t)), &hits)
	defer srv.Close()
	m := New(t.TempDir(), srv.URL)
	if _, err := m.Ensure(context.Background(), "9.9.9"); err == nil || !strings.Contains(err.Error(), "版本号不匹配") {
		t.Fatalf("want version mismatch error, got %v", err)
	}
	if fileExists(m.BinPath("9.9.9")) {
		t.Fatal("mismatched binary must not be installed")
	}
}

func TestEnsureNotFoundIsFriendly(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	m := New(t.TempDir(), srv.URL)
	_, err := m.Ensure(context.Background(), "1.14.2")
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("want 404 message, got %v", err)
	}
}

func TestEnsureRejectsGarbageArchive(t *testing.T) {
	var hits int32
	srv := releaseServer(t, "1.14.2", bytes.Repeat([]byte("x"), 4096), &hits)
	defer srv.Close()
	m := New(t.TempDir(), srv.URL)
	if _, err := m.Ensure(context.Background(), "1.14.2"); err == nil {
		t.Fatal("garbage archive must fail")
	}
	if fileExists(m.BinPath("1.14.2")) {
		t.Fatal("nothing should be installed")
	}
}

func TestStartReportsProgressUntilDone(t *testing.T) {
	var hits int32
	srv := releaseServer(t, "1.14.2", buildArchive(t, "1.14.2", selfBytes(t)), &hits)
	defer srv.Close()
	m := New(t.TempDir(), srv.URL)

	st, err := m.Start("1.14.2")
	if err != nil || st.State != "downloading" {
		t.Fatalf("start: %+v %v", st, err)
	}
	if _, err := m.Start("not a version"); err == nil {
		t.Fatal("invalid version must be rejected")
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		st = m.Status()
		if st.State == "done" || st.State == "error" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if st.State != "done" || st.Path != m.BinPath("1.14.2") {
		t.Fatalf("final status: %+v", st)
	}
}

func TestResolveOrder(t *testing.T) {
	exe, _ := os.Executable()
	m := New(t.TempDir(), "http://127.0.0.1:1") // 不应被访问
	// 1) 自定义路径可用 → 直接用
	bin, ver, msg := m.Resolve(context.Background(), exe, "1.14.2", false)
	if bin != exe || ver != "1.14.2" || msg != "" {
		t.Fatalf("custom: %q %q %q", bin, ver, msg)
	}
	// 2) 都没有且关闭自动下载 → 给出明确原因，不联网
	bin, _, msg = m.Resolve(context.Background(), filepath.Join(t.TempDir(), "missing"), "1.14.2", false)
	if bin != "" || !strings.Contains(msg, "自动下载已关闭") {
		// PATH 里恰好有真实 sing-box 时会命中，此时视为通过
		if bin == "" {
			t.Fatalf("unexpected: %q %q", bin, msg)
		}
	}
	// 3) 非法版本号
	if _, _, msg := m.Resolve(context.Background(), "", "../x", true); !strings.Contains(msg, "版本号格式不正确") && msg != "" {
		t.Fatalf("invalid version: %q", msg)
	}
}
