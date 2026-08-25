package suggestions

import (
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/gemini"
)

// KeywordAutoApplyOptions mirrors discover auto-apply weekly cap state.
type KeywordAutoApplyOptions struct {
	MaxPerWeek int
	StatePath  string
	Now        time.Time
}

var keywordPhraseDenylist = []string{
	// Job boards and social surfaces are never pain keywords for this ICP.
	"linkedin",
	"glassdoor",
	"indeed",
	"ziprecruiter",
	"monster.com",
	"facebook.com",
}

// FilterKeywordDiff applies the same denylist guardrails as discover auto-apply.
func FilterKeywordDiff(diff gemini.KeywordDiff) (gemini.KeywordDiff, map[string]string) {
	blocked := map[string]string{}
	var keepKW, keepHR []gemini.KeywordEntry
	for _, e := range diff.AddKeywords {
		if reason := blockedKeywordPhrase(e.Phrase); reason != "" {
			blocked[e.Phrase] = reason
			continue
		}
		keepKW = append(keepKW, e)
	}
	for _, e := range diff.AddHardReject {
		if reason := blockedKeywordPhrase(e.Phrase); reason != "" {
			blocked[e.Phrase] = reason
			continue
		}
		keepHR = append(keepHR, e)
	}
	diff.AddKeywords = keepKW
	diff.AddHardReject = keepHR
	return diff, blocked
}

func blockedKeywordPhrase(phrase string) string {
	lower := strings.ToLower(strings.TrimSpace(phrase))
	if len(lower) < 3 {
		return "too_short"
	}
	for _, pat := range keywordPhraseDenylist {
		if strings.Contains(lower, pat) {
			return pat
		}
	}
	return ""
}

func ApplyKeywordsAuto(pendingPath, keywordsPath string, opts KeywordAutoApplyOptions, dryRun bool) (string, error) {
	diff, base, err := loadKeywordPending(pendingPath, keywordsPath)
	if err != nil {
		return "", err
	}
	diff, blocked := FilterKeywordDiff(diff)
	merged, addedKW, addedHR := mergeKeywords(base, diff)
	totalNew := addedKW + addedHR
	if totalNew == 0 {
		return "", fmt.Errorf("auto-apply: no new keywords")
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	maxWeek := opts.MaxPerWeek
	if maxWeek <= 0 {
		maxWeek = discover.DefaultAutoApplyMaxPerWeek
	}
	statePath := opts.StatePath
	if statePath == "" {
		statePath = discover.DefaultAutoApplyStatePath
	}
	state := discover.LoadAutoApplyState(statePath, now)
	quota := discover.RemainingWeeklyQuota(state, now, maxWeek)
	if totalNew > quota {
		return "", fmt.Errorf("auto-apply: weekly cap exceeded (new=%d quota=%d)", totalNew, quota)
	}
	summary := fmt.Sprintf("add_keywords=%d add_hard_reject=%d", addedKW, addedHR)
	if len(blocked) > 0 {
		summary += fmt.Sprintf(" blocked=%d", len(blocked))
	}
	if dryRun {
		return summary + " (dry-run)", nil
	}
	if err := backupFile(keywordsPath); err != nil {
		return "", err
	}
	if err := writeJSON(keywordsPath, merged); err != nil {
		return "", err
	}
	state.Applied += addedKW + addedHR
	if err := discover.SaveAutoApplyState(statePath, state); err != nil {
		return "", err
	}
	diff.Status = "applied"
	if err := writeJSON(pendingPath, diff); err != nil {
		return "", err
	}
	return summary, nil
}
