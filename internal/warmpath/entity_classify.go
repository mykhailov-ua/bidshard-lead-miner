package warmpath

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/queue"
	"github.com/bidshard/parser/internal/worker"
)

// EntityClassifyEvent queues one entity for Gemini same-actor validation.
type EntityClassifyEvent struct {
	EntityID   string
	ForceFresh bool
	TS         time.Time
}

// EntityClassifyCapturer enqueues entity classify jobs without blocking the hot path.
type EntityClassifyCapturer = queue.Capturer[EntityClassifyEvent]

func NewEntityClassifyCapturer(buffer int) *EntityClassifyCapturer {
	return queue.NewCapturer(buffer, 128, prepareEntityClassifyEvent, logDroppedEntityClassifyEvent)
}

func prepareEntityClassifyEvent(ev EntityClassifyEvent) EntityClassifyEvent {
	if ev.TS.IsZero() {
		ev.TS = queue.ZeroTime(ev.TS)
	}
	return ev
}

func logDroppedEntityClassifyEvent(ev EntityClassifyEvent) {
	slog.Debug("entity classify queue full, dropping", "entity_id", ev.EntityID)
}

// EntityClassifier runs Gemini entity-level validation.
type EntityClassifier interface {
	ClassifyEntity(ctx context.Context, in gemini.EntityClassifyInput) (gemini.EntityClassifyResult, error)
}

// EntityClassifyConfig controls warm-path entity Gemini worker.
type EntityClassifyConfig struct {
	AnalyzeInterval time.Duration
	Debounce        time.Duration
}

// EntityClassifyService batches entity classification with per-entity debounce.
type EntityClassifyService struct {
	cfg        EntityClassifyConfig
	capturer   *EntityClassifyCapturer
	reader     entity.DocReader
	patcher    entity.ClassificationPatcher
	classifier EntityClassifier

	mu       sync.Mutex
	debounce map[string]time.Time
}

func NewEntityClassifyService(
	cfg EntityClassifyConfig,
	capturer *EntityClassifyCapturer,
	reader entity.DocReader,
	patcher entity.ClassificationPatcher,
	classifier EntityClassifier,
) *EntityClassifyService {
	cfg.AnalyzeInterval = worker.DurationOr(cfg.AnalyzeInterval, 5*time.Minute)
	cfg.Debounce = worker.DurationOr(cfg.Debounce, 6*time.Hour)
	return &EntityClassifyService{
		cfg:        cfg,
		capturer:   capturer,
		reader:     reader,
		patcher:    patcher,
		classifier: classifier,
		debounce:   make(map[string]time.Time),
	}
}

func (s *EntityClassifyService) Run(ctx context.Context, wg *sync.WaitGroup) {
	if s == nil || s.capturer == nil || s.reader == nil || s.patcher == nil || s.classifier == nil {
		return
	}
	worker.Run(ctx, wg, s.run)
}

func (s *EntityClassifyService) run(ctx context.Context) {
	slog.Info("entity classify worker started",
		"analyze_interval", s.cfg.AnalyzeInterval,
		"debounce", s.cfg.Debounce,
	)
	ch := s.capturer.Events()
	if ch == nil {
		return
	}
	ticker := time.NewTicker(s.cfg.AnalyzeInterval)
	defer ticker.Stop()

	pending := make(map[string]EntityClassifyEvent)
	flush := func() {
		for id, ev := range pending {
			s.classifyOne(ctx, ev)
			delete(pending, id)
		}
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case ev, ok := <-ch:
			if !ok {
				flush()
				return
			}
			if ev.EntityID == "" {
				continue
			}
			pending[ev.EntityID] = ev
		case <-ticker.C:
			flush()
		}
	}
}

func (s *EntityClassifyService) classifyOne(ctx context.Context, ev EntityClassifyEvent) {
	now := time.Now().UTC()
	if !s.allowClassify(ev.EntityID, ev.ForceFresh, now) {
		return
	}

	doc, ok, err := s.reader.GetEntity(ctx, ev.EntityID)
	if err != nil {
		slog.Warn("entity classify load failed", "entity_id", ev.EntityID, "error", err)
		return
	}
	if !ok {
		return
	}

	sightings := entity.ClassifySightingsFromDoc(doc)
	if len(sightings) == 0 {
		return
	}
	input := gemini.EntityClassifyInput{EntityID: ev.EntityID}
	for _, sight := range sightings {
		input.Sightings = append(input.Sightings, gemini.EntitySightingInput{
			Source:   sight.Source,
			PostedAt: sight.PostedAt,
			Matched:  sight.Matched,
			Snippet:  sight.Snippet,
		})
	}

	res, err := s.classifier.ClassifyEntity(ctx, input)
	if err != nil {
		slog.Warn("entity classify gemini failed", "entity_id", ev.EntityID, "error", err)
		return
	}

	patch := entity.ApplyEntityClassification(&doc, entity.GeminiClassifyResult{
		SameActor:        res.SameActor,
		ActorConfidence:  res.ActorConfidence,
		UnifiedPain:      res.UnifiedPain,
		BuyerIntent:      res.BuyerIntent,
		SplitRecommended: res.SplitRecommended,
		Why:              res.Why,
	}, now)
	if err := s.patcher.PatchEntityClassification(ctx, ev.EntityID, patch); err != nil {
		slog.Warn("entity classify patch failed", "entity_id", ev.EntityID, "error", err)
		return
	}
	s.markClassified(ev.EntityID, now)
	slog.Info("entity classified",
		"entity_id", ev.EntityID,
		"same_actor", res.SameActor,
		"actor_confidence", res.ActorConfidence,
		"buyer_intent", res.BuyerIntent,
		"needs_review", patch.NeedsReview,
	)
}

func (s *EntityClassifyService) allowClassify(entityID string, forceFresh bool, now time.Time) bool {
	if forceFresh {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.debounce[entityID]
	return !ok || now.Sub(last) >= s.cfg.Debounce
}

func (s *EntityClassifyService) markClassified(entityID string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.debounce[entityID] = now
}
