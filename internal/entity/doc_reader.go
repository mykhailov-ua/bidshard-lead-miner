package entity

import "context"

// DocReader loads entity documents for warm-path classification.
type DocReader interface {
	GetEntity(ctx context.Context, entityID string) (EntityDoc, bool, error)
}

// ClassificationPatcher persists Gemini entity validation results.
type ClassificationPatcher interface {
	PatchEntityClassification(ctx context.Context, entityID string, patch EntityClassificationPatch) error
}

// ClassifyForceLister returns entity ids flagged for ops/parser re-classify.
type ClassifyForceLister interface {
	ListClassifyForce(ctx context.Context, limit int) ([]string, error)
	ClearClassifyForce(ctx context.Context, entityID string) error
}

// LinkedLeadHeatSync updates stored leads when entity heat tier changes on classify.
type LinkedLeadHeatSync struct {
	HeatScore     float64
	HeatTier      string
	SightingCount int
	SourceCount   int
}

// LinkedLeadHeatPatcher syncs entity heat fields onto linked lead rows.
type LinkedLeadHeatPatcher interface {
	PatchLinkedLeadsHeat(ctx context.Context, entityID string, hashIDs []string, sync LinkedLeadHeatSync) error
}

// LinkedLeadOutreachSync patches GTM narrative fields on leads linked to an entity.
type LinkedLeadOutreachSync struct {
	OutreachAngle string
	EntityProof   string
}

// LinkedLeadOutreachPatcher writes entity-level outreach narrative onto linked leads.
type LinkedLeadOutreachPatcher interface {
	PatchLinkedLeadsOutreach(ctx context.Context, entityID string, hashIDs []string, sync LinkedLeadOutreachSync) error
}

// SightingContactsReader loads masked contact summaries for entity classify prompts.
type SightingContactsReader interface {
	MaskedContactsSummary(ctx context.Context, hashID string) string
}

// PainSampleLister returns recent unified_pain strings from entity graph.
type PainSampleLister interface {
	ListUnifiedPainSamples(ctx context.Context, limit int) ([]string, error)
}

// LowConfidenceHotLister returns hot/blazing entity ids with low actor confidence.
type LowConfidenceHotLister interface {
	ListLowConfidenceHotEntities(ctx context.Context, limit int) ([]string, error)
}
