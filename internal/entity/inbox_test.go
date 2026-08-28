package entity

import "testing"

func TestBuildInboxCard(t *testing.T) {
	t.Parallel()

	doc := EntityDoc{
		EntityID:       "ent-1",
		HeatTier:       HeatTierHot,
		HeatScoreRound: 88,
		SightingCount:  5,
		SourceCount:    3,
		SourceFamilies: []string{"forum", "telegram"},
		UnifiedPain:    "Voluum postback pain",
		BuyerIntent:    "buying_tracker",
		NeedsReview:    true,
		ReviewSuggestions: []ReviewSuggestion{
			{Status: "pending"},
			{Status: "done"},
		},
	}
	card := BuildInboxCard(doc)
	if card.EntityID != "ent-1" {
		t.Fatalf("id=%q", card.EntityID)
	}
	if card.HeatScore != 88 {
		t.Fatalf("heat=%d", card.HeatScore)
	}
	if card.PendingSuggestions != 1 {
		t.Fatalf("pending=%d", card.PendingSuggestions)
	}
	if card.EntityProof == "" {
		t.Fatal("expected entity proof summary")
	}
}
