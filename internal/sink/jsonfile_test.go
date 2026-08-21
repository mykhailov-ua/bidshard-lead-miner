package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestJSONFileSinkCreatesParentDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "export", "leads.jsonl")
	sink, err := NewJSONFileSink(path, "")
	if err != nil {
		t.Fatal(err)
	}

	lead := model.Lead{HashID: "abc", Source: "stub:test", Score: 10}
	if err := sink.Upsert(context.Background(), lead); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
}

func TestJSONFileSinkAppendsLead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "leads.jsonl")
	sink, err := NewJSONFileSink(path, "")
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

func TestJSONFileSinkPrettyJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "leads.json")
	sink, err := NewJSONFileSink(path, "pretty")
	if err != nil {
		t.Fatal(err)
	}

	lead := model.Lead{
		HashID: "abc",
		Source: "stub:test",
		Score:  10,
	}
	if err := sink.Upsert(context.Background(), lead); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("\"hash_id\": \"abc\"")) {
		t.Fatalf("expected pretty json export, got: %s", raw)
	}
}
