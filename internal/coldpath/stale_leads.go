package coldpath

import (
	"context"
	"log/slog"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/worker"
)

// ICPClassifier runs lite ICP classification for stale lead re-grade.
type ICPClassifier interface {
	ClassifyICP(ctx context.Context, text string) (gemini.ICPResult, error)
}

// StaleLeadLister loads old status=new leads for ICP re-grade.
type StaleLeadLister interface {
	ListStaleNewLeads(ctx context.Context, olderThan time.Duration, limit int) ([]sink.LeadDoc, error)
}

// StaleLeadPatcher updates re-graded lead fields.
type StaleLeadPatcher interface {
	PatchLeadAnalysis(ctx context.Context, patch sink.LeadAnalysisPatch) error
}

// StaleLeadRegrader re-runs ICP on old new leads (weekly cold job).
type StaleLeadRegrader struct {
	interval time.Duration
	age      time.Duration
	limit    int
	lister   StaleLeadLister
	patcher  StaleLeadPatcher
	icp      ICPClassifier
	registry *scoring.Registry
}

func NewStaleLeadRegrader(interval, age time.Duration, limit int, lister StaleLeadLister, patcher StaleLeadPatcher, icp ICPClassifier, registry *scoring.Registry) *StaleLeadRegrader {
	interval = worker.DurationOr(interval, 7*24*time.Hour)
	if age <= 0 {
		age = 30 * 24 * time.Hour
	}
	if limit <= 0 {
		limit = 50
	}
	return &StaleLeadRegrader{
		interval: interval,
		age:      age,
		limit:    limit,
		lister:   lister,
		patcher:  patcher,
		icp:      icp,
		registry: registry,
	}
}

func (r *StaleLeadRegrader) Run(ctx context.Context) {
	if r == nil || r.lister == nil || r.patcher == nil || r.icp == nil {
		return
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	r.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runOnce(ctx)
		}
	}
}

func (r *StaleLeadRegrader) runOnce(ctx context.Context) {
	docs, err := r.lister.ListStaleNewLeads(ctx, r.age, r.limit)
	if err != nil {
		slog.Warn("stale lead list failed", "error", err)
		return
	}
	if len(docs) == 0 {
		return
	}
	highMin := 0
	if r.registry != nil {
		_, _, _, highMin, _ = r.registry.Snapshot()
	}
	updated := 0
	for _, doc := range docs {
		if doc.HashID == "" || doc.Snippet == "" {
			continue
		}
		res, err := r.icp.ClassifyICP(ctx, doc.Snippet)
		if err != nil {
			slog.Debug("stale lead icp failed", "hash_id", doc.HashID, "error", err)
			continue
		}
		score := doc.Score
		priority := scoring.Priority(doc.Priority)
		if r.registry != nil {
			score, _ = gemini.ApplyICPToScore(score, res, highMin)
			priority = scoring.PriorityFromScore(r.registry, score)
		}
		stale := res.ICP == "none" && !res.Hot && priority == scoring.PriorityLow
		if err := r.patcher.PatchLeadAnalysis(ctx, sink.LeadAnalysisPatch{
			HashID:   doc.HashID,
			Score:    score,
			Priority: string(priority),
			ICP:      res.ICP,
			Hot:      res.Hot,
			ICPWhy:   res.Why,
			Stale:    stale,
		}); err != nil {
			slog.Warn("stale lead patch failed", "hash_id", doc.HashID, "error", err)
			continue
		}
		updated++
	}
	if updated > 0 {
		slog.Info("stale lead re-grade complete", "count", updated)
	}
}

// RunStaleRegradeOnce runs one stale pass (CLI/tests).
func (r *StaleLeadRegrader) RunStaleRegradeOnce(ctx context.Context) {
	if r != nil {
		r.runOnce(ctx)
	}
}
