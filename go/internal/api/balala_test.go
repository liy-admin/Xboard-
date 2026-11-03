package api

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func makeTarGz(t *testing.T, name string, body string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	_, _ = tw.Write([]byte(body))
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestUntarGzSafeRejectsTraversal(t *testing.T) {
	dest := t.TempDir()
	if err := untarGzSafe(makeTarGz(t, "../escape.txt", "x"), dest); err == nil {
		t.Fatal("path traversal must be rejected")
	}
}

func TestUntarGzSafeAndLocateFirefox(t *testing.T) {
	dest := t.TempDir()
	if err := untarGzSafe(makeTarGz(t, "ff/firefox", "bin"), dest); err != nil {
		t.Fatalf("extract: %v", err)
	}
	got := locateFirefox(dest)
	if filepath.Base(got) != "firefox" {
		t.Fatalf("locateFirefox = %q", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("located file missing: %v", err)
	}
}

func TestLocateFirefoxPrefersExe(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "firefox"), []byte("x"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "firefox.exe"), []byte("x"), 0o755)
	if filepath.Base(locateFirefox(dir)) != "firefox.exe" {
		t.Fatal("firefox.exe should win over firefox")
	}
	if locateFirefox(t.TempDir()) != "" {
		t.Fatal("empty dir must return empty string")
	}
}
