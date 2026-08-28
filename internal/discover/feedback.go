package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

// DorkPruneConfig controls auto-disable rules for underperforming SERP dorks.
type DorkPruneConfig struct {
	MinRaw            int
	MaxAcceptRate     float64
	RequireZeroOutcomes bool
}

// DefaultDorkPruneConfig is the conservative prod default for dork governor.
func DefaultDorkPruneConfig() DorkPruneConfig {
	return DorkPruneConfig{
		MinRaw:              30,
		MaxAcceptRate:       0.05,
		RequireZeroOutcomes: true,
	}
}

// EvaluateDorkPrune returns dork queries to disable from outcome-ranked rows.
func EvaluateDorkPrune(rows []OutcomeDorkRow, cfg DorkPruneConfig) []string {
	if cfg.MinRaw <= 0 {
		cfg.MinRaw = 30
	}
	if cfg.MaxAcceptRate <= 0 {
		cfg.MaxAcceptRate = 0.05
	}
	var out []string
	for _, row := range rows {
		q := strings.TrimSpace(row.Query)
		if q == "" {
			continue
		}
		total := row.Accepted + row.Junk
		if total < cfg.MinRaw {
			continue
		}
		if row.AcceptRate > cfg.MaxAcceptRate {
			continue
		}
		if cfg.RequireZeroOutcomes && row.TotalOutcomes() > 0 {
			continue
		}
		out = append(out, q)
	}
	return out
}

// TotalOutcomes sums downstream CRM outcome counters on a dork row.
func (r OutcomeDorkRow) TotalOutcomes() int64 {
	return r.OutcomeContacted + r.OutcomeReplied + r.OutcomePilot + r.OutcomeMigration
}

// KeywordTuneRow is a stats-driven keyword weight/disable recommendation.
type KeywordTuneRow struct {
	KeywordID        string  `json:"keyword_id"`
	Accepted         int     `json:"accepted"`
	Junk             int     `json:"junk"`
	JunkRate         float64 `json:"junk_rate"`
	SuggestedWeight  int     `json:"suggested_weight,omitempty"`
	RecommendDisable bool    `json:"recommend_disable"`
}

// BuildKeywordTuneRows turns Mongo keyword_stats into manual-review tune rows.
func BuildKeywordTuneRows(docs []sink.KeywordStatDoc, currentWeights map[string]int) []KeywordTuneRow {
	rows := make([]KeywordTuneRow, 0, len(docs))
	for _, doc := range docs {
		if doc.KeywordID == "" {
			continue
		}
		total := doc.TotalSamples()
		if total < 10 {
			continue
		}
		weight := currentWeights[doc.KeywordID]
		if weight <= 0 {
			weight = 20
		}
		suggested, enabled := doc.Recommendation(weight)
		row := KeywordTuneRow{
			KeywordID:       doc.KeywordID,
			Accepted:        doc.AcceptedCount,
			Junk:            doc.JunkCount,
			JunkRate:        doc.JunkRate(),
			SuggestedWeight: suggested,
		}
		if !enabled {
			row.RecommendDisable = true
		}
		if row.RecommendDisable || row.SuggestedWeight != weight {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].JunkRate == rows[j].JunkRate {
			return rows[i].KeywordID < rows[j].KeywordID
		}
		return rows[i].JunkRate > rows[j].JunkRate
	})
	return rows
}

// WriteKeywordTuneReport writes keyword_tune_YYYYMMDD.json under outDir.
func WriteKeywordTuneReport(outDir string, rows []KeywordTuneRow) (string, error) {
	if outDir == "" {
		outDir = "data/suggestions"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102")
	path := filepath.Join(outDir, "keyword_tune_"+stamp+".json")
	raw, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// BuildOutcomeDorkRows merges source_stats and CRM outcomes without writing a file.
func BuildOutcomeDorkRows(channelsPath string, stats []sink.SourceStatsDoc, outcomes []OutcomeSourceRow) ([]OutcomeDorkRow, error) {
	idx, err := loadDorkIndex(channelsPath)
	if err != nil {
		idx = dorkIndex{
			usernameToQuery: map[string]string{},
			hitsPerQuery:    map[string]int{},
		}
	}
	byDork := map[string]*OutcomeDorkRow{}
	for _, row := range stats {
		source := strings.TrimSpace(row.Source)
		if source == "" {
			continue
		}
		q := matchTelegramDork(source, idx.usernameToQuery)
		if q == "" {
			continue
		}
		entry := byDork[q]
		if entry == nil {
			entry = &OutcomeDorkRow{Query: q}
			byDork[q] = entry
		}
		entry.Accepted += row.Accepted
		entry.Junk += row.Junk
	}
	for _, row := range outcomes {
		q := matchTelegramDork(row.Source, idx.usernameToQuery)
		if q == "" {
			continue
		}
		entry := byDork[q]
		if entry == nil {
			entry = &OutcomeDorkRow{Query: q}
			byDork[q] = entry
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
	rows := make([]OutcomeDorkRow, 0, len(byDork))
	for q, row := range byDork {
		total := row.Accepted + row.Junk
		if total > 0 {
			row.AcceptRate = float64(row.Accepted) / float64(total)
		}
		row.ChannelHits = idx.hitsPerQuery[q]
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].OutcomePilot != rows[j].OutcomePilot {
			return rows[i].OutcomePilot > rows[j].OutcomePilot
		}
		if rows[i].AcceptRate == rows[j].AcceptRate {
			return rows[i].Accepted > rows[j].Accepted
		}
		return rows[i].AcceptRate > rows[j].AcceptRate
	})
	return rows, nil
}
