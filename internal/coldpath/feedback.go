package coldpath

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

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
	recent, err := s.embed.RecentVectors(ctx, 200)
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
	_ = s.embed.Upsert(ctx, sink.EmbeddingDoc{Key: key, Vector: vec, TS: time.Now().UTC()})
}

func (s *Service) maybeWriteKeywordDiff(ctx context.Context, reportID string, result gemini.ReportResult, periodStart time.Time) {
	if s.keywordDiffEvery <= 0 || s.keywordDiffDir == "" || s.gemini == nil {
		return
	}
	s.reportCount++
	if s.reportCount%s.keywordDiffEvery != 0 {
		return
	}
	samples, err := s.junk.FindByCategorySince(ctx, "false_negative", periodStart, 15)
	if err != nil {
		slog.Warn("false_negative sample fetch failed", "error", err)
	}
	snippets := make([]string, 0, len(samples))
	for _, doc := range samples {
		snippets = append(snippets, doc.Snippet)
	}
	diff, err := s.gemini.BuildKeywordDiff(ctx, result.KeywordSuggestions, snippets)
	if err != nil {
		slog.Warn("keyword diff build failed", "error", err)
		return
	}
	diff = enrichKeywordDiffWithStats(ctx, s.keywordStats, diff)
	diff.ReportID = reportID
	diff.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(s.keywordDiffDir, 0o755); err != nil {
		slog.Warn("keyword diff mkdir failed", "error", err)
		return
	}
	path := filepath.Join(s.keywordDiffDir, "keywords_pending_"+reportID+".json")
	raw, _ := json.MarshalIndent(diff, "", "  ")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		slog.Warn("keyword diff write failed", "path", path, "error", err)
		return
	}
	slog.Info("keyword diff pending approve", "path", path, "add_keywords", len(diff.AddKeywords), "add_hard_reject", len(diff.AddHardReject))
}

func snippetKey(snippet string) string {
	sum := sha256.Sum256([]byte(snippet))
	return hex.EncodeToString(sum[:16])
}
