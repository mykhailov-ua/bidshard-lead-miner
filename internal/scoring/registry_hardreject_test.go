package scoring

import (
	"context"
	"testing"
)

func TestHardRejectTagOnNegative(t *testing.T) {
	t.Parallel()

	reg := NewRegistry("")
	file := &keywordsFile{
		Keywords: []keywordEntry{{ID: "kw1", Phrase: "voluum alternative", Weight: 25, Tag: "intent"}},
		Negative: []keywordEntry{
			{ID: "neg-soft", Phrase: "promo code", Weight: -8, Tag: "negative"},
			{ID: "neg-hard", Phrase: "free alternative", Weight: -10, Tag: "hard-reject"},
		},
	}
	file = normalizeKeywordFile(file)
	reg.mu.Lock()
	reg.keywordRules = rulesFromEntries(file.Keywords)
	reg.negativeRules = rulesFromEntries(file.Negative)
	reg.hardRejectRules = rulesFromEntries(file.HardReject)
	reg.mu.Unlock()

	if hit, ok := reg.HardReject("looking for free alternative tracker"); !ok || hit.Phrase != "free alternative" {
		t.Fatalf("hard reject miss: ok=%v hit=%+v", ok, hit)
	}
	result := AnalyzeWithRegistry(reg, "promo code voluum alternative")
	if result.Score <= 0 {
		t.Fatalf("soft negative should still score: %d", result.Score)
	}
	_ = context.Background()
}
