package app

import (
	"context"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/worker"
)

// startPendingAnalysisMetrics polls Mongo for analysis_status=pending gauge.
func startPendingAnalysisMetrics(ctx context.Context, wg *sync.WaitGroup, store sink.StalePendingLister, interval time.Duration) {
	if store == nil {
		return
	}
	interval = worker.DurationOr(interval, 2*time.Minute)
	worker.Run(ctx, wg, func(ctx context.Context) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		poll := func() {
			n, err := store.CountPendingAnalysis(ctx)
			if err != nil {
				return
			}
			metrics.SetLeadsAnalysisPending(n)
		}
		poll()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				poll()
			}
		}
	})
}
