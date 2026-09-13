package telethon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionPathEnvOverride(t *testing.T) {
	t.Setenv("TELEGRAM_SESSION", "data/runtime/telethon.session.1")
	got := SessionPath("")
	if got != "data/runtime/telethon.session.1" {
		t.Fatalf("SessionPath()=%q want env override", got)
	}
}

func TestSessionPathYAMLFallback(t *testing.T) {
	t.Setenv("TELEGRAM_SESSION", "")

	dir := t.TempDir()
	cfg := filepath.Join(dir, "sources.telegram.yaml")
	if err := os.WriteFile(cfg, []byte("session: data/runtime/telethon.session.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := SessionPath(cfg)
	if got != "data/runtime/telethon.session.2" {
		t.Fatalf("SessionPath()=%q want yaml session", got)
	}
}
