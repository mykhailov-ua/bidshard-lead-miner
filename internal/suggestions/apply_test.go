package suggestions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/gemini"
)

func TestApplyKeywordsDryRunDoesNotWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	keywordsPath := filepath.Join(dir, "keywords.json")
	pendingPath := filepath.Join(dir, "keywords_pending_test.json")

	base := `{"priority":{"high_min":35,"medium_min":15},"keywords":[],"titles":[],"negative":[],"hard_reject":[]}`
	if err := os.WriteFile(keywordsPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	pending := gemini.KeywordDiff{
		Status: "pending",
		AddKeywords: []gemini.KeywordEntry{
			{ID: "kw-new", Phrase: "voluum alternative", Weight: 25, Tag: "intent"},
		},
	}
	if err := writeJSON(pendingPath, pending); err != nil {
		t.Fatal(err)
	}

	summary, err := ApplyKeywords(pendingPath, keywordsPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if summary == "" {
		t.Fatal("expected summary")
	}

	var after keywordsFile
	if err := readJSON(keywordsPath, &after); err != nil {
		t.Fatal(err)
	}
	if len(after.Keywords) != 0 {
		t.Fatalf("dry-run wrote keywords: %+v", after.Keywords)
	}
}

func TestApplyKeywordsMergesAndMarksApplied(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	keywordsPath := filepath.Join(dir, "keywords.json")
	pendingPath := filepath.Join(dir, "keywords_pending_test.json")

	base := `{"priority":{"high_min":35,"medium_min":15},"keywords":[],"titles":[],"negative":[],"hard_reject":[]}`
	if err := os.WriteFile(keywordsPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	pending := gemini.KeywordDiff{
		Status: "pending",
		AddKeywords: []gemini.KeywordEntry{
			{ID: "kw-new", Phrase: "voluum alternative", Weight: 25, Tag: "intent"},
		},
		AddHardReject: []gemini.KeywordEntry{
			{ID: "hr-new", Phrase: "hiring media buyer", Tag: "hard-reject"},
		},
	}
	if err := writeJSON(pendingPath, pending); err != nil {
		t.Fatal(err)
	}

	if _, err := ApplyKeywords(pendingPath, keywordsPath, false); err != nil {
		t.Fatal(err)
	}
	var merged keywordsFile
	if err := readJSON(keywordsPath, &merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.Keywords) != 1 || len(merged.HardReject) != 1 {
		t.Fatalf("merge failed: %+v", merged)
	}
	var status gemini.KeywordDiff
	if err := readJSON(pendingPath, &status); err != nil {
		t.Fatal(err)
	}
	if status.Status != "applied" {
		t.Fatalf("status=%q want applied", status.Status)
	}
}
