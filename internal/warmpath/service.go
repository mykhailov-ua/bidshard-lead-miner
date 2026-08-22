package warmpath

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/metrics"
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
	CRMWebhookAfterAnalysis bool          // PARSER_CRM_WEBHOOK_AFTER_ANALYSIS: notify CRM only after analysis_status=done
	RetryMaxAttempts        int           // WARM_ANALYSIS_RETRY_MAX; Gemini batch retries before DLQ
	RetryBaseDelay          time.Duration // WARM_ANALYSIS_RETRY_BASE; doubled each retry
	PendingRescanInterval   time.Duration // WARM_ANALYSIS_PENDING_SCAN_INTERVAL; 0 disables Mongo rescan
	PendingStaleAge         time.Duration // WARM_ANALYSIS_PENDING_STALE; min age of analysis_status=pending
	ShutdownDrainTimeout    time.Duration // WARM_ANALYSIS_SHUTDOWN_DRAIN; flush budget after ctx cancel
	EngageMediumEnabled     bool          // PARSER_GEMINI_ENGAGE_MEDIUM: lite outreach_angle for Medium+warm entity
}

// ServiceExtras wires optional Mongo pending scan, DLQ, embed prescan, cluster, junk insert.
type ServiceExtras struct {
	PendingScanner PendingLeadScanner
	DLQ            AnalysisDLQWriter
	Prescan        EmbedPrescanner
	Cluster        LeadClusterer
	Junk           WarmJunkInserter
	EngageMedium   MediumEngager
}

type Service struct {
	cfg            Config
	capturer       *Capturer
	patcher        sink.LeadAnalysisPatcher
	analyzer       LeadBatchAnalyzer
	registry       *scoring.Registry
	pendingScanner PendingLeadScanner
	dlq            AnalysisDLQWriter
	prescan        EmbedPrescanner
	cluster        LeadClusterer
	junk           WarmJunkInserter
	engageMedium   MediumEngager

	mu     sync.Mutex
	buffer []Event
}

func NewService(cfg Config, capturer *Capturer, patcher sink.LeadAnalysisPatcher, analyzer LeadBatchAnalyzer, reg *scoring.Registry, extras ServiceExtras) *Service {
	cfg.AnalyzeInterval = worker.DurationOr(cfg.AnalyzeInterval, 5*time.Minute)
	cfg.BatchSize = worker.IntOr(cfg.BatchSize, 15)
	cfg.RetryMaxAttempts = worker.IntOr(cfg.RetryMaxAttempts, 3)
	cfg.RetryBaseDelay = worker.DurationOr(cfg.RetryBaseDelay, 5*time.Second)
	cfg.PendingRescanInterval = worker.DurationOr(cfg.PendingRescanInterval, 15*time.Minute)
	cfg.PendingStaleAge = worker.DurationOr(cfg.PendingStaleAge, time.Hour)
	cfg.ShutdownDrainTimeout = worker.DurationOr(cfg.ShutdownDrainTimeout, 2*time.Minute)
	if len(cfg.GeoBlockCountries) == 0 {
		cfg.GeoBlockCountries = []string{"RU", "BY"}
	}
	return &Service{
		cfg:            cfg,
		capturer:       capturer,
		patcher:        patcher,
		analyzer:       analyzer,
		registry:       reg,
		pendingScanner: extras.PendingScanner,
		dlq:            extras.DLQ,
		prescan:        extras.Prescan,
		cluster:        extras.Cluster,
		junk:           extras.Junk,
		engageMedium:   extras.EngageMedium,
		buffer:         make([]Event, 0, cfg.BatchSize*2),
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
		"retry_max", s.cfg.RetryMaxAttempts,
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
			s.enqueueDedupe([]Event{ev})
		}
	}
}

func (s *Service) analyzeLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.AnalyzeInterval)
	defer ticker.Stop()

	var rescanTicker *time.Ticker
	var rescanC <-chan time.Time
	if s.pendingScanner != nil && s.cfg.PendingRescanInterval > 0 {
		rescanTicker = time.NewTicker(s.cfg.PendingRescanInterval)
		defer rescanTicker.Stop()
		rescanC = rescanTicker.C
	}

	for {
		select {
		case <-ctx.Done():
			// Parent ctx is cancelled on shutdown; use a fresh timeout so in-flight Gemini batches can finish.
			drainCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownDrainTimeout)
			s.flushAll(drainCtx)
			cancel()
			return
		case <-ticker.C:
			s.flush(ctx)
		case <-rescanC:
			s.rescanPending(ctx)
		}
	}
}

func (s *Service) flush(ctx context.Context) {
	batch := s.takeBatch()
	if len(batch) == 0 {
		return
	}
	s.processBatch(ctx, batch)
}

func (s *Service) flushAll(ctx context.Context) {
	for {
		batch := s.takeBatch()
		if len(batch) == 0 {
			return
		}
		s.processBatch(ctx, batch)
	}
}

func (s *Service) processBatch(ctx context.Context, batch []Event) {
	// Batch is already removed from the in-memory buffer; on shutdown mid-retry we requeue instead of DLQ.
	batch = s.filterWarmPrescan(ctx, batch)
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

	delay := s.cfg.RetryBaseDelay
	maxAttempts := s.cfg.RetryMaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1 // tests may construct Service without NewService defaults
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		results, err := s.analyzer.AnalyzeLeadBatch(ctx, inputs, s.cfg.GeoClassifyEnabled)
		if err == nil {
			highMin := highMinFromReg(s.registry)
			for _, res := range results {
				ev, ok := byID[res.HashID]
				if !ok {
					continue
				}
				s.applyResult(ctx, ev, res, highMin)
			}
			return
		}
		lastErr = err
		if attempt >= maxAttempts {
			break
		}
		slog.Warn("warm path lead batch retry",
			"count", len(batch),
			"attempt", attempt,
			"error", err,
		)
		select {
		case <-ctx.Done():
			s.requeueFront(batch)
			return
		case <-time.After(delay):
			delay *= 2
		}
	}

	metrics.RecordWarmAnalysisFailed(len(batch))
	if s.dlq != nil {
		if err := s.dlq.InsertWarmAnalysisFailures(ctx, batch, maxAttempts, lastErr); err != nil {
			slog.Warn("warm path dlq insert failed", "count", len(batch), "error", err)
		}
	}
	// Lead documents stay analysis_status=pending; ops replay from warm_analysis_dlq or wait for rescan.
	slog.Warn("warm path lead batch failed after retries",
		"count", len(batch),
		"attempts", maxAttempts,
		"error", lastErr,
	)
}

func (s *Service) rescanPending(ctx context.Context) {
	if s.pendingScanner == nil {
		return
	}
	if n, err := s.pendingScanner.CountPendingAnalysis(ctx); err == nil {
		metrics.SetWarmAnalysisPending(n)
	} else {
		slog.Debug("warm path pending count failed", "error", err)
	}
	events, err := s.pendingScanner.ListStalePendingLeads(ctx, s.cfg.PendingStaleAge, s.cfg.BatchSize)
	if err != nil {
		slog.Warn("warm path pending rescan failed", "error", err)
		return
	}
	if len(events) == 0 {
		return
	}
	added := s.enqueueDedupe(events)
	if added > 0 {
		slog.Info("warm path re-queued stale pending leads", "count", added)
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

func (s *Service) requeueFront(batch []Event) {
	if len(batch) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffer = append(append([]Event(nil), batch...), s.buffer...)
}

func (s *Service) enqueueDedupe(events []Event) int {
	// Rescan and hot-path capture can reference the same hash_id in one tick.
	if len(events) == 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := make(map[string]struct{}, len(s.buffer))
	for _, ev := range s.buffer {
		if ev.HashID != "" {
			existing[ev.HashID] = struct{}{}
		}
	}
	added := 0
	for _, ev := range events {
		if ev.HashID == "" {
			continue
		}
		if _, ok := existing[ev.HashID]; ok {
			continue
		}
		s.buffer = append(s.buffer, ev)
		existing[ev.HashID] = struct{}{}
		added++
	}
	return added
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
		if inline := strings.TrimSpace(ev.InlineICP); inline != "" {
			warm := strings.TrimSpace(res.ICP.ICP)
			if warm != "" && !strings.EqualFold(inline, warm) {
				metrics.RecordICPDrift(ev.Source)
			}
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
	if q := scoring.ScoreContactQuality(ev.Contacts); q != "" {
		patch.ContactQuality = q
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

	if s.cfg.EngageMediumEnabled && priority == scoring.PriorityMedium && entity.HeatTierMeetsMin(ev.HeatTier, entity.HeatTierWarm) {
		if s.engageMedium != nil {
			engage, err := s.engageMedium.ClassifyEngagement(ctx, gemini.EngagementInput{
				Text:         ev.Snippet,
				Stack:        append([]string(nil), ev.Stack...),
				ContactTypes: append([]string(nil), ev.ContactTypes...),
			})
			if err != nil {
				slog.Debug("warm medium engage failed", "hash_id", ev.HashID, "error", err)
			} else if angle := engage.OutreachAngle; angle != "" {
				patch.OutreachAngle = angle
			}
		}
	}

	if s.cluster != nil && scoring.MeetsMinPriority(priority, scoring.PriorityHigh) {
		if dup, clusterOf, err := s.cluster.CheckDuplicate(ctx, ev.HashID, ev.Snippet); err != nil {
			slog.Warn("warm lead cluster check failed", "hash_id", ev.HashID, "error", err)
		} else if dup {
			patch.AnalysisStatus = "duplicate"
			patch.Status = "duplicate"
			patch.DuplicateOf = clusterOf
			if err := s.patcher.PatchLeadAnalysis(ctx, patch); err != nil {
				slog.Warn("warm path duplicate patch failed", "hash_id", ev.HashID, "error", err)
			}
			slog.Info("warm path semantic duplicate", "hash_id", ev.HashID, "duplicate_of", clusterOf)
			return
		}
	}

	if err := s.patcher.PatchLeadAnalysis(ctx, patch); err != nil {
		slog.Warn("warm path analysis patch failed", "hash_id", ev.HashID, "error", err)
		return
	}
	// Defer-mode CRM contract: webhook only after Mongo patch succeeds and lead is not geo/icp rejected.
	if s.cfg.CRMWebhook != nil && s.cfg.CRMWebhookAfterAnalysis {
		s.cfg.CRMWebhook.NotifyLead(leadForCRM(ev, patch))
	}
	if s.cluster != nil && scoring.MeetsMinPriority(priority, scoring.PriorityHigh) {
		if err := s.cluster.Record(ctx, ev.HashID, ev.Snippet); err != nil {
			slog.Debug("warm lead cluster record failed", "hash_id", ev.HashID, "error", err)
		}
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

// ErrWarmAnalysisExhausted is returned by test analyzers to simulate batch failure.
var ErrWarmAnalysisExhausted = errors.New("warm analysis exhausted")
