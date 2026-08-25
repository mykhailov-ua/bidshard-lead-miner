package suggestions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/discover"
)

func TestApplyDiscoverAutoBlocksDenylist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	icpPath := filepath.Join(dir, "discover.icp.json")
	pendingPath := filepath.Join(dir, "discover_icp_pending_test.json")
	statePath := filepath.Join(dir, "auto_apply.json")

	base := `{"telegram_search":["voluum alternative"],"serp_dorks":["site:t.me voluum"]}`
	if err := os.WriteFile(icpPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	pending := discover.PendingICPDiff{
		Status:            "pending",
		AddTelegramSearch: []string{"binom migration pain"},
		AddSerpDorks:      []string{"site:t.me binom", "site:linkedin.com affiliate jobs"},
	}
	if err := writeJSON(pendingPath, pending); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	summary, err := ApplyDiscoverAuto(pendingPath, icpPath, DiscoverAutoApplyOptions{
		MaxPerWeek: 10,
		StatePath:  statePath,
		Now:        now,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary == "" {
		t.Fatal("expected summary")
	}

	var merged discover.ICPConfig
	if err := readJSON(icpPath, &merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.TelegramSearch) != 2 {
		t.Fatalf("telegram_search=%v", merged.TelegramSearch)
	}
	if len(merged.SerpDorks) != 2 {
		t.Fatalf("serp_dorks=%v", merged.SerpDorks)
	}

	var after discover.PendingICPDiff
	if err := readJSON(pendingPath, &after); err != nil {
		t.Fatal(err)
	}
	if after.Status != "pending" {
		t.Fatalf("status=%q want pending (linkedin blocked)", after.Status)
	}
	if len(after.AddSerpDorks) != 1 || after.AddSerpDorks[0] != "site:linkedin.com affiliate jobs" {
		t.Fatalf("pending serp=%v", after.AddSerpDorks)
	}
}
