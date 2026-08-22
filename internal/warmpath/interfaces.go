package warmpath

import (
	"context"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

// EmbedPrescanner runs embed-only spam/pain checks (no generateContent).
type EmbedPrescanner interface {
	EvaluateSpam(ctx context.Context, text string) (gemini.PrescanVerdict, error)
}

// LeadClusterer detects semantic duplicate accepted leads.
type LeadClusterer interface {
	CheckDuplicate(ctx context.Context, hashID, text string) (bool, string, error)
	Record(ctx context.Context, hashID, text string) error
}

// MediumEngager returns lite outreach for Medium leads with warm entity heat.
type MediumEngager interface {
	ClassifyEngagement(ctx context.Context, in gemini.EngagementInput) (gemini.EngagementResult, error)
}

// WarmJunkInserter persists warm-path rejects into junk_leads.
type WarmJunkInserter interface {
	InsertMany(ctx context.Context, docs []sink.JunkDoc) error
}
