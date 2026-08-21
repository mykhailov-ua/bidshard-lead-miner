package scoring

import (
	"regexp"
	"strings"
)

// TagPublisherSurface marks accepted leads from ads.txt / sellers.json supply crawl (ads_txt: source).
const TagPublisherSurface = "publisher-surface"

var (
	spendRe      = regexp.MustCompile(`(?i)(?:\$\d+k?|\d+\$/day|500/day|budget|high spend|enterprise|1k|5k|10k)`)
	painRe       = regexp.MustCompile(`(?i)(overburn|click loss|postback fail|missing ftd|billing|expensive|overpriced|broken|bug)`)
	infraRe      = regexp.MustCompile(`(?i)(vps|server|hetzner|digitalocean|docker|clickhouse|self-hosted|aws|ovh)`)
	usdtRe       = regexp.MustCompile(`(?i)(usdt|crypto|binance|trc20|tether|webmoney)`)
	buyerRoleRe  = regexp.MustCompile(`(?i)(founder|co-founder|ceo|cto|cio|cfo|owner|president|managing director|general manager|head of (?:media|acquisition|affiliate|programmatic|ad ops)|vp (?:of )?(?:engineering|media|acquisition|affiliate)|affiliate director|director of acquisition)`)
	volumeRe     = regexp.MustCompile(`(?i)(100\+ ftd|high volume|scaling|10k clicks|50k clicks|volume|traffic)`)
	migrRe       = regexp.MustCompile(`(?i)(switching from|migrate|migrating|looking for alternative|alternative to|replacing|replace)`)
	competitorRe = regexp.MustCompile(`(?i)(voluum|keitaro|binom|redtrack|clickflare|bemob|thrivetracker)`)
)

// PilotQualified evaluates 8 checklist signals for pilot eligibility.
// Qualified requires at least 3 independent signals; tags only (no auto-reject).
func PilotQualified(spendTier string, stack []string, text string) (bool, []string) {
	lower := strings.ToLower(text)
	var tags []string

	if spendRe.MatchString(lower) || spendTierQualifies(spendTier) {
		tags = append(tags, "pilot-spend-budget")
	}

	if len(stack) > 0 || len(DetectCompetitorStack(text)) > 0 || competitorRe.MatchString(lower) {
		tags = append(tags, "pilot-competitor-stack")
	}

	if painRe.MatchString(lower) {
		tags = append(tags, "pilot-tracker-pain")
	}

	if infraRe.MatchString(lower) {
		tags = append(tags, "pilot-infra-vps")
	}

	if usdtRe.MatchString(lower) {
		tags = append(tags, "pilot-usdt-ok")
	}

	if buyerRoleRe.MatchString(lower) {
		tags = append(tags, "pilot-buyer-role")
	}

	if volumeRe.MatchString(lower) {
		tags = append(tags, "pilot-high-volume")
	}

	if migrRe.MatchString(lower) {
		tags = append(tags, "pilot-migration-intent")
	}

	isQualified := len(tags) >= 3
	if isQualified {
		tags = append([]string{"pilot-qualified"}, tags...)
	} else {
		tags = append([]string{"pilot-nurture"}, tags...)
	}

	return isQualified, tags
}

func spendTierQualifies(tier string) bool {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "", "unknown", "none":
		return false
	default:
		return true
	}
}
