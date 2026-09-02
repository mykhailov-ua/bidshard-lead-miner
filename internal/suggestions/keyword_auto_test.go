package suggestions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/gemini"
)

func TestApplyKeywordsAutoBlocksDenylist(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	keywordsPath := filepath.Join(dir, "keywords.json")
	pendingPath := filepath.Join(dir, "keywords_pending_test.json")
	statePath := filepath.Join(dir, "auto_apply.json")

	base := `{"keywords":[{"id":"voluum","phrase":"voluum","weight":10}],"hard_reject":[]}`
	if err := os.WriteFile(keywordsPath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	pending := gemini.KeywordDiff{
		Status: "pending",
		AddKeywords: []gemini.KeywordEntry{
			{ID: "binom_migration", Phrase: "binom migration", Weight: 8},
			{ID: "linkedin_jobs", Phrase: "linkedin jobs affiliate", Weight: 5},
		},
	}
	if err := writeJSON(pendingPath, pending); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	summary, err := ApplyKeywordsAuto(pendingPath, keywordsPath, KeywordAutoApplyOptions{
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

	var merged keywordsFile
	if err := readJSON(keywordsPath, &merged); err != nil {
		t.Fatal(err)
	}
	if len(merged.Keywords) != 2 {
		t.Fatalf("keywords=%v", merged.Keywords)
	}

	var after gemini.KeywordDiff
	if err := readJSON(pendingPath, &after); err != nil {
		t.Fatal(err)
	}
	if after.Status != "applied" {
		t.Fatalf("status=%q", after.Status)
	}
}

func TestFilterKeywordDiffRejectsProgrammatic(t *testing.T) {
	t.Parallel()
	diff, blocked := FilterKeywordDiff(gemini.KeywordDiff{
		AddKeywords: []gemini.KeywordEntry{{Phrase: "programmatic guaranteed", Weight: 10}},
	})
	if len(diff.AddKeywords) != 0 {
		t.Fatalf("keywords=%v", diff.AddKeywords)
	}
	if blocked["programmatic guaranteed"] != "anti-icp-programmatic" {
		t.Fatalf("blocked=%v", blocked)
	}
}

func TestFilterKeywordDiffRejectsLinkedIn(t *testing.T) {
	t.Parallel()
	diff, blocked := FilterKeywordDiff(gemini.KeywordDiff{
		AddKeywords: []gemini.KeywordEntry{{Phrase: "linkedin recruiter"}},
	})
	if len(diff.AddKeywords) != 0 {
		t.Fatalf("keywords=%v", diff.AddKeywords)
	}
	if blocked["linkedin recruiter"] == "" {
		t.Fatal("expected block reason")
	}
}
