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
