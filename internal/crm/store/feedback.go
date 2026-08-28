package store

import (
	"context"
	"fmt"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/dorkdisable"
	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FeedbackReport is a read-only snapshot for CRM admin feedback dashboards.
type FeedbackReport struct {
	TopDorks     []discover.OutcomeDorkRow `json:"top_dorks"`
	TopSources   []SourceOutcomeSummary    `json:"top_sources"`
	KeywordTune  []discover.KeywordTuneRow `json:"keyword_tune,omitempty"`
	DisabledDork int                       `json:"disabled_dork_count,omitempty"`
}

// SourceOutcomeSummary rolls up CRM outcomes per crawl source family.
type SourceOutcomeSummary struct {
	Source             string `json:"source"`
	Accepted           int    `json:"accepted"`
	Junk               int    `json:"junk"`
	OutcomeContacted   int64  `json:"outcome_contacted"`
	OutcomeReplied     int64  `json:"outcome_replied"`
	OutcomePilot       int64  `json:"outcome_pilot_started"`
	OutcomeMigration   int64  `json:"outcome_migration_imported"`
}

// FeedbackReport builds outcome/dork/keyword feedback for admin API.
func (s *LeadStore) FeedbackReport(ctx context.Context, channelsPath string, disabledDorksPath string) (FeedbackReport, error) {
	if s == nil || s.leads == nil {
		return FeedbackReport{}, fmt.Errorf("lead store not initialized")
	}
	var report FeedbackReport

	outcomes, err := sink.AggregateOutcomeBySource(ctx, s.leads, s.statsTimeout)
	if err != nil {
		return report, err
	}
	outcomeRows := make([]discover.OutcomeSourceRow, 0, len(outcomes))
	for _, o := range outcomes {
		outcomeRows = append(outcomeRows, discover.OutcomeSourceRow{
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
			options.Find().SetSort(bson.D{{Key: "accepted", Value: -1}}).SetLimit(200),
		)
		if err != nil {
			return report, err
		}
		defer func() { _ = cur.Close(queryCtx) }()
		if err := cur.All(queryCtx, &stats); err != nil {
			return report, err
		}
	}

	dorkRows, err := discover.BuildOutcomeDorkRows(channelsPath, stats, outcomeRows)
	if err != nil {
		return report, err
	}
	if len(dorkRows) > 20 {
		dorkRows = dorkRows[:20]
	}
	report.TopDorks = dorkRows
	report.TopSources = buildSourceOutcomeSummaries(stats, outcomeRows)

	if s.keywordStats != nil {
		queryCtx, cancel := context.WithTimeout(ctx, s.statsTimeout)
		defer cancel()
		cur, err := s.keywordStats.Find(queryCtx, bson.M{}, options.Find().SetLimit(500))
		if err == nil {
			var docs []sink.KeywordStatDoc
			if err := cur.All(queryCtx, &docs); err == nil {
				report.KeywordTune = discover.BuildKeywordTuneRows(docs, nil)
				if len(report.KeywordTune) > 30 {
					report.KeywordTune = report.KeywordTune[:30]
				}
			}
			_ = cur.Close(queryCtx)
		}
	}

	if disabledDorksPath != "" {
		report.DisabledDork = dorkdisable.CountDisabled(disabledDorksPath)
	}
	return report, nil
}

func buildSourceOutcomeSummaries(stats []sink.SourceStatsDoc, outcomes []discover.OutcomeSourceRow) []SourceOutcomeSummary {
	bySource := map[string]*SourceOutcomeSummary{}
	for _, row := range stats {
		source := row.Source
		if source == "" {
			continue
		}
		entry := bySource[source]
		if entry == nil {
			entry = &SourceOutcomeSummary{Source: source}
			bySource[source] = entry
		}
		entry.Accepted += row.Accepted
		entry.Junk += row.Junk
		entry.OutcomeContacted += int64(row.OutcomeContacted)
		entry.OutcomeReplied += int64(row.OutcomeReplied)
		entry.OutcomePilot += int64(row.OutcomePilot)
		entry.OutcomeMigration += int64(row.OutcomeMigration)
	}
	for _, row := range outcomes {
		entry := bySource[row.Source]
		if entry == nil {
			entry = &SourceOutcomeSummary{Source: row.Source}
			bySource[row.Source] = entry
		}
		switch row.Outcome {
		case "contacted":
			entry.OutcomeContacted += row.Count
		case "replied":
			entry.OutcomeReplied += row.Count
		case "pilot_started":
			entry.OutcomePilot += row.Count
		case "migration_imported":
			entry.OutcomeMigration += row.Count
		}
	}
	out := make([]SourceOutcomeSummary, 0, len(bySource))
	for _, row := range bySource {
		out = append(out, *row)
	}
	sortSourceOutcomeSummaries(out)
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

func sortSourceOutcomeSummaries(rows []SourceOutcomeSummary) {
	if len(rows) < 2 {
		return
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if sourceOutcomeScore(rows[j]) > sourceOutcomeScore(rows[i]) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

func sourceOutcomeScore(row SourceOutcomeSummary) int64 {
	return row.OutcomePilot*100 + row.OutcomeMigration*80 + row.OutcomeReplied*20 + row.OutcomeContacted*5 + int64(row.Accepted)
}
