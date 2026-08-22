package entity

import (
	"fmt"
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
	if patch.HeatTierDowngrade != HeatTierWarm {
		t.Fatalf("patch downgrade=%q want warm", patch.HeatTierDowngrade)
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

func TestEntityClassificationPatchOmitsHeatUnlessDowngrade(t *testing.T) {
	doc := EntityDoc{HeatTier: HeatTierBlazing, HeatScore: 99}
	patch := ApplyEntityClassification(&doc, GeminiClassifyResult{
		SameActor:       true,
		ActorConfidence: 0.95,
		UnifiedPain:     "Voluum postback pain",
		BuyerIntent:     "hot",
	}, time.Now().UTC())
	if patch.HeatTierDowngrade != "" {
		t.Fatalf("downgrade=%q want empty", patch.HeatTierDowngrade)
	}
	if doc.HeatScore != 99 {
		t.Fatalf("heat_score=%v want unchanged 99", doc.HeatScore)
	}
}

func TestClassifySightingsFromDocNewestFirst(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	doc := EntityDoc{
		Sightings: []EntitySighting{
			{HashID: "old-1", PostedAt: base},
			{HashID: "old-2", PostedAt: base.Add(24 * time.Hour)},
			{HashID: "old-3", PostedAt: base.Add(48 * time.Hour)},
			{HashID: "old-4", PostedAt: base.Add(72 * time.Hour)},
			{HashID: "old-5", PostedAt: base.Add(96 * time.Hour)},
			{HashID: "newest", PostedAt: base.Add(120 * time.Hour)},
		},
	}
	got := ClassifySightingsFromDoc(doc)
	if len(got) != 5 {
		t.Fatalf("len=%d want 5", len(got))
	}
	if got[0].HashID != "newest" {
		t.Fatalf("first=%q want newest", got[0].HashID)
	}
	if got[4].HashID != "old-2" {
		t.Fatalf("fifth=%q want old-2 (oldest of selected five)", got[4].HashID)
	}
}

func TestClassifySightingsFromDocLimitTwenty(t *testing.T) {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	var sightings []EntitySighting
	for i := 0; i < 25; i++ {
		sightings = append(sightings, EntitySighting{
			HashID:   fmt.Sprintf("h-%02d", i),
			PostedAt: base.Add(time.Duration(i) * time.Hour),
		})
	}
	got := ClassifySightingsFromDocLimit(EntityDoc{Sightings: sightings}, MaxLowConfidenceClassifySightings)
	if len(got) != MaxLowConfidenceClassifySightings {
		t.Fatalf("len=%d want %d", len(got), MaxLowConfidenceClassifySightings)
	}
	if got[0].HashID != "h-24" {
		t.Fatalf("first=%q want h-24", got[0].HashID)
	}
}

func TestIsLowConfidenceHot(t *testing.T) {
	if !IsLowConfidenceHot(EntityDoc{HeatTier: HeatTierHot, ActorConfidence: 0.2}) {
		t.Fatal("expected low confidence hot")
	}
	if IsLowConfidenceHot(EntityDoc{HeatTier: HeatTierWarm, ActorConfidence: 0.1}) {
		t.Fatal("warm tier should not qualify")
	}
	if !IsLowConfidenceHot(EntityDoc{HeatTier: HeatTierBlazing}) {
		t.Fatal("unclassified blazing should qualify")
	}
}
