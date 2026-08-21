package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadICP(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "config", "discover.icp.json")
	cfg, err := LoadICP(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.TelegramSearch) < 5 {
		t.Fatalf("telegram_search too short: %d", len(cfg.TelegramSearch))
	}
	if len(cfg.SerpDorks) < 5 {
		t.Fatalf("serp_dorks too short: %d", len(cfg.SerpDorks))
	}
}

func TestResolveICPPath(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	p := ResolveICPPath(wd)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("resolved path missing: %s err=%v", p, err)
	}
}
