package store

import (
	"context"
	"fmt"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// WriteOutcomeDorkReport writes outcome_dork_rank_YYYYMMDD.json under outDir.
func (s *LeadStore) WriteOutcomeDorkReport(ctx context.Context, channelsPath, outDir string) (string, error) {
	if s == nil || s.leads == nil {
		return "", fmt.Errorf("lead store not initialized")
	}
	outcomes, err := s.AggregateOutcomeBySource(ctx)
	if err != nil {
		return "", err
	}
	rows := make([]discover.OutcomeSourceRow, 0, len(outcomes))
	for _, o := range outcomes {
		rows = append(rows, discover.OutcomeSourceRow{
			Source:  o.Source,
			Outcome: o.Outcome,
			Count:   o.Count,
		})
	}
	var stats []sink.SourceStatsDoc
	if s.sourceStats != nil {
		queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
		defer cancel()
		cur, err := s.sourceStats.Find(queryCtx, bson.M{},
			options.Find().SetSort(bson.D{{Key: "accepted", Value: -1}}),
		)
		if err != nil {
			return "", err
		}
		defer func() { _ = cur.Close(queryCtx) }()
		if err := cur.All(queryCtx, &stats); err != nil {
			return "", err
		}
	}
	return discover.WriteOutcomeDorkReport(channelsPath, outDir, stats, rows)
}
