package warmpath

import (
	"path/filepath"
	"testing"
)

func TestBatchSpillAppendDrain(t *testing.T) {
	dir := t.TempDir()
	spill := NewBatchSpill(filepath.Join(dir, "spill.jsonl"))
	rec := SpillRecord{
		Key:    "warm-1",
		Events: []Event{{HashID: "h1", Snippet: "tracker pain"}},
	}
	if err := spill.Append(rec); err != nil {
		t.Fatal(err)
	}
	out, err := spill.Drain()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Key != "warm-1" {
		t.Fatalf("unexpected %+v", out)
	}
	again, err := spill.Drain()
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("expected empty drain, got %+v", again)
	}
}
