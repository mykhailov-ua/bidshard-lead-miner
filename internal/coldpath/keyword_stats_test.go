package coldpath

import (
	"context"
	"testing"

	"github.com/bidshard/parser/internal/gemini"
)

func TestEnrichKeywordDiffWithStats(t *testing.T) {
	enabled := true
	diff := gemini.KeywordDiff{
		AddKeywords: []gemini.KeywordEntry{
			{ID: "kw-test", Phrase: "voluum alternative", Weight: 20},
		},
	}
	out := enrichKeywordDiffWithStats(context.Background(), nil, diff)
	if len(out.AddKeywords) != 1 || out.AddKeywords[0].SuggestedWeight != 0 {
		t.Fatalf("unexpected diff without store: %+v", out.AddKeywords[0])
	}

	_ = enabled
}
