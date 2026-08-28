package filter

import "strings"

const (
	CollectPriorityHigh   = 100
	CollectPriorityMedium = 50
	CollectPriorityLow    = 10
)

// CollectPriorityFamily ranks crawler families for scan-round ordering (higher first).
func CollectPriorityFamily(family string) int {
	switch strings.ToLower(strings.TrimSpace(family)) {
	case "reddit", "forum", "reviews":
		return CollectPriorityHigh
	case "supply", "serp", "discord":
		return CollectPriorityMedium
	case "lander", "github", "webpain":
		return CollectPriorityLow
	default:
		return CollectPriorityMedium
	}
}

// CollectPriority ranks accepted lead sources for intent-gate and observability.
func CollectPriority(source string) int {
	source = strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.HasPrefix(source, "reddit:"), strings.HasPrefix(source, "forum:"), strings.HasPrefix(source, "reviews:"):
		return CollectPriorityHigh
	case strings.HasPrefix(source, "supply:"), strings.HasPrefix(source, "ads_txt:"), strings.HasPrefix(source, "serp:"), strings.HasPrefix(source, "discord:"):
		return CollectPriorityMedium
	case strings.HasPrefix(source, "webpain:"), strings.HasPrefix(source, "github:"), strings.HasPrefix(source, "lander:"):
		return CollectPriorityLow
	case strings.HasPrefix(source, "telegram:invite:"):
		return CollectPriorityLow
	case isTelegramSource(source):
		return CollectPriorityMedium
	default:
		return CollectPriorityMedium
	}
}

// IsIntelOnlySource reports crawl items that feed supply/tgweb intel, not CRM outreach.
func IsIntelOnlySource(source string, landerOutreach bool) bool {
	if landerOutreach {
		return false
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "lander:")
}

// SourceRequiresIntentGate is true for low-trust surfaces that need Gemini buyer-intent classify.
func SourceRequiresIntentGate(source, chatType string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.HasPrefix(source, "reddit:"), strings.HasPrefix(source, "forum:"), strings.HasPrefix(source, "reviews:"):
		return false
	case strings.HasPrefix(source, "lander:"), strings.HasPrefix(source, "github:"), strings.HasPrefix(source, "webpain:"):
		return true
	case strings.HasPrefix(source, "telegram:invite:"):
		return true
	case isTelegramSource(source):
		return strings.EqualFold(strings.TrimSpace(chatType), "channel")
	default:
		return false
	}
}
