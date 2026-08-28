package discover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/sink"
)

func TestWriteOutcomeDorkReport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	channels := filepath.Join(dir, "channels.json")
	if err := os.WriteFile(channels, []byte(`{"channels":[{"username":"aff_net","query":"voluum alternative telegram"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	stats := []sink.SourceStatsDoc{
		{Source: "telegram:aff_net", Accepted: 10, Junk: 2},
	}
	outcomes := []OutcomeSourceRow{
		{Source: "telegram:aff_net", Outcome: "pilot_started", Count: 1},
	}
	path, err := WriteOutcomeDorkReport(channels, dir, stats, outcomes)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("expected report body")
	}
}
