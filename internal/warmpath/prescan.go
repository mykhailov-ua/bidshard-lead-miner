package warmpath

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/coldpath"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
)

func (s *Service) filterWarmPrescan(ctx context.Context, batch []Event) []Event {
	if s == nil || s.prescan == nil || len(batch) == 0 {
		return batch
	}
	out := make([]Event, 0, len(batch))
	for _, ev := range batch {
		if !scoring.MeetsMinPriority(scoring.Priority(ev.Priority), scoring.PriorityMedium) {
			out = append(out, ev)
			continue
		}
		text := strings.TrimSpace(ev.Snippet)
		if text == "" {
			out = append(out, ev)
			continue
		}
		verdict, err := s.prescan.EvaluateSpam(ctx, text)
		if err != nil {
			slog.Debug("warm embed prescan failed", "hash_id", ev.HashID, "error", err)
			out = append(out, ev)
			continue
		}
		if !verdict.SpamMatch {
			out = append(out, ev)
			continue
		}
		s.rejectWarmPrescanSpam(ctx, ev, verdict.SpamScore)
	}
	return out
}

func (s *Service) rejectWarmPrescanSpam(ctx context.Context, ev Event, spamScore float64) {
	detail := "embed spam match on warm path"
	if err := s.patcher.PatchLeadAnalysis(ctx, sink.LeadAnalysisPatch{
		HashID:         ev.HashID,
		AnalysisStatus: coldpath.ReasonWarmPrescanSpam,
		Status:         "junk",
	}); err != nil {
		slog.Warn("warm prescan spam patch failed", "hash_id", ev.HashID, "error", err)
	}
	if s.junk != nil {
		doc := sink.JunkDoc{
			TS:           time.Now().UTC(),
			RoundID:      ev.RoundID,
			Source:       ev.Source,
			Title:        ev.Title,
			Snippet:      ev.Snippet,
			Reason:       coldpath.ReasonWarmPrescanSpam,
			ReasonDetail: detail,
			Score:        ev.Score,
			Matched:      append([]string(nil), ev.Matched...),
		}
		if err := s.junk.InsertMany(ctx, []sink.JunkDoc{doc}); err != nil {
			slog.Warn("warm prescan junk insert failed", "hash_id", ev.HashID, "error", err)
		}
	}
	slog.Info("warm prescan spam reject", "hash_id", ev.HashID, "spam_score", spamScore)
}
