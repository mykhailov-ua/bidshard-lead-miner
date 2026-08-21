package validate

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	emailStripRe    = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	urlStripRe      = regexp.MustCompile(`(?i)https?://\S+`)
	telegramStripRe = regexp.MustCompile(`(?:telegram:)?@[a-zA-Z][a-zA-Z0-9_]{3,}`)
)

var painContextHints = []string{
	"voluum", "keitaro", "binom", "redtrack", "clickflare", "bemob", "thrivetracker",
	"postback", "tracker", "alternative", "afterburn", "ftd", "self-hosted", "self hosted",
	"cloak", "cloaker", "safe page", "clickid", "click id", "conversion", "billing",
	"expensive", "migrate", "switching from", "s2s", "openrtb", "budget overburn",
	"missing ftd", "per-event", "media buyer", "affiliate team",
	"affiliate marketing", "igaming", "partner program", "media buying", "cpa network",
	"affiliate network", "performance marketing", "arbitrage",
}

var strictPainContextHints = []string{
	"voluum", "keitaro", "binom", "redtrack", "clickflare", "bemob", "thrivetracker",
	"postback", "alternative", "afterburn", "ftd", "self-hosted", "self hosted",
	"cloak", "cloaker", "safe page", "clickid", "click id", "conversion", "billing",
	"expensive", "migrate", "switching from", "s2s", "openrtb", "budget overburn",
	"missing ftd", "per-event", "tracker",
}

// HasStrictPainContext requires tracker-specific pain, not generic affiliate page copy.
func HasStrictPainContext(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, hint := range strictPainContextHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// HasPainContext reports whether text has tracker-pain signals beyond a bare contact.
func HasPainContext(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	for _, hint := range painContextHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	remainder := stripContactArtifacts(lower)
	return meaningfulRunes(remainder) >= 48
}

// HasAffiliateNetworkContext reports vertical signals for partner/affiliate program sites.
func HasAffiliateNetworkContext(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	hints := []string{
		"affiliate program", "partner program", "affiliate network", "affiliate marketing",
		"igaming", "media buying", "performance marketing", "cpa network", "become a partner",
		"partner with us", "affiliate team", "operators and affiliates",
	}
	for _, hint := range hints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

// EmailWithoutPainContext is true when the snippet is essentially only an email/contact.
func EmailWithoutPainContext(text string) bool {
	if !emailStripRe.MatchString(text) {
		return false
	}
	return !HasPainContext(text)
}

func stripContactArtifacts(text string) string {
	text = emailStripRe.ReplaceAllString(text, " ")
	text = urlStripRe.ReplaceAllString(text, " ")
	text = telegramStripRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func meaningfulRunes(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n
}
