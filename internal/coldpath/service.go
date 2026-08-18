package coldpath

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

type Config struct {
	AnalyzeInterval  time.Duration
	ReportInterval   time.Duration
	BatchSize        int
	FlushInterval    time.Duration
	FlushBatch       int
	KeywordDiffEvery int
	KeywordDiffDir   string
	EmbedThreshold   float64
}

type Service struct {
	cfg          Config
	capturer     *Capturer
	junk         *sink.JunkStore
	gemini       *gemini.Client
	crm          *sink.CrmBoostStore
	embed        *sink.EmbeddingStore
	keywordStats *sink.KeywordStatsStore

	lastReportEnd    time.Time
	reportCount      int
	embedThreshold   float64
	keywordDiffEvery int
	keywordDiffDir   string
}

func NewService(cfg Config, capturer *Capturer, junk *sink.JunkStore, client *gemini.Client, crm *sink.CrmBoostStore, embed *sink.EmbeddingStore, keywordStats *sink.KeywordStatsStore) *Service {
	if cfg.AnalyzeInterval <= 0 {
		cfg.AnalyzeInterval = 15 * time.Minute
	}
	if cfg.ReportInterval <= 0 {
		cfg.ReportInterval = 6 * time.Hour
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 20
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 2 * time.Second
	}
	if cfg.FlushBatch <= 0 {
		cfg.FlushBatch = 50
	}
	if cfg.EmbedThreshold <= 0 {
		cfg.EmbedThreshold = 0.92
	}
	if cfg.KeywordDiffEvery <= 0 {
		cfg.KeywordDiffEvery = 5
	}
	return &Service{
		cfg:              cfg,
		capturer:         capturer,
		junk:             junk,
		gemini:           client,
		crm:              crm,
		embed:            embed,
		keywordStats:     keywordStats,
		lastReportEnd:    time.Now().UTC(),
		embedThreshold:   cfg.EmbedThreshold,
		keywordDiffEvery: cfg.KeywordDiffEvery,
		keywordDiffDir:   cfg.KeywordDiffDir,
	}
}

func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) {
	if s == nil || s.capturer == nil || s.junk == nil || s.gemini == nil {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		s.run(ctx)
	}()
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
	wg.Add(3)
	go func() {
		defer wg.Done()
		s.ingestLoop(ctx)
	}()
	go func() {
		defer wg.Done()
		s.analyzeLoop(ctx)
	}()
	go func() {
		defer wg.Done()
		s.reportLoop(ctx)
	}()
	wg.Wait()

	if dropped := s.capturer.Dropped(); dropped > 0 {
		slog.Warn("cold path junk queue dropped events", "dropped", dropped)
	}
	slog.Info("cold path gemini worker stopped")
}

func (s *Service) ingestLoop(ctx context.Context) {
	events := s.capturer.Events()
	if events == nil {
		return
	}

	buf := make([]sink.JunkDoc, 0, s.cfg.FlushBatch)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		batch := append([]sink.JunkDoc(nil), buf...)
		buf = buf[:0]
		if err := s.junk.InsertMany(ctx, batch); err != nil {
			slog.Warn("junk insert failed", "count", len(batch), "error", err)
		} else {
			slog.Debug("junk batch inserted", "count", len(batch))
		}
	}

	ticker := time.NewTicker(s.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			flush()
		case ev, ok := <-events:
			if !ok {
				flush()
				return
			}
			buf = append(buf, JunkDocFromEvent(ev))
			if len(buf) >= s.cfg.FlushBatch {
				flush()
			}
		}
	}
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
	docs, err := s.junk.FindPendingAnalysis(ctx, s.cfg.BatchSize)
	if err != nil {
		slog.Warn("junk find pending failed", "error", err)
		return
	}
	if len(docs) == 0 {
		return
	}

	results, err := s.gemini.AnalyzeJunkBatch(ctx, docs)
	if err != nil {
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
		s.processFalseNegative(ctx, doc, r)
		s.processSemanticDedup(ctx, doc)
		applied++
	}

	slog.Info("gemini junk batch analyzed", "requested", len(docs), "applied", applied)
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

	samples, err := s.junk.SampleAnalyzed(ctx, periodStart, 30)
	if err != nil {
		slog.Warn("junk sample analyzed failed", "error", err)
		return
	}

	in := gemini.ReportInputFromStore(periodStart, periodEnd, total, stats, samples)
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
	}
	if err := s.junk.InsertReport(ctx, doc); err != nil {
		slog.Warn("junk report insert failed", "error", err)
		return
	}

	reportID := doc.ID.Hex()
	s.maybeWriteKeywordDiff(ctx, reportID, result, periodStart)

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
