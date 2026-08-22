package coldpath

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/worker"
)

// WebhookAuditReporter writes monthly webhook spam feedback summaries.
type WebhookAuditReporter struct {
	interval time.Duration
	feedback *sink.WebhookFeedbackStore
	outDir   string
}

func NewWebhookAuditReporter(interval time.Duration, feedback *sink.WebhookFeedbackStore, outDir string) *WebhookAuditReporter {
	interval = worker.DurationOr(interval, 30*24*time.Hour)
	if outDir == "" {
		outDir = "data/suggestions"
	}
	return &WebhookAuditReporter{interval: interval, feedback: feedback, outDir: outDir}
}

func (r *WebhookAuditReporter) Run(ctx context.Context) {
	if r == nil || r.feedback == nil {
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

func (r *WebhookAuditReporter) runOnce(ctx context.Context) {
	since := time.Now().UTC().Add(-30 * 24 * time.Hour)
	rows, err := r.feedback.ListSince(ctx, since, 200)
	if err != nil {
		slog.Warn("webhook audit list failed", "error", err)
		return
	}
	if len(rows) == 0 {
		return
	}
	bySource := map[string]int{}
	for _, row := range rows {
		bySource[row.Source]++
	}
	report := map[string]any{
		"ts":        time.Now().UTC().Format(time.RFC3339),
		"count":     len(rows),
		"by_source": bySource,
	}
	if err := os.MkdirAll(r.outDir, 0o755); err != nil {
		slog.Warn("webhook audit mkdir failed", "error", err)
		return
	}
	path := filepath.Join(r.outDir, "webhook_audit_"+time.Now().UTC().Format("20060102")+".json")
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		slog.Warn("webhook audit write failed", "error", err)
		return
	}
	slog.Info("webhook audit report written", "path", path, "count", len(rows))
}
