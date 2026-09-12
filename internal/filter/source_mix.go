package filter

import "strings"

// H13 source mix tiers (TG-first, not TG-only).
const (
	SourceMixHot          = "hot"
	SourceMixWarm         = "warm"
	SourceMixIntel        = "intel"
	SourceMixDeprioritize = "deprioritize"
)

// TGFirstSources is the recommended PARSER_SOURCE list for BidShard hot/warm path (H13).
func TGFirstSources() string {
	return "forum,serp,jobboard,tgweb"
}

// SourceMixTier classifies crawler families for ops dashboards and collect ordering.
func SourceMixTier(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.HasPrefix(source, "telegram:"):
		return SourceMixHot
	case strings.HasPrefix(source, "forum:"), strings.HasPrefix(source, "reviews:"):
		return SourceMixHot
	case strings.HasPrefix(source, "serp:"), strings.HasPrefix(source, "jobboard:"), strings.HasPrefix(source, "discord:"):
		return SourceMixWarm
	case strings.HasPrefix(source, "tgweb:"), strings.HasPrefix(source, "infraosint:"), strings.HasPrefix(source, "lander:"), strings.HasPrefix(source, "supply:"):
		return SourceMixIntel
	case strings.HasPrefix(source, "reddit:"), strings.HasPrefix(source, "webpain:"), strings.HasPrefix(source, "github:"):
		return SourceMixDeprioritize
	default:
		return SourceMixWarm
	}
}

// IsH13DeprioritizedSource reports cron-only surfaces (H13).
func IsH13DeprioritizedSource(source string) bool {
	return SourceMixTier(source) == SourceMixDeprioritize
}
