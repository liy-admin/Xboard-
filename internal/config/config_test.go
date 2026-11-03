package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, ".env")
	body := "# comment\nDB_HOST=db.internal\nDB_PASSWORD=\"p w#d\"\nexport REDIS_PORT=6380 # trailing\nEMPTY=\nQUOTED='single'\n"
	if err := os.WriteFile(f, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XBOARD_ENV_FILE", f)
	t.Setenv("DB_USERNAME", "from-real-env")
	for _, k := range []string{"DB_HOST", "DB_PASSWORD", "REDIS_PORT", "QUOTED"} {
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for _, k := range []string{"DB_HOST", "DB_PASSWORD", "REDIS_PORT", "QUOTED", "EMPTY"} {
			os.Unsetenv(k)
		}
	})
	if got := LoadDotEnv(); got != f {
		t.Fatalf("path %q", got)
	}
	cfg := Load()
	if cfg.DBHost != "db.internal" || cfg.DBPass != "p w#d" {
		t.Fatalf("host/pass: %q %q", cfg.DBHost, cfg.DBPass)
	}
	if cfg.RedisAddr != "127.0.0.1:6380" {
		t.Fatalf("redis addr %q", cfg.RedisAddr)
	}
	if os.Getenv("QUOTED") != "single" {
		t.Fatalf("single quotes: %q", os.Getenv("QUOTED"))
	}
	if cfg.DBUser != "from-real-env" {
		t.Fatalf("real env must win over the file, got %q", cfg.DBUser)
	}
	if cfg.Timezone != "Asia/Shanghai" {
		t.Fatalf("timezone %q", cfg.Timezone)
	}
}
