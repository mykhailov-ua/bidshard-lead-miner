package pipeline

import (
	"log/slog"

	"github.com/bidshard/parser/internal/metrics"
)

// TrySendStats sends round stats without blocking. On drop, increments parser_stats_dropped_total.
func TrySendStats(statsCh chan<- RoundStats, stats RoundStats) {
	select {
	case statsCh <- stats:
	default:
		metrics.RecordStatsDropped()
		slog.Warn("stats channel full, dropping round stats", "round_id", stats.RoundID)
	}
}
