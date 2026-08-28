package coldpath

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/batch"
	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/salesexport"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/worker"
)

type Config struct {
	AnalyzeInterval          time.Duration
	ReportInterval           time.Duration
	BatchSize                int
	FlushInterval            time.Duration
	FlushBatch               int
	KeywordDiffEvery         int
	KeywordDiffDir           string
	DiscoverDiffEvery        int
	DiscoverDiffDir          string
	PainVocabDiffEvery       int
	EmbedThreshold           float64
	HardRejectShadowDailyCap int
}

// ServiceExtras wires optional cold-path audit and enrichment workers.
type ServiceExtras struct {
	Stale          *StaleLeadRegrader
	DupSuggest     *DuplicateSuggestScanner
	GeoAudit       *GeoAuditRunner
	WebhookAudit   *WebhookAuditReporter
	SourceStats    *sink.SourceStatsStore
	ChannelsPath   string
	SalesExportRU  bool
	SalesExportDir string
}

type Service struct {
	cfg            Config
	capturer       *Capturer
	junk           *sink.JunkStore
	gemini         *gemini.Client
	crm            BoostStore
	embed          *sink.EmbeddingStore
	keywordStats   *sink.KeywordStatsStore
	registry       *scoring.Registry
	leads          BoostLeadLookup
	entityPains    entity.PainSampleLister
	stale          *StaleLeadRegrader
	dupSuggest     *DuplicateSuggestScanner
	geoAudit       *GeoAuditRunner
	webhookAudit   *WebhookAuditReporter
	sourceStats    *sink.SourceStatsStore
	channelsPath   string
	salesExportRU  bool
	salesExportDir string

	lastReportEnd      time.Time
	reportCount        int
	embedThreshold     float64
	keywordDiffEvery   int
	keywordDiffDir     string
	discoverDiffEvery  int
	discoverDiffDir    string
	painVocabDiffEvery int
	shadowDailyCap     int
}

func NewService(cfg Config, capturer *Capturer, junk *sink.JunkStore, client *gemini.Client, crm BoostStore, embed *sink.EmbeddingStore, keywordStats *sink.KeywordStatsStore, registry *scoring.Registry, leads BoostLeadLookup, entityPains entity.PainSampleLister, extras ServiceExtras) *Service {
	cfg.AnalyzeInterval = worker.DurationOr(cfg.AnalyzeInterval, 15*time.Minute)
	cfg.ReportInterval = worker.DurationOr(cfg.ReportInterval, 6*time.Hour)
	cfg.BatchSize = worker.IntOr(cfg.BatchSize, 20)
	cfg.FlushInterval = worker.DurationOr(cfg.FlushInterval, 2*time.Second)
	cfg.FlushBatch = worker.IntOr(cfg.FlushBatch, 50)
	cfg.EmbedThreshold = worker.FloatOr(cfg.EmbedThreshold, 0.92)
	cfg.KeywordDiffEvery = worker.IntOr(cfg.KeywordDiffEvery, 5)
	if cfg.DiscoverDiffEvery <= 0 {
		cfg.DiscoverDiffEvery = cfg.KeywordDiffEvery
	}
	if cfg.DiscoverDiffDir == "" {
		cfg.DiscoverDiffDir = cfg.KeywordDiffDir
	}
	if cfg.HardRejectShadowDailyCap <= 0 {
		cfg.HardRejectShadowDailyCap = 10
	}
	painEvery := cfg.PainVocabDiffEvery
	if painEvery <= 0 {
		painEvery = cfg.KeywordDiffEvery
	}
	return &Service{
		cfg:                cfg,
		capturer:           capturer,
		junk:               junk,
		gemini:             client,
		crm:                crm,
		embed:              embed,
		keywordStats:       keywordStats,
		registry:           registry,
		leads:              leads,
		entityPains:        entityPains,
		stale:              extras.Stale,
		dupSuggest:         extras.DupSuggest,
		geoAudit:           extras.GeoAudit,
		webhookAudit:       extras.WebhookAudit,
		sourceStats:        extras.SourceStats,
		channelsPath:       extras.ChannelsPath,
		salesExportRU:      extras.SalesExportRU,
		salesExportDir:     extras.SalesExportDir,
		lastReportEnd:      time.Now().UTC(),
		embedThreshold:     cfg.EmbedThreshold,
		keywordDiffEvery:   cfg.KeywordDiffEvery,
		keywordDiffDir:     cfg.KeywordDiffDir,
		discoverDiffEvery:  cfg.DiscoverDiffEvery,
		discoverDiffDir:    cfg.DiscoverDiffDir,
		painVocabDiffEvery: painEvery,
		shadowDailyCap:     cfg.HardRejectShadowDailyCap,
	}
}

func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) {
	if s == nil || s.capturer == nil || s.junk == nil || s.gemini == nil {
		return
	}
	worker.Run(ctx, wg, s.run)
}

func (s *Service) run(ctx context.Context) {
	slog.Info("cold path gemini worker started",
		"analyze_interval", s.cfg.AnalyzeInterval,
		"report_interval", s.cfg.ReportInterval,
		"batch_size", s.cfg.BatchSize,
		"embed_threshold", s.embedThreshold,
		"keyword_diff_every", s.keywordDiffEvery,
	)

	var wg sync.WaitGroup
	worker.Run(ctx, &wg, s.ingestLoop)
	worker.Run(ctx, &wg, s.analyzeLoop)
	worker.Run(ctx, &wg, s.reportLoop)
	if s.stale != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.stale.Run(ctx)
		}()
	}
	if s.dupSuggest != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.dupSuggest.Run(ctx)
		}()
	}
	if s.geoAudit != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.geoAudit.Run(ctx)
		}()
	}
	if s.webhookAudit != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.webhookAudit.Run(ctx)
		}()
	}
	wg.Wait()

	if dropped := s.capturer.Dropped(); dropped > 0 {
		slog.Warn("cold path junk queue dropped events", "dropped", dropped)
	}
	slog.Info("cold path gemini worker stopped")
}

func (s *Service) ingestLoop(ctx context.Context) {
	batch.RunTickerFlush(ctx, s.capturer.Events(), s.cfg.FlushBatch, s.cfg.FlushInterval, func(events []Event) error {
		if len(events) == 0 {
			return nil
		}
		docs := make([]sink.JunkDoc, 0, len(events))
		for _, ev := range events {
			docs = append(docs, JunkDocFromEvent(ev))
		}
		if err := s.junk.InsertMany(ctx, docs); err != nil {
			slog.Warn("junk insert failed", "count", len(docs), "error", err)
			return err
		}
		slog.Debug("junk batch inserted", "count", len(docs))
		return nil
	})
}

func (s *Service) analyzeLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.AnalyzeInterval)
	defer ticker.Stop()

	s.runAnalyze(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runAnalyze(ctx)
		}
	}
}

func (s *Service) runAnalyze(ctx context.Context) {
	docs, err := s.junk.FindPendingAnalysisExcluding(ctx, ReasonHardRejectShadow, s.cfg.BatchSize)
	if err != nil {
		slog.Warn("junk find pending failed", "error", err)
		return
	}
	if len(docs) < s.cfg.BatchSize {
		shadowBudget := s.shadowAnalyzeBudget(ctx)
		if shadowBudget > 0 {
			shadow, err := s.junk.FindPendingByReason(ctx, ReasonHardRejectShadow, shadowBudget)
			if err != nil {
				slog.Warn("junk find shadow pending failed", "error", err)
			} else {
				docs = append(docs, shadow...)
			}
		}
	}
	if len(docs) == 0 {
		return
	}

	results, err := s.gemini.AnalyzeJunkBatch(ctx, docs)
	if err != nil {
		metrics.RecordGeminiJunkBatchFailed()
		slog.Warn("gemini analyze batch failed", "count", len(docs), "error", err)
		return
	}

	byID := make(map[string]gemini.AnalyzeResult, len(results))
	for _, r := range results {
		byID[r.ID] = r
	}

	now := time.Now().UTC()
	applied := 0
	for _, doc := range docs {
		r, ok := byID[doc.ID.Hex()]
		if !ok {
			continue
		}
		analysis := sink.JunkAnalysis{
			AnalyzedAt:  now,
			Category:    r.Category,
			Why:         r.Why,
			Suggestions: r.Suggestions,
		}
		if err := s.junk.SaveAnalysis(ctx, doc.ID, analysis); err != nil {
			slog.Warn("junk save analysis failed", "id", doc.ID.Hex(), "error", err)
			continue
		}
		// Run CRM boost and semantic dedup after analysis is persisted.
		s.processFalseNegative(ctx, doc, r)
		s.processSemanticDedup(ctx, doc)
		applied++
	}

	slog.Info("gemini junk batch analyzed", "requested", len(docs), "applied", applied)
	s.runBoostWorker(ctx)
}

func (s *Service) shadowAnalyzeBudget(ctx context.Context) int {
	if s == nil || s.junk == nil || s.shadowDailyCap <= 0 {
		return 0
	}
	start := time.Now().UTC().Truncate(24 * time.Hour)
	n, err := s.junk.CountAnalyzedByReasonSince(ctx, ReasonHardRejectShadow, start)
	if err != nil {
		return s.shadowDailyCap
	}
	remaining := s.shadowDailyCap - int(n)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *Service) reportLoop(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runReport(ctx)
		}
	}
}

func (s *Service) runReport(ctx context.Context) {
	periodEnd := time.Now().UTC()
	periodStart := s.lastReportEnd
	if periodEnd.Sub(periodStart) < time.Minute {
		return
	}

	total, err := s.junk.CountSince(ctx, periodStart)
	if err != nil {
		slog.Warn("junk count since failed", "error", err)
		return
	}
	if total == 0 {
		s.lastReportEnd = periodEnd
		return
	}

	stats, err := s.junk.ReasonBreakdown(ctx, periodStart)
	if err != nil {
		slog.Warn("junk reason breakdown failed", "error", err)
		return
	}

	sourceStats, err := s.junk.SourceBreakdown(ctx, periodStart, 20)
	if err != nil {
		slog.Warn("junk source breakdown failed", "error", err)
		return
	}

	samples, err := s.junk.SampleAnalyzed(ctx, periodStart, 30)
	if err != nil {
		slog.Warn("junk sample analyzed failed", "error", err)
		return
	}

	in := gemini.ReportInputFromStore(periodStart, periodEnd, total, stats, sourceStats, samples)
	result, err := s.gemini.BuildJunkReport(ctx, in)
	if err != nil {
		slog.Warn("gemini report failed", "error", err)
		return
	}

	doc := sink.JunkReportDoc{
		TS:                      periodEnd,
		PeriodFrom:              periodStart,
		PeriodTo:                periodEnd,
		SampleCount:             int(total),
		Summary:                 result.Summary,
		TopReasons:              mergeReasonStats(stats, result.TopReasons),
		FalseNegativeCandidates: result.FalseNegativeCandidates,
		Recommendations:         result.Recommendations,
		KeywordSuggestions:      result.KeywordSuggestions,
		SourceStats:             sourceStats,
	}
	if err := s.junk.InsertReport(ctx, doc); err != nil {
		slog.Warn("junk report insert failed", "error", err)
		return
	}
	if s.salesExportRU {
		report := salesexport.JunkReportFromDoc(doc)
		if s.gemini != nil {
			if localized, err := s.gemini.LocalizeJunkReportRU(ctx, doc); err == nil {
				report = localized
			} else {
				slog.Warn("sales junk report RU localize failed", "error", err)
			}
		}
		dir := s.salesExportDir
		if dir == "" {
			dir = "data/export/sales"
		}
		if path, err := salesexport.WriteJSON(dir, "junk_report_ru", report); err == nil {
			slog.Info("sales junk report RU written", "path", path)
		} else {
			slog.Warn("sales junk report RU write failed", "error", err)
		}
	}

	reportID := doc.ID.Hex()
	s.maybeWriteKeywordDiff(ctx, reportID, result, periodStart)
	s.maybeWriteDiscoverDiff(ctx, reportID, result, periodStart)
	s.maybeWritePainVocabDiff(ctx, reportID, periodStart)
	if s.sourceStats != nil && s.channelsPath != "" {
		if stats, err := s.sourceStats.ListAll(ctx); err == nil {
			if path, err := discover.WriteDorkRankReport(s.channelsPath, s.keywordDiffDir, stats); err == nil && path != "" {
				slog.Info("dork rank report written", "path", path)
			}
		}
	}

	s.lastReportEnd = periodEnd
	slog.Info("gemini junk report saved",
		"period_from", periodStart.Format(time.RFC3339),
		"period_to", periodEnd.Format(time.RFC3339),
		"junk_count", total,
		"false_negative_candidates", result.FalseNegativeCandidates,
		"keyword_suggestions", len(result.KeywordSuggestions),
		"report_id", reportID,
	)
}

func mergeReasonStats(db []sink.ReasonCount, llm []sink.ReasonCount) []sink.ReasonCount {
	if len(llm) == 0 {
		return db
	}
	whyByReason := make(map[string]string, len(llm))
	for _, r := range llm {
		if r.Why != "" {
			whyByReason[r.Reason] = r.Why
		}
	}
	out := make([]sink.ReasonCount, 0, len(db))
	for _, r := range db {
		if why, ok := whyByReason[r.Reason]; ok {
			r.Why = why
		}
		out = append(out, r)
	}
	return out
}

func (s *Service) RunReportOnce(ctx context.Context) {
	s.runReport(ctx)
}
