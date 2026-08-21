package warmpath

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/worker"
)

type Config struct {
	AnalyzeInterval         time.Duration
	BatchSize               int
	EngageEnabled           bool
	EnrichEnabled           bool
	PilotTagEnabled         bool
	GeoBlockCountries       []string
	GeoClassifyEnabled      bool // mirrors PARSER_GEO_CLASSIFY; when false warm path skips geo_rejected
	CRMWebhook              *sink.WebhookClient
	CRMWebhookAfterAnalysis bool // PARSER_CRM_WEBHOOK_AFTER_ANALYSIS: notify CRM only after analysis_status=done
}

type Service struct {
	cfg      Config
	capturer *Capturer
	patcher  sink.LeadAnalysisPatcher
	analyzer LeadBatchAnalyzer
	registry *scoring.Registry

	mu     sync.Mutex
	buffer []Event
}

func NewService(cfg Config, capturer *Capturer, patcher sink.LeadAnalysisPatcher, analyzer LeadBatchAnalyzer, reg *scoring.Registry) *Service {
	cfg.AnalyzeInterval = worker.DurationOr(cfg.AnalyzeInterval, 5*time.Minute)
	cfg.BatchSize = worker.IntOr(cfg.BatchSize, 15)
	if len(cfg.GeoBlockCountries) == 0 {
		cfg.GeoBlockCountries = []string{"RU", "BY"}
	}
	return &Service{
		cfg:      cfg,
		capturer: capturer,
		patcher:  patcher,
		analyzer: analyzer,
		registry: reg,
		buffer:   make([]Event, 0, cfg.BatchSize*2),
	}
}

func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) {
	if s == nil || s.capturer == nil || s.patcher == nil || s.analyzer == nil {
		return
	}
	worker.Run(ctx, wg, s.run)
}

func (s *Service) run(ctx context.Context) {
	slog.Info("warm path gemini worker started",
		"analyze_interval", s.cfg.AnalyzeInterval,
		"batch_size", s.cfg.BatchSize,
	)

	var wg sync.WaitGroup
	worker.Run(ctx, &wg, s.ingestLoop)
	worker.Run(ctx, &wg, s.analyzeLoop)
	wg.Wait()
}

func (s *Service) ingestLoop(ctx context.Context) {
	ch := s.capturer.Events()
	if ch == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			s.mu.Lock()
			s.buffer = append(s.buffer, ev)
			s.mu.Unlock()
		}
	}
}

func (s *Service) analyzeLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.AnalyzeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.flush(ctx)
			return
		case <-ticker.C:
			s.flush(ctx)
		}
	}
}

func (s *Service) flush(ctx context.Context) {
	// takeBatch removes events from the buffer before the Gemini call; failed batches are not re-queued.
	batch := s.takeBatch()
	if len(batch) == 0 {
		return
	}

	inputs := make([]gemini.LeadBatchInput, 0, len(batch))
	byID := make(map[string]Event, len(batch))
	for _, ev := range batch {
		byID[ev.HashID] = ev
		inputs = append(inputs, gemini.LeadBatchInput{
			ID:               ev.HashID,
			Source:           ev.Source,
			Priority:         ev.Priority,
			Score:            ev.Score,
			Snippet:          ev.Snippet,
			Contacts:         append([]string(nil), ev.Contacts...),
			ContactTypes:     append([]string(nil), ev.ContactTypes...),
			Stack:            append([]string(nil), ev.Stack...),
			Domain:           ev.Domain,
			RDAPCountry:      ev.RDAPCountry,
			DomainAgeDays:    ev.DomainAgeDays,
			DisplayName:      ev.DisplayName,
			BlockedCountries: append([]string(nil), s.cfg.GeoBlockCountries...),
		})
	}

	results, err := s.analyzer.AnalyzeLeadBatch(ctx, inputs, s.cfg.GeoClassifyEnabled)
	if err != nil {
		slog.Warn("warm path lead batch failed", "count", len(batch), "error", err)
		return // dropped batch; lead stays analysis_status=pending until next sighting
	}

	highMin := highMinFromReg(s.registry)
	for _, res := range results {
		ev, ok := byID[res.HashID]
		if !ok {
			continue
		}
		s.applyResult(ctx, ev, res, highMin)
	}
}

func (s *Service) takeBatch() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.buffer) == 0 {
		return nil
	}
	n := s.cfg.BatchSize
	if n > len(s.buffer) {
		n = len(s.buffer)
	}
	out := append([]Event(nil), s.buffer[:n]...)
	s.buffer = append(s.buffer[:0], s.buffer[n:]...)
	return out
}

func (s *Service) applyResult(ctx context.Context, ev Event, res gemini.LeadBatchResult, highMin int) {
	// Apply deferred Gemini policy: geo hard-reject -> ICP reject only at Low -> hot bump -> pilot/enrich at High.
	blocked := s.cfg.GeoBlockCountries
	if s.cfg.GeoClassifyEnabled && res.Geo.ShouldReject(blocked) {
		if err := s.patcher.PatchLeadAnalysis(ctx, sink.LeadAnalysisPatch{
			HashID:         ev.HashID,
			AnalysisStatus: "geo_rejected",
			Status:         "geo_rejected",
			GeoCountry:     res.Geo.PersonCountry,
			CompanyCountry: res.Geo.CompanyCountry,
			CompanyName:    res.Geo.CompanyName,
			GeoSignals:     append(append([]string(nil), res.Geo.RegistrationSignals...), res.Geo.RUBYSignals...),
			GeoWhy:         res.Geo.Why,
		}); err != nil {
			slog.Warn("warm path geo reject patch failed", "hash_id", ev.HashID, "error", err)
		}
		return
	}

	score := ev.Score
	priority := scoring.Priority(ev.Priority)
	if s.registry != nil {
		score, _ = gemini.ApplyICPToScore(score, res.ICP, highMin)
		priority = scoring.PriorityFromScore(s.registry, score)
		if res.ICP.ICP == "none" && !res.ICP.Hot && priority == scoring.PriorityLow {
			// Do not reject Medium/High leads on ICP alone; inline tgweb path is stricter.
			if err := s.patcher.PatchLeadAnalysis(ctx, sink.LeadAnalysisPatch{
				HashID:         ev.HashID,
				AnalysisStatus: "icp_rejected",
				Status:         "icp_rejected",
				ICP:            res.ICP.ICP,
				ICPWhy:         res.ICP.Why,
				Score:          score,
				Priority:       string(priority),
			}); err != nil {
				slog.Warn("warm path icp reject patch failed", "hash_id", ev.HashID, "error", err)
			}
			return
		}
		if res.ICP.Hot && priority == scoring.PriorityMedium {
			priority = scoring.PriorityHigh
		}
	}

	patch := sink.LeadAnalysisPatch{
		HashID:         ev.HashID,
		AnalysisStatus: "done",
		Score:          score,
		Priority:       string(priority),
		ICP:            res.ICP.ICP,
		Hot:            res.ICP.Hot,
		SpendTier:      res.ICP.SpendTier,
		ICPWhy:         res.ICP.Why,
		GeoCountry:     res.Geo.PersonCountry,
		CompanyCountry: res.Geo.CompanyCountry,
		CompanyName:    res.Geo.CompanyName,
		GeoSignals:     append(append([]string(nil), res.Geo.RegistrationSignals...), res.Geo.RUBYSignals...),
		GeoWhy:         res.Geo.Why,
	}

	if s.cfg.PilotTagEnabled && priority == scoring.PriorityHigh {
		if s.cfg.EngageEnabled {
			patch.PilotQualified = res.PilotQualified
			patch.PilotWhy = res.Engagement.PilotWhy
			patch.Tags = append([]string(nil), res.PilotTags...)
			patch.OutreachChannel = res.Engagement.OutreachChannel
			patch.OutreachAngle = res.Engagement.OutreachAngle
			patch.OutreachDraft = res.Engagement.OutreachDraft
		}
	}

	if s.cfg.EnrichEnabled && priority == scoring.PriorityHigh {
		patch.CompanyType = res.Enrichment.CompanyType
		patch.EnrichSummary = res.Enrichment.Summary
		patch.GeoConfidence = res.Enrichment.GeoConfidence
	}

	if err := s.patcher.PatchLeadAnalysis(ctx, patch); err != nil {
		slog.Warn("warm path analysis patch failed", "hash_id", ev.HashID, "error", err)
		return
	}
	// Defer-mode CRM contract: webhook only after Mongo patch succeeds and lead is not geo/icp rejected.
	if s.cfg.CRMWebhook != nil && s.cfg.CRMWebhookAfterAnalysis {
		s.cfg.CRMWebhook.NotifyLead(leadForCRM(ev, patch))
	}
	slog.Debug("warm path lead analyzed", "hash_id", ev.HashID, "priority", patch.Priority, "icp", patch.ICP)
}

func highMinFromReg(reg *scoring.Registry) int {
	if reg == nil {
		return 0
	}
	_, _, _, highMin, _ := reg.Snapshot()
	return highMin
}

// DeferredCRMWebhook reports CRM notify timing for defer + after-analysis mode.
func (s *Service) DeferredCRMWebhook() bool {
	if s == nil {
		return false
	}
	return s.cfg.CRMWebhookAfterAnalysis && s.cfg.CRMWebhook != nil
}
