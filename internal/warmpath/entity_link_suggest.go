package warmpath

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/worker"
)

// EntityLinkCandidateReader loads entities for domain-pair review.
type EntityLinkCandidateReader interface {
	ListWarmEntitiesForLinkSuggest(ctx context.Context, limit int) ([]entity.EntityDoc, error)
}

// EntityLinkClassifier runs Gemini merge/split/keep on entity pairs.
type EntityLinkClassifier interface {
	ClassifyEntityLink(ctx context.Context, in gemini.EntityLinkInput) (gemini.EntityLinkResult, error)
}

// EntityReviewSuggestionWriter persists pending graph review rows.
type EntityReviewSuggestionWriter interface {
	AppendReviewSuggestion(ctx context.Context, entityID string, suggestion entity.ReviewSuggestion) error
}

// EntityLinkSuggestConfig controls periodic entity link review.
type EntityLinkSuggestConfig struct {
	Interval time.Duration
	Limit    int
}

// EntityLinkSuggestService scans domain-shared entities and writes review_suggestions.
type EntityLinkSuggestService struct {
	cfg        EntityLinkSuggestConfig
	reader     EntityLinkCandidateReader
	classifier EntityLinkClassifier
	writer     EntityReviewSuggestionWriter
}

func NewEntityLinkSuggestService(cfg EntityLinkSuggestConfig, reader EntityLinkCandidateReader, classifier EntityLinkClassifier, writer EntityReviewSuggestionWriter) *EntityLinkSuggestService {
	cfg.Interval = worker.DurationOr(cfg.Interval, 6*time.Hour)
	if cfg.Limit <= 0 {
		cfg.Limit = 100
	}
	return &EntityLinkSuggestService{
		cfg:        cfg,
		reader:     reader,
		classifier: classifier,
		writer:     writer,
	}
}

func (s *EntityLinkSuggestService) Run(ctx context.Context, wg *sync.WaitGroup) {
	if s == nil || s.reader == nil || s.classifier == nil || s.writer == nil {
		return
	}
	worker.Run(ctx, wg, s.run)
}

func (s *EntityLinkSuggestService) run(ctx context.Context) {
	slog.Info("entity link suggest worker started", "interval", s.cfg.Interval)
	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	s.scanOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scanOnce(ctx)
		}
	}
}

func (s *EntityLinkSuggestService) scanOnce(ctx context.Context) {
	docs, err := s.reader.ListWarmEntitiesForLinkSuggest(ctx, s.cfg.Limit)
	if err != nil {
		slog.Warn("entity link suggest load failed", "error", err)
		return
	}
	pairs := entity.FindLinkSuggestPairs(docs)
	if len(pairs) == 0 {
		return
	}
	written := 0
	for _, pair := range pairs {
		var sightA, sightB int
		for _, doc := range docs {
			switch doc.EntityID {
			case pair.EntityA:
				sightA = doc.SightingCount
			case pair.EntityB:
				sightB = doc.SightingCount
			}
		}
		action, why, err := s.classifyPair(ctx, pair, sightA, sightB)
		if err != nil {
			slog.Warn("entity link classify failed",
				"entity_a", pair.EntityA,
				"entity_b", pair.EntityB,
				"error", err,
			)
			continue
		}
		if action == "keep" {
			continue
		}
		suggestion := entity.ReviewSuggestion{
			PeerEntityID: pair.EntityB,
			SharedDomain: pair.SharedDomain,
			Action:       action,
			Why:          why,
			Status:       "pending",
			TS:           time.Now().UTC(),
		}
		if err := s.writer.AppendReviewSuggestion(ctx, pair.EntityA, suggestion); err != nil {
			slog.Warn("entity link suggestion write failed", "entity_id", pair.EntityA, "error", err)
			continue
		}
		peer := entity.ReviewSuggestion{
			PeerEntityID: pair.EntityA,
			SharedDomain: pair.SharedDomain,
			Action:       action,
			Why:          why,
			Status:       "pending",
			TS:           suggestion.TS,
		}
		_ = s.writer.AppendReviewSuggestion(ctx, pair.EntityB, peer)
		written++
	}
	if written > 0 {
		slog.Info("entity link suggestions written", "pairs", written)
	}
}

func (s *EntityLinkSuggestService) classifyPair(ctx context.Context, pair entity.LinkSuggestPair, sightA, sightB int) (string, string, error) {
	res, err := s.classifier.ClassifyEntityLink(ctx, gemini.EntityLinkInput{
		EntityA:      pair.EntityA,
		EntityB:      pair.EntityB,
		SharedDomain: pair.SharedDomain,
		PainA:        pair.PainA,
		PainB:        pair.PainB,
		SightingA:    sightA,
		SightingB:    sightB,
	})
	if err != nil {
		return "", "", err
	}
	return res.Action, res.Why, nil
}

// RunLinkSuggestOnce runs one scan (CLI/tests).
func (s *EntityLinkSuggestService) RunLinkSuggestOnce(ctx context.Context) {
	if s != nil {
		s.scanOnce(ctx)
	}
}
