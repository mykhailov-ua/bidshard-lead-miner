package pipeline

import (
	"testing"

	"github.com/bidshard/parser/internal/metrics"
)

func TestTrySendStatsDropRecordsMetric(t *testing.T) {
	before := metrics.StatsDroppedTotal()
	ch := make(chan RoundStats, 1)
	TrySendStats(ch, RoundStats{RoundID: "r1"})
	TrySendStats(ch, RoundStats{RoundID: "r2"})
	if got := metrics.StatsDroppedTotal(); got != before+1 {
		t.Fatalf("stats_dropped=%d want %d", got, before+1)
	}
	got := <-ch
	if got.RoundID != "r1" {
		t.Fatalf("round_id=%q want r1", got.RoundID)
	}
}
