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
// Hot when >=2 source families, >=1 pain hit, and (LastSeen-FirstSeen) fits the window (span, not recency).
func CrossSourceHot(doc EntityDoc, window time.Duration) bool {
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

// EnrichRecordResult adds cross-source hot fields from the entity document.
func EnrichRecordResult(result RecordResult, doc EntityDoc, window time.Duration) RecordResult {
	result.CanonicalHash = doc.CanonicalHash
	result.CrossSourceHot = CrossSourceHot(doc, window)
	return result
}
