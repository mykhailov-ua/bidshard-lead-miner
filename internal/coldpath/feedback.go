package coldpath

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
)

func (s *Service) processFalseNegative(ctx context.Context, doc sink.JunkDoc, result gemini.AnalyzeResult) {
	if s.crm == nil || result.Category != "false_negative" {
		return
	}
	err := s.crm.Insert(ctx, sink.CrmBoostDoc{
		TS:          time.Now().UTC(),
		JunkID:      doc.ID.Hex(),
		Source:      doc.Source,
		Snippet:     doc.Snippet,
		ContactHint: doc.ContactHint,
		Why:         result.Why,
		Priority:    "High",
		Status:      "pending",
	})
	if err != nil {
		slog.Warn("crm boost insert failed", "junk_id", doc.ID.Hex(), "error", err)
	}
}

func (s *Service) processSemanticDedup(ctx context.Context, doc sink.JunkDoc) {
	if s.embed == nil || s.gemini == nil {
		return
	}
	vec, err := s.gemini.EmbedText(ctx, doc.Snippet)
	if err != nil {
		slog.Debug("embed failed", "junk_id", doc.ID.Hex(), "error", err)
		return
	}
	key := snippetKey(doc.Snippet)
	// Compare against last 200 junk vectors before storing; 0.92 cosine marks semantic duplicate.
	recent, err := s.embed.RecentVectorsByKind(ctx, sink.EmbedKindJunk, 200)
	if err != nil {
		return
	}
	for _, other := range recent {
		if other.Key == key {
			continue
		}
		if gemini.CosineSimilarity(vec, other.Vector) >= s.embedThreshold {
			_ = s.junk.MarkSemanticDup(ctx, doc.ID, other.Key)
			slog.Info("semantic dup junk", "junk_id", doc.ID.Hex(), "dup_of", other.Key)
			return
		}
	}
	_ = s.embed.Upsert(ctx, sink.EmbeddingDoc{
		Key:    key,
		Vector: vec,
		Kind:   sink.EmbedKindJunk,
		TS:     time.Now().UTC(),
	})
}

func (s *Service) maybeWriteKeywordDiff(ctx context.Context, reportID string, result gemini.ReportResult, periodStart time.Time) {
	if s.keywordDiffEvery <= 0 || s.keywordDiffDir == "" || s.gemini == nil {
		return
	}
	s.reportCount++
	if s.reportCount%s.keywordDiffEvery != 0 {
		return
	}
	snippets, err := s.falseNegativeSnippets(ctx, periodStart)
	if err != nil {
		return
	}
	diff, err := s.gemini.BuildKeywordDiff(ctx, result.KeywordSuggestions, snippets)
	if err != nil {
		slog.Warn("keyword diff build failed", "error", err)
		return
	}
	diff = enrichKeywordDiffWithStats(ctx, s.keywordStats, diff)
	diff.ReportID = reportID
	diff.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	path, err := discover.WritePendingJSON(s.keywordDiffDir, "keywords_pending", reportID, diff)
	if err != nil {
		slog.Warn("keyword diff write failed", "error", err)
		return
	}
	slog.Info("keyword diff pending approve", "path", path, "add_keywords", len(diff.AddKeywords), "add_hard_reject", len(diff.AddHardReject))
}

func (s *Service) maybeWriteDiscoverDiff(ctx context.Context, reportID string, result gemini.ReportResult, periodStart time.Time) {
	if s.discoverDiffEvery <= 0 || s.discoverDiffDir == "" || s.gemini == nil {
		return
	}
	// Shares reportCount with keyword diff; cadence tied to cold-path report loop unless configured separately.
	if s.reportCount%s.discoverDiffEvery != 0 {
		return
	}

	icpPath := discover.ResolveICPPath("")
	current, err := discover.LoadICP(icpPath)
	if err != nil {
		slog.Warn("discover icp load failed", "path", icpPath, "error", err)
		current = discover.ICPConfig{}
	}

	snippets, err := s.falseNegativeSnippets(ctx, periodStart)
	if err != nil {
		return
	}

	diff, err := s.gemini.BuildDiscoverICPDiff(ctx, current, result.KeywordSuggestions, snippets)
	if err != nil {
		slog.Warn("discover icp diff build failed", "error", err)
		return
	}

	path, err := discover.WritePending(s.discoverDiffDir, reportID, discover.PendingICPDiff{
		AddTelegramSearch: diff.AddTelegramSearch,
		AddSerpDorks:      diff.AddSerpDorks,
		Summary:           diff.Summary,
		ReportID:          reportID,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		Status:            "pending",
	})
	if err != nil {
		slog.Warn("discover icp diff write failed", "error", err)
		return
	}
	slog.Info("discover icp diff pending approve",
		"path", path,
		"add_telegram_search", len(diff.AddTelegramSearch),
		"add_serp_dorks", len(diff.AddSerpDorks),
	)
}

func (s *Service) maybeWritePainVocabDiff(ctx context.Context, reportID string, periodStart time.Time) {
	if s.painVocabDiffEvery <= 0 || s.keywordDiffDir == "" || s.gemini == nil {
		return
	}
	if s.reportCount%s.painVocabDiffEvery != 0 {
		return
	}
	var pains []string
	if s.entityPains != nil {
		var err error
		pains, err = s.entityPains.ListUnifiedPainSamples(ctx, 20)
		if err != nil {
			slog.Warn("unified_pain sample fetch failed", "error", err)
		}
	}
	snippets, err := s.falseNegativeSnippets(ctx, periodStart)
	if err != nil {
		return
	}
	if len(pains) == 0 && len(snippets) == 0 {
		return
	}
	diff, err := s.gemini.BuildPainVocabDiff(ctx, pains, snippets)
	if err != nil {
		slog.Warn("pain vocab diff build failed", "error", err)
		return
	}
	diff = enrichKeywordDiffWithStats(ctx, s.keywordStats, diff)
	diff.ReportID = reportID
	diff.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	path, err := discover.WritePendingJSON(s.keywordDiffDir, "pain_vocab_pending", reportID, diff)
	if err != nil {
		slog.Warn("pain vocab diff write failed", "error", err)
		return
	}
	slog.Info("pain vocab diff pending approve", "path", path, "add_keywords", len(diff.AddKeywords), "add_hard_reject", len(diff.AddHardReject))
}

func (s *Service) falseNegativeSnippets(ctx context.Context, periodStart time.Time) ([]string, error) {
	samples, err := s.junk.FindByCategorySince(ctx, "false_negative", periodStart, 15)
	if err != nil {
		slog.Warn("false_negative sample fetch failed", "error", err)
		return nil, err
	}
	snippets := make([]string, 0, len(samples))
	for _, doc := range samples {
		snippets = append(snippets, doc.Snippet)
	}
	return snippets, nil
}

func snippetKey(snippet string) string {
	sum := sha256.Sum256([]byte(snippet))
	return hex.EncodeToString(sum[:16])
}
