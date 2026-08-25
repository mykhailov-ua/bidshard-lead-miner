package warmpath

import (
	"context"
	"log/slog"
	"strings"
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
	return queue.NewCapturer("entity_classify", buffer, 128, prepareEntityClassifyEvent, logDroppedEntityClassifyEvent)
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
	AnalyzeInterval          time.Duration
	Debounce                 time.Duration
	LowConfidenceDebounce    time.Duration
	OutreachNarrativeEnabled bool
}

// EntityOutreachNarrator generates cross-lead GTM hooks from entity graph context.
type EntityOutreachNarrator interface {
	BuildEntityOutreach(ctx context.Context, in gemini.EntityOutreachInput) (gemini.EntityOutreachResult, error)
}

// EntityClassifyExtras wires optional entity classify side effects.
type EntityClassifyExtras struct {
	Force    entity.ClassifyForceLister
	LowConf  entity.LowConfidenceHotLister
	LeadHeat entity.LinkedLeadHeatPatcher
	Contacts entity.SightingContactsReader
	Outreach entity.LinkedLeadOutreachPatcher
	Narrator EntityOutreachNarrator
}

// EntityClassifyService batches entity classification with per-entity debounce.
type EntityClassifyService struct {
	cfg        EntityClassifyConfig
	capturer   *EntityClassifyCapturer
	reader     entity.DocReader
	patcher    entity.ClassificationPatcher
	classifier EntityClassifier
	force      entity.ClassifyForceLister
	lowConf    entity.LowConfidenceHotLister
	leadHeat   entity.LinkedLeadHeatPatcher
	contacts   entity.SightingContactsReader
	outreach   entity.LinkedLeadOutreachPatcher
	narrator   EntityOutreachNarrator

	mu       sync.Mutex
	debounce map[string]time.Time
}

func NewEntityClassifyService(
	cfg EntityClassifyConfig,
	capturer *EntityClassifyCapturer,
	reader entity.DocReader,
	patcher entity.ClassificationPatcher,
	classifier EntityClassifier,
	extras EntityClassifyExtras,
) *EntityClassifyService {
	cfg.AnalyzeInterval = worker.DurationOr(cfg.AnalyzeInterval, 5*time.Minute)
	cfg.Debounce = worker.DurationOr(cfg.Debounce, 6*time.Hour)
	cfg.LowConfidenceDebounce = worker.DurationOr(cfg.LowConfidenceDebounce, time.Hour)
	return &EntityClassifyService{
		cfg:        cfg,
		capturer:   capturer,
		reader:     reader,
		patcher:    patcher,
		classifier: classifier,
		force:      extras.Force,
		lowConf:    extras.LowConf,
		leadHeat:   extras.LeadHeat,
		contacts:   extras.Contacts,
		outreach:   extras.Outreach,
		narrator:   extras.Narrator,
		debounce:   make(map[string]time.Time),
	}
}

func (s *EntityClassifyService) Run(ctx context.Context, wg *sync.WaitGroup) {
	if s == nil || s.capturer == nil || s.reader == nil || s.patcher == nil || s.classifier == nil {
		return
	}
	worker.Run(ctx, wg, s.run)
}

func mergeEntityClassifyEvent(prev, ev EntityClassifyEvent) EntityClassifyEvent {
	// Same tick may enqueue duplicate entity_id; OR ForceFresh so split/re-classify is not dropped.
	if prev.EntityID != "" {
		ev.ForceFresh = prev.ForceFresh || ev.ForceFresh
	}
	return ev
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
		s.pollClassifyForce(ctx, pending)
		s.pollLowConfidenceHot(ctx, pending)
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
			if prev, exists := pending[ev.EntityID]; exists {
				ev = mergeEntityClassifyEvent(prev, ev)
			}
			pending[ev.EntityID] = ev
		case <-ticker.C:
			flush()
		}
	}
}

func (s *EntityClassifyService) classifyOne(ctx context.Context, ev EntityClassifyEvent) {
	now := time.Now().UTC()

	doc, ok, err := s.reader.GetEntity(ctx, ev.EntityID)
	if err != nil {
		slog.Warn("entity classify load failed", "entity_id", ev.EntityID, "error", err)
		return
	}
	if !ok {
		return
	}

	lowConf := entity.IsLowConfidenceHot(doc)
	if !s.allowClassify(ev.EntityID, ev.ForceFresh, lowConf, now) {
		return
	}

	sightLimit := entity.MaxEntityClassifySightings
	if lowConf {
		sightLimit = entity.MaxLowConfidenceClassifySightings
	}
	sightings := entity.ClassifySightingsFromDocLimit(doc, sightLimit)
	if len(sightings) == 0 {
		return
	}
	input := gemini.EntityClassifyInput{EntityID: ev.EntityID}
	for _, sight := range sightings {
		item := gemini.EntitySightingInput{
			Source:   sight.Source,
			PostedAt: sight.PostedAt,
			Matched:  sight.Matched,
			Snippet:  sight.Snippet,
		}
		if s.contacts != nil {
			item.ContactsSummary = s.contacts.MaskedContactsSummary(ctx, sight.HashID)
		}
		input.Sightings = append(input.Sightings, item)
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
	if patch.HeatTierDowngrade != "" && s.leadHeat != nil {
		if err := s.leadHeat.PatchLinkedLeadsHeat(ctx, ev.EntityID, doc.HashIDs, entity.LinkedLeadHeatSync{
			HeatScore:     doc.HeatScore,
			HeatTier:      patch.HeatTierDowngrade,
			SightingCount: doc.SightingCount,
			SourceCount:   doc.SourceCount,
		}); err != nil {
			slog.Warn("entity classify lead heat sync failed", "entity_id", ev.EntityID, "error", err)
		}
	}
	s.syncEntityOutreach(ctx, doc, res)
	if s.force != nil {
		if err := s.force.ClearClassifyForce(ctx, ev.EntityID); err != nil {
			slog.Warn("entity classify force clear failed", "entity_id", ev.EntityID, "error", err)
		}
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

func (s *EntityClassifyService) allowClassify(entityID string, forceFresh, lowConf bool, now time.Time) bool {
	if forceFresh {
		return true
	}
	debounce := s.cfg.Debounce
	if lowConf && s.cfg.LowConfidenceDebounce > 0 {
		debounce = s.cfg.LowConfidenceDebounce
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	last, ok := s.debounce[entityID]
	return !ok || now.Sub(last) >= debounce
}

func (s *EntityClassifyService) markClassified(entityID string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.debounce[entityID] = now
}

func (s *EntityClassifyService) pollClassifyForce(ctx context.Context, pending map[string]EntityClassifyEvent) {
	if s.force == nil {
		return
	}
	ids, err := s.force.ListClassifyForce(ctx, 32)
	if err != nil {
		slog.Warn("entity classify force poll failed", "error", err)
		return
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		ev := EntityClassifyEvent{EntityID: id, ForceFresh: true}
		if prev, exists := pending[id]; exists {
			ev = mergeEntityClassifyEvent(prev, ev)
		}
		pending[id] = ev
	}
}

func (s *EntityClassifyService) pollLowConfidenceHot(ctx context.Context, pending map[string]EntityClassifyEvent) {
	if s.lowConf == nil {
		return
	}
	ids, err := s.lowConf.ListLowConfidenceHotEntities(ctx, 32)
	if err != nil {
		slog.Warn("entity low confidence poll failed", "error", err)
		return
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		ev := EntityClassifyEvent{EntityID: id, ForceFresh: true}
		if prev, exists := pending[id]; exists {
			ev = mergeEntityClassifyEvent(prev, ev)
		}
		pending[id] = ev
	}
}

func (s *EntityClassifyService) syncEntityOutreach(ctx context.Context, doc entity.EntityDoc, res gemini.EntityClassifyResult) {
	if !s.cfg.OutreachNarrativeEnabled || s.outreach == nil {
		return
	}
	if !entity.HeatTierMeetsMin(doc.HeatTier, entity.HeatTierHot) {
		// Outreach narrative is sales-facing; skip warm/cold to limit Gemini spend.
		return
	}
	pain := strings.TrimSpace(res.UnifiedPain)
	if pain == "" {
		pain = strings.TrimSpace(doc.UnifiedPain)
	}
	if pain == "" {
		return
	}

	sync := entity.LinkedLeadOutreachSync{
		EntityProof: entity.BuildEntityProofSummary(doc),
	}
	if s.narrator != nil {
		out, err := s.narrator.BuildEntityOutreach(ctx, gemini.EntityOutreachInput{
			EntityID:      doc.EntityID,
			HeatTier:      doc.HeatTier,
			SightingCount: doc.SightingCount,
			SourceCount:   doc.SourceCount,
			UnifiedPain:   pain,
			BuyerIntent:   res.BuyerIntent,
			Sources:       append([]string(nil), doc.SourceFamilies...),
		})
		if err != nil {
			slog.Warn("entity outreach narrative failed", "entity_id", doc.EntityID, "error", err)
		} else {
			if sync.EntityProof == "" {
				sync.EntityProof = out.EntityProof
			}
			sync.OutreachAngle = out.OutreachAngle
		}
	}
	if sync.EntityProof == "" {
		sync.EntityProof = entity.BuildEntityProofSummary(doc)
	}
	if sync.EntityProof == "" && sync.OutreachAngle == "" {
		return
	}
	if err := s.outreach.PatchLinkedLeadsOutreach(ctx, doc.EntityID, doc.HashIDs, sync); err != nil {
		slog.Warn("entity outreach lead patch failed", "entity_id", doc.EntityID, "error", err)
	}
}
