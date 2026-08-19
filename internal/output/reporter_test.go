package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
)

func TestReporterTable(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewReporter("table", &buf)
	r.handle(pipeline.RoundStats{
		RoundID:  "abc123",
		Duration: 1500 * time.Millisecond,
		SourcesOK: 2,
		RawTotal: 3,
		Accepted: 3,
		High:     1,
		Leads: []model.Lead{{
			RoundID:  "abc123",
			Priority: "High",
			Score:    62,
			Source:   "stub:telegram_en",
			Contacts: []string{"telegram:@buyer"},
			Matched:  []string{"voluum alternative"},
			Snippet:  "postback failing",
		}},
	})

	out := buf.String()
	if !strings.Contains(out, "scan round abc123") {
		t.Fatalf("missing table header: %q", out)
	}
	if !strings.Contains(out, "HIGH  score=62") {
		t.Fatalf("missing lead row: %q", out)
	}
}

func TestReporterNDJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	r := NewReporter("ndjson", &buf)
	r.handle(pipeline.RoundStats{
		Leads: []model.Lead{{
			RoundID:  "abc123",
			Priority: "High",
			Score:    62,
			Source:   "stub:telegram_en",
			Contacts: []string{"telegram:@buyer"},
			Matched:  []string{"voluum alternative"},
		}},
	})

	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, `"source":"stub:telegram_en"`) {
		t.Fatalf("missing ndjson line: %q", out)
	}
}
