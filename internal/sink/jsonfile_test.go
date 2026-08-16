package sink

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestJSONFileSinkAppendsLead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "leads.jsonl")
	sink, err := NewJSONFileSink(path)
	if err != nil {
		t.Fatal(err)
	}

	lead := model.Lead{
		TS:       time.Now().UTC(),
		HashID:   "abc",
		Priority: "High",
		Score:    10,
		Source:   "stub:test",
		Contacts: []string{"ops@igaming-team.com"},
		Matched:  []string{"voluum"},
		Snippet:  "voluum alternative",
	}
	if err := sink.Upsert(context.Background(), lead); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var line map[string]any
	if err := json.Unmarshal(raw, &line); err != nil {
		t.Fatal(err)
	}
	if line["hash_id"] != "abc" {
		t.Fatalf("hash_id=%v", line["hash_id"])
	}
	if line["source"] != "stub:test" {
		t.Fatalf("source=%v", line["source"])
	}
}
