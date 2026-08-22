package coldpath

import (
	"context"
	"log/slog"
	"time"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/worker"
)

// HighLeadLister returns recent High leads for duplicate suggest scan.
type HighLeadLister interface {
	ListHighLeadsSince(ctx context.Context, since time.Time, limit int) ([]sink.LeadDoc, error)
}

// DuplicateSuggestWriter patches duplicate_suggest on leads.
type DuplicateSuggestWriter interface {
	PatchDuplicateSuggest(ctx context.Context, hashID, suggestHash string) error
}

// DuplicateSuggestScanner finds semantic duplicate pairs among accepted High leads.
type DuplicateSuggestScanner struct {
	interval time.Duration
	window   time.Duration
	limit    int
	lister   HighLeadLister
	writer   DuplicateSuggestWriter
	cluster  *gemini.LeadCluster
}

func NewDuplicateSuggestScanner(interval, window time.Duration, limit int, lister HighLeadLister, writer DuplicateSuggestWriter, cluster *gemini.LeadCluster) *DuplicateSuggestScanner {
	interval = worker.DurationOr(interval, 24*time.Hour)
	if window <= 0 {
		window = 7 * 24 * time.Hour
	}
	if limit <= 0 {
		limit = 200
	}
	return &DuplicateSuggestScanner{
		interval: interval,
		window:   window,
		limit:    limit,
		lister:   lister,
		writer:   writer,
		cluster:  cluster,
	}
}

func (d *DuplicateSuggestScanner) Run(ctx context.Context) {
	if d == nil || d.lister == nil || d.writer == nil || d.cluster == nil {
		return
	}
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	d.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.runOnce(ctx)
		}
	}
}

func (d *DuplicateSuggestScanner) runOnce(ctx context.Context) {
	since := time.Now().UTC().Add(-d.window)
	docs, err := d.lister.ListHighLeadsSince(ctx, since, d.limit)
	if err != nil {
		slog.Warn("duplicate suggest list failed", "error", err)
		return
	}
	pairs := 0
	for _, doc := range docs {
		if doc.HashID == "" || doc.Snippet == "" || doc.DuplicateSuggest != "" {
			continue
		}
		dup, other, err := d.cluster.CheckDuplicate(ctx, doc.HashID, doc.Snippet)
		if err != nil {
			slog.Debug("duplicate suggest check failed", "hash_id", doc.HashID, "error", err)
			continue
		}
		if !dup || other == "" {
			_ = d.cluster.Record(ctx, doc.HashID, doc.Snippet)
			continue
		}
		if err := d.writer.PatchDuplicateSuggest(ctx, doc.HashID, other); err != nil {
			slog.Warn("duplicate suggest patch failed", "hash_id", doc.HashID, "error", err)
			continue
		}
		_ = d.writer.PatchDuplicateSuggest(ctx, other, doc.HashID)
		pairs++
	}
	if pairs > 0 {
		slog.Info("duplicate suggest pairs written", "count", pairs)
	}
}
