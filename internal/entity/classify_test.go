package entity

import (
	"testing"
	"time"
)

func TestShouldTriggerEntityClassify(t *testing.T) {
	if ShouldTriggerEntityClassify(RecordResult{}) {
		t.Fatal("empty result")
	}
	if !ShouldTriggerEntityClassify(RecordResult{EntityID: "e1", SightingCount: 2}) {
		t.Fatal("expected trigger at 2 sightings")
	}
	if !ShouldTriggerEntityClassify(RecordResult{EntityID: "e1", NewSourceFamily: true}) {
		t.Fatal("expected trigger on new family")
	}
	if !ShouldTriggerEntityClassify(RecordResult{EntityID: "e1", HeatTier: HeatTierHot}) {
		t.Fatal("expected trigger on hot tier")
	}
}

func TestApplyEntityClassificationDowngradesSplit(t *testing.T) {
	doc := EntityDoc{
		EntityID:  "e1",
		HeatTier:  HeatTierHot,
		HeatScore: 65,
	}
	patch := ApplyEntityClassification(&doc, GeminiClassifyResult{
		SameActor:        false,
		ActorConfidence:  0.85,
		UnifiedPain:      "",
		BuyerIntent:      "cold",
		SplitRecommended: true,
		Why:              "recruiting vs buyer",
	}, time.Now().UTC())

	if !patch.NeedsReview {
		t.Fatal("expected needs_review")
	}
	if doc.HeatTier != HeatTierWarm {
		t.Fatalf("heat_tier=%q want warm", doc.HeatTier)
	}
	if doc.BuyerIntent != "cold" {
		t.Fatalf("buyer_intent=%q", doc.BuyerIntent)
	}
}

func TestApplyEntityClassificationKeepsHotWhenSameActor(t *testing.T) {
	doc := EntityDoc{HeatTier: HeatTierBlazing, HeatScore: 90}
	ApplyEntityClassification(&doc, GeminiClassifyResult{
		SameActor:       true,
		ActorConfidence: 0.9,
		UnifiedPain:     "Voluum postback pain",
		BuyerIntent:     "hot",
	}, time.Now().UTC())
	if doc.HeatTier != HeatTierBlazing {
		t.Fatalf("heat_tier=%q", doc.HeatTier)
	}
	if doc.UnifiedPain == "" {
		t.Fatal("expected unified_pain")
	}
}
