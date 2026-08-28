package scoring

import (
	"regexp"
	"strings"
)

// DisplacementTier marks migration/switch intent strength for scoring and outreach sort.
type DisplacementTier string

const (
	DisplacementNone DisplacementTier = ""
	DisplacementWarm DisplacementTier = "warm"
	DisplacementHot  DisplacementTier = "hot"
)

const (
	displacementWarmBoost = 12
	displacementHotBoost  = 22
)

var displacementMigrRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:switching|switched|migrate|migrating|moving)\s+(?:from|off|away from|to)\b`),
	regexp.MustCompile(`(?i)\b(?:looking for|need|searching for)\s+(?:an?\s+)?alternative\s+to\b`),
	regexp.MustCompile(`(?i)\breplac(?:e|ing)\s+(?:our\s+)?(?:current\s+)?(?:tracker|stack|voluum|keitaro|binom)\b`),
	regexp.MustCompile(`(?i)\bmove(?:d|ing)?\s+(?:off|away from)\s+(?:voluum|keitaro|binom|redtrack)\b`),
}

// DetectDisplacementTier scores explicit migration language; requires buyer-intent signal.
func DetectDisplacementTier(text string, stack []string) (DisplacementTier, int) {
	lower := strings.ToLower(text)
	hasMigr := false
	for _, re := range displacementMigrRe {
		if re.MatchString(lower) {
			hasMigr = true
			break
		}
	}
	if !hasMigr || !HasBuyerIntentSignal(text) {
		return DisplacementNone, 0
	}
	hasComp := len(stack) > 0 || HasCompetitorMention(lower)
	if hasComp {
		return DisplacementHot, displacementHotBoost
	}
	return DisplacementWarm, displacementWarmBoost
}

func DisplacementTag(tier DisplacementTier) string {
	if tier == "" {
		return ""
	}
	return "displacement-" + string(tier)
}
