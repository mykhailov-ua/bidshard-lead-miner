package suggestions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/gemini"
)

func TestApplyKeywordTuneUpdatesWeightAndDisables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "keywords.json")
	base := keywordsFile{
		Keywords: []gemini.KeywordEntry{
			{ID: "kw_ok", Phrase: "voluum alternative", Weight: 20, Tag: "hot-lead"},
			{ID: "kw_bad", Phrase: "spam phrase", Weight: 20, Tag: "pain-point"},
		},
	}
	if err := writeJSON(path, base); err != nil {
		t.Fatal(err)
	}
	rows := []discover.KeywordTuneRow{
		{KeywordID: "kw_ok", SuggestedWeight: 25},
		{KeywordID: "kw_bad", RecommendDisable: true},
	}
	summary, err := ApplyKeywordTune(path, rows, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary == "" {
		t.Fatal("expected summary")
	}
	var after keywordsFile
	if err := readJSON(path, &after); err != nil {
		t.Fatal(err)
	}
	if len(after.Keywords) != 1 || after.Keywords[0].Weight != 25 {
		t.Fatalf("keywords=%+v", after.Keywords)
	}
	backups, _ := filepath.Glob(path + ".bak.*")
	if len(backups) == 0 {
		t.Fatal("expected backup file")
	}
	_ = os.Remove(backups[0])
}

func TestApplyKeywordTuneAutoRespectsQuota(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "keywords.json")
	statePath := filepath.Join(dir, "state.json")
	if err := writeJSON(path, keywordsFile{
		Keywords: []gemini.KeywordEntry{
			{ID: "kw_a", Phrase: "a", Weight: 20},
			{ID: "kw_b", Phrase: "b", Weight: 20},
		},
	}); err != nil {
		t.Fatal(err)
	}
	rows := []discover.KeywordTuneRow{
		{KeywordID: "kw_a", SuggestedWeight: 15},
		{KeywordID: "kw_b", SuggestedWeight: 15},
	}
	_, err := ApplyKeywordTuneAuto(path, rows, KeywordTuneAutoApplyOptions{
		MaxPerWeek: 1,
		StatePath:  statePath,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	var after keywordsFile
	if err := readJSON(path, &after); err != nil {
		t.Fatal(err)
	}
	changed := 0
	for _, e := range after.Keywords {
		if e.Weight == 15 {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("expected 1 change under quota, got %d", changed)
	}
}
