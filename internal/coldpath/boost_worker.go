package coldpath

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
)

// BoostLeadLookup finds existing leads for boost merge outcomes.
type BoostLeadLookup interface {
	FindHashIDByContactValue(ctx context.Context, value string) (string, error)
}

// BoostStore reads and resolves crm_boosts rows.
type BoostStore interface {
	Insert(ctx context.Context, doc sink.CrmBoostDoc) error
	ListPending(ctx context.Context, limit int) ([]sink.CrmBoostDoc, error)
	Resolve(ctx context.Context, junkID, status, leadHashID, why string) error
}

func (s *Service) runBoostWorker(ctx context.Context) {
	if s == nil || s.crm == nil || s.registry == nil {
		return
	}
	docs, err := s.crm.ListPending(ctx, 20)
	if err != nil {
		slog.Warn("crm boost list pending failed", "error", err)
		return
	}
	if len(docs) == 0 {
		return
	}
	for _, doc := range docs {
		status, hashID, why := s.classifyBoost(ctx, doc)
		if err := s.crm.Resolve(ctx, doc.JunkID, status, hashID, why); err != nil {
			slog.Warn("crm boost resolve failed", "junk_id", doc.JunkID, "error", err)
			continue
		}
		slog.Info("crm boost resolved",
			"junk_id", doc.JunkID,
			"status", status,
			"lead_hash_id", hashID,
		)
	}
}

func (s *Service) classifyBoost(ctx context.Context, doc sink.CrmBoostDoc) (status, leadHashID, why string) {
	text := doc.Snippet
	if hit, ok := s.registry.HardReject(text); ok {
		return sink.CrmBoostDismissed, "", "hard_reject:" + hit.Phrase
	}
	result := scoring.AnalyzeWithRegistry(s.registry, text)
	_, _, _, _, mediumMin := s.registry.Snapshot()
	if result.Score < mediumMin {
		return sink.CrmBoostDismissed, "", fmt.Sprintf("score=%d below medium_min=%d", result.Score, mediumMin)
	}
	if s.leads != nil && doc.ContactHint != "" {
		if hash, err := s.leads.FindHashIDByContactValue(ctx, doc.ContactHint); err == nil && hash != "" {
			return sink.CrmBoostMerged, hash, "existing lead contact"
		}
	}
	return sink.CrmBoostPromoted, "", fmt.Sprintf("score=%d", result.Score)
}
