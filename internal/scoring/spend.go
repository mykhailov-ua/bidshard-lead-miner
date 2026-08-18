package scoring

import (
	"regexp"
	"strings"
)

const spendBoost = 15

var spendPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\$\s*\d+\s*k(?:\s*/\s*(?:mo|month))?`),
	regexp.MustCompile(`(?i)\$\d{2,3}(?:,\d{3})+\s*/\s*(?:mo|month)`),
	regexp.MustCompile(`(?i)\b\d+\s*k\s*/\s*(?:mo|month)\b`),
	regexp.MustCompile(`(?i)\b(?:spend|spending|budget)\s+(?:is\s+)?\$?\d`),
	regexp.MustCompile(`(?i)\bour buyers\b`),
	regexp.MustCompile(`(?i)\bwe run \d+ campaigns\b`),
	regexp.MustCompile(`(?i)\bmedia buying team\b`),
	regexp.MustCompile(`(?i)\bteam of buyers\b`),
	regexp.MustCompile(`(?i)\$\d+k\+?\s*(?:spend|on traffic)`),
	regexp.MustCompile(`(?i)\b\d{1,3}\s*k\s*/\s*(?:mo|month)\b`),
	regexp.MustCompile(`(?i)\$\d{1,3}k\s*/\s*(?:mo|month)`),
	regexp.MustCompile(`(?i)\bwe run \d+\+?\s*campaigns?\b`),
	regexp.MustCompile(`(?i)\b\d+k\+?\s*(?:spend|budget)\b`),
}

var competitorPhrases = []string{
	"voluum", "keitaro", "binom", "redtrack", "bemob", "clickflare",
	"thrivetracker", "peerclick", "peer click", "voluumtrk",
}

// ApplySpendGate boosts spend signals and caps score when no spend/competitor proof.
func ApplySpendGate(score int, text string, mediumMin int) int {
	body := strings.ToLower(text)
	if HasSpendSignal(body) {
		score += spendBoost
	}
	if !HasSpendSignal(body) && !HasCompetitorMention(body) {
		cap := mediumMin - 1
		if cap < 0 {
			cap = 0
		}
		if score > cap {
			score = cap
		}
	}
	return score
}

func HasSpendSignal(text string) bool {
	for _, re := range spendPatterns {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

func HasCompetitorMention(text string) bool {
	for _, phrase := range competitorPhrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

func PriorityFromScore(reg *Registry, score int) Priority {
	_, _, _, highMin, mediumMin := reg.Snapshot()
	switch {
	case score >= highMin:
		return PriorityHigh
	case score >= mediumMin:
		return PriorityMedium
	default:
		return PriorityLow
	}
}
