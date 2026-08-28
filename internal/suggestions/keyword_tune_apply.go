package suggestions

import (
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/gemini"
)

// KeywordTuneAutoApplyOptions limits weekly in-place keyword registry edits.
type KeywordTuneAutoApplyOptions struct {
	MaxPerWeek int
	StatePath  string
	Now        time.Time
}

// ApplyKeywordTune patches keywords.json weights and removes stats-disabled phrases.
func ApplyKeywordTune(keywordsPath string, rows []discover.KeywordTuneRow, dryRun bool) (string, error) {
	if keywordsPath == "" {
		return "", fmt.Errorf("keywords path empty")
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("keyword tune: no rows")
	}
	var base keywordsFile
	if err := readJSON(keywordsPath, &base); err != nil {
		return "", err
	}
	merged, disabled, updated := applyKeywordTuneRows(base, rows)
	summary := fmt.Sprintf("weight_updates=%d disabled=%d", updated, disabled)
	if dryRun {
		return summary + " (dry-run)", nil
	}
	if updated == 0 && disabled == 0 {
		return "", fmt.Errorf("keyword tune: nothing to change")
	}
	if err := backupFile(keywordsPath); err != nil {
		return "", err
	}
	if err := writeJSON(keywordsPath, merged); err != nil {
		return "", err
	}
	return summary, nil
}

// ApplyKeywordTuneAuto applies tune rows with weekly quota (shared state file with discover auto-apply).
func ApplyKeywordTuneAuto(keywordsPath string, rows []discover.KeywordTuneRow, opts KeywordTuneAutoApplyOptions, dryRun bool) (string, error) {
	if len(rows) == 0 {
		return "", fmt.Errorf("keyword tune auto: no rows")
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	maxWeek := opts.MaxPerWeek
	if maxWeek <= 0 {
		maxWeek = 15
	}
	statePath := opts.StatePath
	if statePath == "" {
		statePath = discover.DefaultAutoApplyStatePath
	}
	state := discover.LoadAutoApplyState(statePath, now)
	quota := discover.RemainingKeywordTuneQuota(state, now, maxWeek)

	var eligible []discover.KeywordTuneRow
	changes := 0
	for _, row := range rows {
		if changes >= quota {
			break
		}
		if row.RecommendDisable {
			eligible = append(eligible, row)
			changes++
			continue
		}
		if row.SuggestedWeight > 0 {
			eligible = append(eligible, row)
			changes++
		}
	}
	if len(eligible) == 0 {
		return "", fmt.Errorf("keyword tune auto: weekly cap reached or no eligible rows (quota=%d)", quota)
	}
	summary, err := ApplyKeywordTune(keywordsPath, eligible, dryRun)
	if err != nil {
		return "", err
	}
	if dryRun {
		return summary + fmt.Sprintf(" eligible=%d quota=%d", len(eligible), quota), nil
	}
	state.KeywordTuneApplied += len(eligible)
	if err := discover.SaveAutoApplyState(statePath, state); err != nil {
		return "", err
	}
	return summary + fmt.Sprintf(" eligible=%d", len(eligible)), nil
}

func applyKeywordTuneRows(base keywordsFile, rows []discover.KeywordTuneRow) (keywordsFile, int, int) {
	byID := map[string]discover.KeywordTuneRow{}
	for _, row := range rows {
		id := strings.ToLower(strings.TrimSpace(row.KeywordID))
		if id != "" {
			byID[id] = row
		}
	}
	disabled := 0
	updated := 0
	var kept []gemini.KeywordEntry
	for _, entry := range base.Keywords {
		id := strings.ToLower(strings.TrimSpace(entry.ID))
		row, ok := byID[id]
		if !ok {
			kept = append(kept, entry)
			continue
		}
		if row.RecommendDisable {
			disabled++
			continue
		}
		if row.SuggestedWeight > 0 && entry.Weight != row.SuggestedWeight {
			entry.Weight = row.SuggestedWeight
			updated++
		}
		kept = append(kept, entry)
	}
	base.Keywords = kept
	return base, disabled, updated
}
