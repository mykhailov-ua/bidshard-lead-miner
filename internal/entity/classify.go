package entity

import (
	"sort"
	"time"
)

const splitActorConfidenceMin = 0.7

// EntityClassificationPatch updates Gemini entity validation fields on Mongo.
type EntityClassificationPatch struct {
	UnifiedPain        string
	ActorConfidence    float64
	BuyerIntent        string
	NeedsReview        bool
	EntityClassifiedAt time.Time
	// HeatTierDowngrade is set only when classify downgrades hot/blazing to warm.
	// Empty means the patch must not touch heat_tier or heat_score in Mongo.
	HeatTierDowngrade string
	SemanticCluster   string
}

// ShouldTriggerEntityClassify reports whether warm-path Gemini entity validation should run.
func ShouldTriggerEntityClassify(result RecordResult) bool {
	if result.EntityID == "" {
		return false
	}
	if result.NewSourceFamily {
		return true
	}
	if result.SightingCount >= 2 {
		return true
	}
	return HeatTierRank(result.HeatTier) >= HeatTierRank(HeatTierHot)
}

// ApplyEntityClassification merges Gemini output onto an entity document.
func ApplyEntityClassification(doc *EntityDoc, res GeminiClassifyResult, now time.Time) EntityClassificationPatch {
	if doc == nil {
		return EntityClassificationPatch{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	patch := EntityClassificationPatch{
		UnifiedPain:        res.UnifiedPain,
		ActorConfidence:    res.ActorConfidence,
		BuyerIntent:        res.BuyerIntent,
		NeedsReview:        res.SplitRecommended,
		EntityClassifiedAt: now,
	}

	if !res.SameActor && res.ActorConfidence >= splitActorConfidenceMin {
		patch.NeedsReview = true
		if HeatTierRank(doc.HeatTier) >= HeatTierRank(HeatTierHot) {
			patch.HeatTierDowngrade = HeatTierWarm
		}
	}
	if res.SplitRecommended {
		patch.NeedsReview = true
	}

	doc.UnifiedPain = patch.UnifiedPain
	doc.ActorConfidence = patch.ActorConfidence
	doc.BuyerIntent = patch.BuyerIntent
	doc.NeedsReview = patch.NeedsReview
	doc.EntityClassifiedAt = patch.EntityClassifiedAt
	if patch.HeatTierDowngrade != "" {
		doc.HeatTier = patch.HeatTierDowngrade
	}
	patch.SemanticCluster = SemanticClusterID(doc.Matched, patch.UnifiedPain)
	doc.SemanticCluster = patch.SemanticCluster
	return patch
}

// GeminiClassifyResult is the entity-local view of Gemini classification output.
type GeminiClassifyResult struct {
	SameActor        bool
	ActorConfidence  float64
	UnifiedPain      string
	BuyerIntent      string
	SplitRecommended bool
	Why              string
}

// ClassifySightingsFromDoc returns up to 5 sightings for Gemini input (newest first).
func ClassifySightingsFromDoc(doc EntityDoc) []EntitySighting {
	return ClassifySightingsFromDocLimit(doc, MaxEntityClassifySightings)
}

// ClassifySightingsFromDocLimit returns up to limit sightings (newest first).
func ClassifySightingsFromDocLimit(doc EntityDoc, limit int) []EntitySighting {
	if len(doc.Sightings) == 0 {
		return nil
	}
	if limit <= 0 {
		limit = MaxEntityClassifySightings
	}
	out := append([]EntitySighting(nil), doc.Sightings...)
	sort.Slice(out, func(i, j int) bool {
		ti := sightingEventTimeOf(out[i])
		tj := sightingEventTimeOf(out[j])
		if ti.Equal(tj) {
			// Stable tie-break when PostedAt missing on both sightings.
			return out[i].HashID > out[j].HashID
		}
		return ti.After(tj)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

const MaxEntityClassifySightings = 5

const MaxLowConfidenceClassifySightings = 20

const lowConfidenceActorThreshold = 0.5

// IsLowConfidenceHot reports hot/blazing entities that need more sightings for classify.
func IsLowConfidenceHot(doc EntityDoc) bool {
	if HeatTierRank(doc.HeatTier) < HeatTierRank(HeatTierHot) {
		return false
	}
	if doc.EntityClassifiedAt.IsZero() {
		return true
	}
	return doc.ActorConfidence < lowConfidenceActorThreshold
}
