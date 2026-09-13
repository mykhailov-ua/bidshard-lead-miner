package filter

import (
	"regexp"
	"strings"
)

var discordInvoiceNoiseRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\binvoice\b`),
	regexp.MustCompile(`(?i)\bpayment\s+request\b`),
	regexp.MustCompile(`(?i)\bnet\s*[- ]?30\b`),
	regexp.MustCompile(`(?i)\bwire\s+transfer\b`),
	regexp.MustCompile(`(?i)\baccounts?\s+receivable\b`),
}

// IsDiscordSource reports Discord API crawl items.
func IsDiscordSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "discord:")
}

// DiscordInvoiceNoise drops finance/admin chatter that is not buyer pain.
func DiscordInvoiceNoise(text string) (bool, string) {
	body := strings.TrimSpace(text)
	if body == "" {
		return false, ""
	}
	for _, re := range discordInvoiceNoiseRe {
		if re.MatchString(body) {
			if HasCommercialPainIntent(body) || HasBuyerQuestionPattern(body) {
				return false, ""
			}
			return true, "discord invoice noise"
		}
	}
	return false, ""
}
