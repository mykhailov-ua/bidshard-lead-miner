package entity

import "time"

const splitActorConfidenceMin = 0.7

// EntityClassificationPatch updates Gemini entity validation fields on Mongo.
type EntityClassificationPatch struct {
	UnifiedPain        string
	ActorConfidence    float64
	BuyerIntent        string
	NeedsReview        bool
	HeatTier           string
	HeatScore          float64
	EntityClassifiedAt time.Time
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
		HeatTier:           doc.HeatTier,
		HeatScore:          doc.HeatScore,
		EntityClassifiedAt: now,
	}

	if !res.SameActor && res.ActorConfidence >= splitActorConfidenceMin {
		patch.NeedsReview = true
		if HeatTierRank(doc.HeatTier) >= HeatTierRank(HeatTierHot) {
			patch.HeatTier = HeatTierWarm
		}
	}
	if res.SplitRecommended {
		patch.NeedsReview = true
	}

	doc.UnifiedPain = patch.UnifiedPain
	doc.ActorConfidence = patch.ActorConfidence
	doc.BuyerIntent = patch.BuyerIntent
	doc.NeedsReview = patch.NeedsReview
	doc.HeatTier = patch.HeatTier
	doc.HeatScore = patch.HeatScore
	doc.EntityClassifiedAt = patch.EntityClassifiedAt
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
	if len(doc.Sightings) == 0 {
		return nil
	}
	out := append([]EntitySighting(nil), doc.Sightings...)
	if len(out) > maxEntityClassifySightings {
		out = out[:maxEntityClassifySightings]
	}
	return out
}

const maxEntityClassifySightings = 5
