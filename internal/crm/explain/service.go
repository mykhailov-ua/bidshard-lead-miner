package explain

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

const cacheTTL = 24 * time.Hour

// Service generates read-only lead explain strings for CRM inbox.
type Service struct {
	store    *store.LeadStore
	gemini   *gemini.Client
	cacheTTL time.Duration
}

func New(store *store.LeadStore, client *gemini.Client) *Service {
	return &Service{store: store, gemini: client, cacheTTL: cacheTTL}
}

func (s *Service) Explain(ctx context.Context, hashID string) (string, error) {
	if s == nil || s.store == nil {
		return "", fmt.Errorf("explain service not initialized")
	}
	hashID = strings.TrimSpace(hashID)
	if hashID == "" {
		return "", fmt.Errorf("hash_id empty")
	}
	if cached, ok, err := s.store.GetExplainCache(ctx, hashID); err != nil {
		slog.Warn("explain cache read failed", "hash_id", hashID, "error", err)
	} else if ok {
		return cached, nil
	}
	lead, err := s.store.GetByHashID(ctx, hashID)
	if err != nil {
		return "", err
	}
	summary := s.ruleSummary(lead)
	if s.gemini != nil {
		if g, err := s.gemini.ExplainLead(ctx, gemini.LeadExplainInput{
			HashID:        lead.HashID,
			Priority:      lead.Priority,
			Score:         lead.Score,
			Source:        lead.Source,
			Matched:       lead.Matched,
			Snippet:       lead.Snippet,
			HeatTier:      lead.HeatTier,
			EntityProof:   lead.EntityProof,
			OutreachAngle: lead.OutreachAngle,
			ICP:           lead.ICP,
		}); err != nil {
			slog.Warn("explain gemini failed, using rule summary", "hash_id", hashID, "error", err)
		} else if strings.TrimSpace(g) != "" {
			summary = strings.TrimSpace(g)
		}
	}
	if err := s.store.SetExplainCache(ctx, hashID, summary, s.cacheTTL); err != nil {
		slog.Warn("explain cache write failed", "hash_id", hashID, "error", err)
	}
	return summary, nil
}

func (s *Service) ruleSummary(lead sink.LeadDoc) string {
	parts := []string{lead.Priority + " lead"}
	if lead.Score > 0 {
		parts = append(parts, fmt.Sprintf("score %d", lead.Score))
	}
	if len(lead.Matched) > 0 {
		parts = append(parts, "matched "+strings.Join(lead.Matched[:min(3, len(lead.Matched))], ", "))
	}
	if lead.HeatTier != "" {
		parts = append(parts, "entity "+lead.HeatTier)
	}
	return strings.Join(parts, "; ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
