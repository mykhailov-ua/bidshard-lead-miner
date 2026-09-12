package warmpath

import (
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/gemini"
)

func TestEngageSpillAppendDrain(t *testing.T) {
	dir := t.TempDir()
	spill := NewEngageSpill(filepath.Join(dir, "engage.jsonl"))
	rec := EngageSpillRecord{
		Key: "engage-1",
		Items: []EngageSpillItem{{
			Event: Event{HashID: "h1", Priority: "High"},
			Core:  gemini.LeadBatchResult{HashID: "h1", ICP: gemini.ICPResult{ICP: "pro"}},
		}},
	}
	if err := spill.Append(rec); err != nil {
		t.Fatal(err)
	}
	out, err := spill.Drain()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Items[0].Core.ICP.ICP != "pro" {
		t.Fatalf("unexpected %+v", out)
	}
}
