package sink

import (
	"testing"
)

func TestKeywordRecommendationHighJunkRate(t *testing.T) {
	doc := KeywordStatDoc{
		KeywordID:     "kw-test",
		AcceptedCount: 10,
		JunkCount:     10, // 50% junk rate on 20 samples
	}

	if rate := doc.JunkRate(); rate != 0.5 {
		t.Errorf("got junk rate %f, want 0.5", rate)
	}

	_, enabled := doc.Recommendation(20)
	if enabled {
		t.Errorf("expected enabled: false for high junk rate (>30%% on 20+ samples)")
	}
}

func TestKeywordRecommendationLowJunkRate(t *testing.T) {
	doc := KeywordStatDoc{
		KeywordID:     "kw-test",
		AcceptedCount: 20,
		JunkCount:     1, // ~4.7% junk rate
	}

	sWeight, enabled := doc.Recommendation(20)
	if !enabled {
		t.Errorf("expected enabled: true")
	}
	if sWeight <= 20 {
		t.Errorf("expected weight boost for low junk rate, got %d", sWeight)
	}
}
