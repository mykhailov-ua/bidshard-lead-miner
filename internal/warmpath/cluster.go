package warmpath

import (
	"context"
	"log/slog"

	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
)

func (s *Service) filterWarmCluster(ctx context.Context, batch []Event) []Event {
	if s == nil || s.cluster == nil || len(batch) == 0 {
		return batch
	}
	out := make([]Event, 0, len(batch))
	for _, ev := range batch {
		if !scoring.MeetsMinPriority(scoring.Priority(ev.Priority), scoring.PriorityMedium) {
			out = append(out, ev)
			continue
		}
		dup, clusterOf, err := s.cluster.CheckDuplicate(ctx, ev.HashID, ev.Snippet)
		if err != nil {
			slog.Debug("warm embed cluster check failed", "hash_id", ev.HashID, "error", err)
			out = append(out, ev)
			continue
		}
		if !dup {
			out = append(out, ev)
			continue
		}
		s.rejectWarmClusterDuplicate(ctx, ev, clusterOf)
	}
	return out
}

func (s *Service) rejectWarmClusterDuplicate(ctx context.Context, ev Event, clusterOf string) {
	if err := s.patcher.PatchLeadAnalysis(ctx, sink.LeadAnalysisPatch{
		HashID:         ev.HashID,
		AnalysisStatus: "duplicate",
		Status:         "duplicate",
		DuplicateOf:    clusterOf,
	}); err != nil {
		slog.Warn("warm cluster duplicate patch failed", "hash_id", ev.HashID, "error", err)
	}
	slog.Info("warm embed cluster duplicate", "hash_id", ev.HashID, "duplicate_of", clusterOf)
}
