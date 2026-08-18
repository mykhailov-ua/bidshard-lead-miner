package coldpath

import (
	"context"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

func enrichKeywordDiffWithStats(ctx context.Context, store *sink.KeywordStatsStore, diff gemini.KeywordDiff) gemini.KeywordDiff {
	if store == nil {
		return diff
	}
	for i, kw := range diff.AddKeywords {
		doc, err := store.GetStats(ctx, kw.ID)
		if err != nil {
			continue
		}
		currentWeight := kw.Weight
		if currentWeight <= 0 {
			currentWeight = 20
		}
		suggested, enabled := doc.Recommendation(currentWeight)
		diff.AddKeywords[i].EvidenceCount = doc.TotalSamples()
		diff.AddKeywords[i].JunkRate = doc.JunkRate()
		diff.AddKeywords[i].SuggestedWeight = suggested
		diff.AddKeywords[i].Enabled = &enabled
	}
	return diff
}
