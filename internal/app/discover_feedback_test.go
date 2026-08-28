package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sink"
)

func TestRunDiscoverFeedbackPrunesWeakDork(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	channels := filepath.Join(dir, "channels.json")
	if err := os.WriteFile(channels, []byte(`{"channels":[{"username":"bad_chan","query":"bad dork query"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	disabledPath := filepath.Join(dir, "disabled_dorks.json")
	outDir := filepath.Join(dir, "suggestions")

	cfg := config.Config{
		TelegramChannelsPath: channels,
		DisabledDorksPath:    disabledPath,
		GeminiKeywordDiffDir: outDir,
		DorkDisableMinRaw:    10,
		DorkDisableMaxAcceptRate: 0.05,
	}

	stats := []sink.SourceStatsDoc{{Source: "telegram:bad_chan", Accepted: 1, Junk: 19}}
	result, err := RunDiscoverFeedback(context.Background(), cfg, nil, &stubSourceStats{docs: stats}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.PrunedDorks != 1 {
		t.Fatalf("pruned=%d want 1", result.PrunedDorks)
	}
	raw, err := os.ReadFile(disabledPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("expected disabled dorks file")
	}
}

type stubSourceStats struct {
	docs []sink.SourceStatsDoc
}

func (s *stubSourceStats) ListAll(ctx context.Context) ([]sink.SourceStatsDoc, error) {
	return s.docs, nil
}

func (s *stubSourceStats) RecordAccepted(source string) {}
func (s *stubSourceStats) RecordJunk(source string)      {}
func (s *stubSourceStats) Boost(source string) int      { return 0 }
