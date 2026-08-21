package entity

import (
	"time"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
)

const (
	TagCrossSourceHot         = "cross-source-hot"
	DefaultCrossSourceWindow  = 7 * 24 * time.Hour
	DefaultCrossSourceBoost   = 20
	MinCrossSourceSourceCount = 2
)

// CrossSourceHot reports whether an entity accumulated enough cross-source pain signals.
// When heat_tier is set, this is an alias for heat_tier >= hot (HEAT-P2 migration).
func CrossSourceHot(doc EntityDoc, window time.Duration) bool {
	if doc.HeatTier != "" {
		return HeatTierRank(doc.HeatTier) >= HeatTierRank(HeatTierHot)
	}
	if doc.SourceCount < MinCrossSourceSourceCount {
		return false
	}
	if doc.PainHits < 1 {
		return false
	}
	if doc.FirstSeen.IsZero() || doc.LastSeen.IsZero() {
		return false
	}
	if window <= 0 {
		window = DefaultCrossSourceWindow
	}
	return doc.LastSeen.Sub(doc.FirstSeen) <= window
}

// ApplyCrossSourceHotBoost updates an accepted lead with cross-source hot signals.
func ApplyCrossSourceHotBoost(lead *model.Lead, reg *scoring.Registry, boost int) {
	if lead == nil {
		return
	}
	if boost <= 0 {
		boost = DefaultCrossSourceBoost
	}
	lead.Score += boost
	lead.Hot = true
	lead.Tags = AppendUniqueTag(lead.Tags, TagCrossSourceHot)

	priority := scoring.PriorityMedium
	if lead.Priority != "" {
		priority = scoring.Priority(lead.Priority)
	}
	if reg != nil {
		priority = scoring.PriorityFromScore(reg, lead.Score)
	}
	if priority == scoring.PriorityMedium {
		priority = scoring.PriorityHigh
	}
	lead.Priority = string(priority)
}

// ApplyEntityHeatToLead patches lead fields from entity heat and applies tier boosts.
func ApplyEntityHeatToLead(lead *model.Lead, result RecordResult, reg *scoring.Registry, cfg HeatConfig) {
	if lead == nil {
		return
	}
	lead.EntityHeat = result.HeatScore
	lead.HeatTier = result.HeatTier
	if result.HeatTier == "" {
		return
	}
	if HeatTierRank(result.HeatTier) >= HeatTierRank(HeatTierHot) {
		ApplyCrossSourceHotBoost(lead, reg, heatBoostForTier(result.HeatTier, cfg))
	}
}

// EnrichRecordResult adds cross-source hot fields from the entity document.
func EnrichRecordResult(result RecordResult, doc EntityDoc, window time.Duration, heat HeatConfig) RecordResult {
	result.CanonicalHash = doc.CanonicalHash
	result.HeatScore = doc.HeatScore
	result.HeatTier = doc.HeatTier
	result.CrossSourceHot = CrossSourceHot(doc, window)
	return result
}
