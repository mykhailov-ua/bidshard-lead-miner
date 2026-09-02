package discover

import (
	"path/filepath"
	"testing"
)

func TestViolatingProgrammaticDorks(t *testing.T) {
	t.Parallel()

	got := ViolatingProgrammaticDorks(
		[]string{"voluum alternative"},
		[]string{"site:t.me postback failing", "openrtb bidder jobs"},
	)
	if len(got) != 1 || got[0] != "openrtb bidder jobs" {
		t.Fatalf("got=%v", got)
	}
}

func TestDiscoverICPNoProgrammaticDorks(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "config", "discover.icp.json")
	cfg, err := LoadICP(path)
	if err != nil {
		t.Fatal(err)
	}
	if bad := ViolatingProgrammaticDorks(cfg.TelegramSearch, cfg.SerpDorks); len(bad) > 0 {
		t.Fatalf("programmatic dorks in discover.icp.json: %v", bad)
	}
}
