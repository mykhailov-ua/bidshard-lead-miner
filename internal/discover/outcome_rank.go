package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

// OutcomeDorkRow correlates SERP dork accept/junk with downstream CRM outcomes.
type OutcomeDorkRow struct {
	Query            string  `json:"query"`
	Accepted         int     `json:"accepted"`
	Junk             int     `json:"junk"`
	AcceptRate       float64 `json:"accept_rate"`
	ChannelHits      int     `json:"channel_hits"`
	OutcomeContacted int64   `json:"outcome_contacted"`
	OutcomeReplied   int64   `json:"outcome_replied"`
	OutcomePilot     int64   `json:"outcome_pilot_started"`
	OutcomeMigration int64   `json:"outcome_migration_imported"`
}

// WriteOutcomeDorkReport merges parser source_stats with CRM outcome counts by dork.
func WriteOutcomeDorkReport(channelsPath, outDir string, stats []sink.SourceStatsDoc, outcomes []OutcomeSourceRow) (string, error) {
	if outDir == "" {
		outDir = "data/suggestions"
	}
	rows, err := BuildOutcomeDorkRows(channelsPath, stats, outcomes)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102")
	path := filepath.Join(outDir, "outcome_dork_rank_"+stamp+".json")
	raw, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// OutcomeSourceRow is a flattened source/outcome count from CRM leads.
type OutcomeSourceRow struct {
	Source  string
	Outcome string
	Count   int64
}
