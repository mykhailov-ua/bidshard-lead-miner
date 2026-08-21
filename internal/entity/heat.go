package entity

import (
	"strings"
	"time"
)

const (
	HeatTierBlazing = "blazing"
	HeatTierHot     = "hot"
	HeatTierWarm    = "warm"
	HeatTierCold    = "cold"

	defaultFreshWindow     = 14 * 24 * time.Hour
	defaultCanonicalWindow = 14 * 24 * time.Hour
	defaultDecayBeyond     = 0.1
	defaultUnknownDecayCap = 0.5
)

// HeatConfig controls local entity heat scoring (HEAT-P2).
type HeatConfig struct {
	Enabled          bool
	BlazingThreshold float64
	HotThreshold     float64
	WarmThreshold    float64
	Decay7D          float64
	Decay30D         float64
	Decay90D         float64
	DecayBeyond      float64
	UnknownDateDecay float64
	FreshWindow      time.Duration
	CanonicalWindow  time.Duration
	HotBoost         int
	BlazingBoost     int
	WarmBoost        int
}

// DefaultHeatConfig returns default heat thresholds and decay table.
func DefaultHeatConfig() HeatConfig {
	return HeatConfig{
		Enabled:          true,
		BlazingThreshold: 80,
		HotThreshold:     50,
		WarmThreshold:    25,
		Decay7D:          1.0,
		Decay30D:         0.6,
		Decay90D:         0.25,
		DecayBeyond:      defaultDecayBeyond,
		UnknownDateDecay: defaultUnknownDecayCap,
		FreshWindow:      defaultFreshWindow,
		CanonicalWindow:  defaultCanonicalWindow,
		HotBoost:         DefaultCrossSourceBoost,
		BlazingBoost:     30,
		WarmBoost:        5,
	}
}

// HeatTierRank orders tiers for comparisons (higher = hotter).
func HeatTierRank(tier string) int {
	switch tier {
	case HeatTierBlazing:
		return 4
	case HeatTierHot:
		return 3
	case HeatTierWarm:
		return 2
	case HeatTierCold:
		return 1
	default:
		return 0
	}
}

// RecomputeEntityHeat updates heat fields and may promote canonical_hash_id by sighting weight.
func RecomputeEntityHeat(doc *EntityDoc, cfg HeatConfig, now time.Time) {
	if doc == nil || !cfg.Enabled {
		return
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	cfg = normalizeHeatConfig(cfg)

	doc.FreshSightingAt = computeFreshSightingAt(doc.Sightings, now, cfg.FreshWindow)
	sum := 0.0
	for _, s := range doc.Sightings {
		sum += sightingWeight(s, cfg, now)
	}
	doc.HeatScore = sum * diversityBonus(doc.Sightings, cfg, now)
	doc.HeatTier = classifyHeatTier(doc.HeatScore, *doc, cfg, now)
	doc.LastHeatAt = now

	promoteCanonicalByHeat(doc, cfg, now)
}

func normalizeHeatConfig(cfg HeatConfig) HeatConfig {
	def := DefaultHeatConfig()
	if cfg.BlazingThreshold <= 0 {
		cfg.BlazingThreshold = def.BlazingThreshold
	}
	if cfg.HotThreshold <= 0 {
		cfg.HotThreshold = def.HotThreshold
	}
	if cfg.WarmThreshold <= 0 {
		cfg.WarmThreshold = def.WarmThreshold
	}
	if cfg.Decay7D <= 0 {
		cfg.Decay7D = def.Decay7D
	}
	if cfg.Decay30D <= 0 {
		cfg.Decay30D = def.Decay30D
	}
	if cfg.Decay90D <= 0 {
		cfg.Decay90D = def.Decay90D
	}
	if cfg.DecayBeyond <= 0 {
		cfg.DecayBeyond = def.DecayBeyond
	}
	if cfg.UnknownDateDecay <= 0 {
		cfg.UnknownDateDecay = def.UnknownDateDecay
	}
	if cfg.FreshWindow <= 0 {
		cfg.FreshWindow = def.FreshWindow
	}
	if cfg.CanonicalWindow <= 0 {
		cfg.CanonicalWindow = def.CanonicalWindow
	}
	if cfg.HotBoost <= 0 {
		cfg.HotBoost = def.HotBoost
	}
	if cfg.BlazingBoost <= 0 {
		cfg.BlazingBoost = def.BlazingBoost
	}
	if cfg.WarmBoost <= 0 {
		cfg.WarmBoost = def.WarmBoost
	}
	return cfg
}

func sightingWeight(s EntitySighting, cfg HeatConfig, now time.Time) float64 {
	pw := painWeight(s)
	if pw <= 0 {
		return 0
	}
	age, unknown := sightingAge(s, now)
	return pw * recencyDecay(age, unknown, cfg)
}

func painWeight(s EntitySighting) float64 {
	base := float64(s.Score)
	if base <= 0 && len(s.Matched) > 0 {
		base = 30
	}
	if base <= 0 {
		return 0
	}
	if base > 50 {
		base = 50
	}
	if len(s.Matched) > 0 {
		base *= 1.2
		if base > 50 {
			base = 50
		}
	}
	return base
}

func sightingAge(s EntitySighting, now time.Time) (time.Duration, bool) {
	if !s.PostedAt.IsZero() {
		age := now.Sub(s.PostedAt.UTC())
		if age < 0 {
			age = 0
		}
		return age, false
	}
	if !s.SeenAt.IsZero() {
		age := now.Sub(s.SeenAt.UTC())
		if age < 0 {
			age = 0
		}
		return age, true
	}
	return 0, true
}

func recencyDecay(age time.Duration, unknownDate bool, cfg HeatConfig) float64 {
	var d float64
	switch {
	case age <= 7*24*time.Hour:
		d = cfg.Decay7D
	case age <= 30*24*time.Hour:
		d = cfg.Decay30D
	case age <= 90*24*time.Hour:
		d = cfg.Decay90D
	default:
		d = cfg.DecayBeyond
	}
	if unknownDate && d > cfg.UnknownDateDecay {
		d = cfg.UnknownDateDecay
	}
	return d
}

func diversityBonus(sightings []EntitySighting, cfg HeatConfig, now time.Time) float64 {
	cutoff := now.Add(-30 * 24 * time.Hour)
	families := make(map[string]struct{})
	for _, s := range sightings {
		if !sightingHasPain(s) {
			continue
		}
		event := sightingEventTimeOf(s)
		if event.IsZero() || event.Before(cutoff) {
			continue
		}
		family := s.Family
		if family == "" {
			family = SourceFamily(s.Source)
		}
		if family == "" {
			continue
		}
		families[family] = struct{}{}
	}
	switch len(families) {
	case 0, 1:
		return 1.0
	case 2:
		return 1.3
	default:
		return 1.5
	}
}

func sightingHasPain(s EntitySighting) bool {
	return len(s.Matched) > 0 || s.Score >= 15
}

func computeFreshSightingAt(sightings []EntitySighting, now time.Time, window time.Duration) time.Time {
	if window <= 0 {
		window = defaultFreshWindow
	}
	cutoff := now.Add(-window)
	var fresh time.Time
	for _, s := range sightings {
		event := sightingEventTimeOf(s)
		if event.IsZero() || event.Before(cutoff) {
			continue
		}
		if event.After(fresh) {
			fresh = event
		}
	}
	return fresh
}

func classifyHeatTier(score float64, doc EntityDoc, cfg HeatConfig, now time.Time) string {
	_ = now
	fresh := !doc.FreshSightingAt.IsZero()

	if score >= cfg.BlazingThreshold && fresh && doc.SourceCount >= 2 && doc.PainHits >= 1 {
		return HeatTierBlazing
	}
	if score >= cfg.HotThreshold && fresh && doc.SourceCount >= 2 && doc.PainHits >= 1 {
		return HeatTierHot
	}
	if score >= cfg.WarmThreshold {
		return HeatTierWarm
	}
	return HeatTierCold
}

func promoteCanonicalByHeat(doc *EntityDoc, cfg HeatConfig, now time.Time) {
	if doc == nil {
		return
	}
	cutoff := now.Add(-cfg.CanonicalWindow)
	bestHash := ""
	bestWeight := -1.0
	for _, s := range doc.Sightings {
		event := sightingEventTimeOf(s)
		if event.IsZero() || event.Before(cutoff) {
			continue
		}
		w := sightingWeight(s, cfg, now)
		if w > bestWeight {
			bestWeight = w
			bestHash = s.HashID
		}
	}
	if bestHash != "" {
		doc.CanonicalHash = bestHash
	}
}

func HeatBoostForTier(tier string, cfg HeatConfig) int {
	return heatBoostForTier(tier, cfg)
}

func HeatTierMeetsMin(tier, minTier string) bool {
	minTier = strings.ToLower(strings.TrimSpace(minTier))
	if minTier == "" || minTier == HeatTierCold {
		return true
	}
	tier = strings.ToLower(strings.TrimSpace(tier))
	if tier == "" {
		tier = HeatTierCold
	}
	return HeatTierRank(tier) >= HeatTierRank(minTier)
}

// HeatTiersAtOrAbove returns tier names at or above minTier (for Mongo $in filters).
func HeatTiersAtOrAbove(minTier string) []string {
	minRank := HeatTierRank(strings.ToLower(strings.TrimSpace(minTier)))
	if minRank <= HeatTierRank(HeatTierCold) {
		return nil
	}
	var out []string
	for _, tier := range []string{HeatTierCold, HeatTierWarm, HeatTierHot, HeatTierBlazing} {
		if HeatTierRank(tier) >= minRank {
			out = append(out, tier)
		}
	}
	return out
}

func heatBoostForTier(tier string, cfg HeatConfig) int {
	switch tier {
	case HeatTierBlazing:
		return cfg.BlazingBoost
	case HeatTierHot:
		return cfg.HotBoost
	case HeatTierWarm:
		return cfg.WarmBoost
	default:
		return 0
	}
}
