package entity

import (
	"testing"
	"time"
)

func TestComputeEntityHeatStalePlusFreshCrossSource(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	cfg := DefaultHeatConfig()

	doc := EntityDoc{
		SourceCount: 2,
		PainHits:    2,
		Sightings: []EntitySighting{
			{
				HashID:   "hash-old",
				Source:   "forum:affiliatefix.com/thread-a",
				Family:   "forum",
				PostedAt: now.Add(-35 * 24 * time.Hour),
				Matched:  []string{"voluum"},
				Score:    80,
			},
			{
				HashID:   "hash-fresh",
				Source:   "reddit:igaming",
				Family:   "reddit",
				PostedAt: now.Add(-5 * 24 * time.Hour),
				Matched:  []string{"postback"},
				Score:    80,
			},
		},
	}
	RecomputeEntityHeat(&doc, cfg, now)

	if doc.HeatTier != HeatTierHot && doc.HeatTier != HeatTierBlazing {
		t.Fatalf("tier=%q score=%.2f want hot or blazing", doc.HeatTier, doc.HeatScore)
	}
	if doc.HeatScore < cfg.HotThreshold {
		t.Fatalf("heat_score=%.2f want >= %.0f", doc.HeatScore, cfg.HotThreshold)
	}
	if doc.CanonicalHash != "hash-fresh" {
		t.Fatalf("canonical=%q want hash-fresh", doc.CanonicalHash)
	}
}

func TestComputeEntityHeatSingleStaleSightingCold(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	cfg := DefaultHeatConfig()

	doc := EntityDoc{
		SourceCount: 1,
		PainHits:    1,
		Sightings: []EntitySighting{
			{
				HashID:   "hash-old",
				Source:   "forum:affiliatefix.com/thread-a",
				Family:   "forum",
				PostedAt: now.Add(-60 * 24 * time.Hour),
				Matched:  []string{"voluum"},
				Score:    80,
			},
		},
	}
	RecomputeEntityHeat(&doc, cfg, now)

	if HeatTierRank(doc.HeatTier) >= HeatTierRank(HeatTierHot) {
		t.Fatalf("tier=%q score=%.2f want below hot", doc.HeatTier, doc.HeatScore)
	}
	if doc.FreshSightingAt != (time.Time{}) {
		t.Fatalf("fresh_sighting_at=%v want zero", doc.FreshSightingAt)
	}
}

func TestCrossSourceHotUsesHeatTierWhenPresent(t *testing.T) {
	doc := EntityDoc{
		HeatTier:  HeatTierHot,
		HeatScore: 55,
		FirstSeen: time.Now().UTC().Add(-30 * 24 * time.Hour),
		LastSeen:  time.Now().UTC(),
	}
	if !CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected hot via heat tier despite wide span")
	}

	doc.HeatTier = HeatTierCold
	if CrossSourceHot(doc, DefaultCrossSourceWindow) {
		t.Fatal("expected false for cold tier")
	}
}

func TestHeatTierMeetsMin(t *testing.T) {
	if !HeatTierMeetsMin(HeatTierHot, HeatTierWarm) {
		t.Fatal("hot should meet warm min")
	}
	if HeatTierMeetsMin(HeatTierWarm, HeatTierHot) {
		t.Fatal("warm should not meet hot min")
	}
	if !HeatTierMeetsMin("", HeatTierCold) {
		t.Fatal("empty tier should pass cold min")
	}
	if len(HeatTiersAtOrAbove(HeatTierHot)) < 2 {
		t.Fatalf("tiers=%v", HeatTiersAtOrAbove(HeatTierHot))
	}
}

func TestDiversityBonusTwoFamilies(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	sightings := []EntitySighting{
		{Family: "forum", PostedAt: now.Add(-2 * 24 * time.Hour), Matched: []string{"voluum"}, Score: 40},
		{Family: "reddit", PostedAt: now.Add(-1 * 24 * time.Hour), Matched: []string{"postback"}, Score: 40},
	}
	if got := diversityBonus(sightings, DefaultHeatConfig(), now); got != 1.3 {
		t.Fatalf("bonus=%v want 1.3", got)
	}
}
