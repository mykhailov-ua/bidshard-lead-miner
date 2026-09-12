package validate

import (
	"regexp"
	"strings"
)

var listicleTitlePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b\d+\s+best\s+.+\s+alternatives?\b`),
	regexp.MustCompile(`(?i)\bbest\s+.+\s+alternatives?\s+in\s+20\d{2}\b`),
	regexp.MustCompile(`(?i)\btop\s+\d+\s+`),
	regexp.MustCompile(`(?i)\btop\s+.+\s+tools?\b`),
	regexp.MustCompile(`(?i)\b.+\s+vs\s+.+\b`),
	regexp.MustCompile(`(?i)\b(?:full\s+)?review\b`),
	regexp.MustCompile(`(?i)\bpricing\b`),
}

// SEO listicle and competitor landing markers (not buyer dialog).
var seoMarketingPhrases = []string{
	"top picks",
	"best software",
	"best affiliate tracking",
	"alternative designed specifically",
	"designed specifically for",
	"sign up for",
	"sign up now",
	"get started free",
	"pricing plans",
	"compare plans",
	"our top pick",
	"read our review",
	"comparison guide",
	"affiliate tracking software in 20",
	"tracking software in 202",
	"free trial",
	"start your free",
	"book a demo",
	"request a demo",
}

// IsSEOMarketingCopy reports vendor listicles and SaaS landing copy without buyer voice.
func IsSEOMarketingCopy(text, title string) bool {
	combined := strings.ToLower(strings.TrimSpace(title + " " + text))
	if combined == "" {
		return false
	}
	marketing := false
	for _, phrase := range seoMarketingPhrases {
		if strings.Contains(combined, phrase) {
			marketing = true
			break
		}
	}
	if !marketing && IsListicleTitle(title) {
		marketing = true
	}
	if !marketing {
		return false
	}
	return !hasRealBuyerVoice(combined)
}

// IsListicleTitle reports SEO comparison articles and competitor listicles.
func IsListicleTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	for _, re := range listicleTitlePatterns {
		if re.MatchString(title) {
			return true
		}
	}
	lower := strings.ToLower(title)
	if strings.Contains(lower, "best ") && strings.Contains(lower, "alternative") {
		return true
	}
	if strings.Contains(lower, "top ") && strings.Contains(lower, "software") {
		return true
	}
	return false
}

// ListicleScorePenalty is applied to keyword-inflated SEO articles.
func ListicleScorePenalty(title string) int {
	if IsListicleTitle(title) {
		return 200
	}
	return 0
}

func hasRealBuyerVoice(combined string) bool {
	if HasBuyerQuestionPattern(combined) {
		return true
	}
	voice := []string{
		"postback", "not working", "failing", "broken", "migrate", "switching from",
		"looking for", "need a tracker", "need tracker", "recommend", "anyone",
		"help with", "does anyone", "how do i", "what tracker",
	}
	for _, hint := range voice {
		if strings.Contains(combined, hint) {
			return true
		}
	}
	return false
}
