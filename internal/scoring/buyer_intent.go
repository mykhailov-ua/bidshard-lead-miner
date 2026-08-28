package scoring

import (
	"regexp"
	"strings"
)

var buyerIntentPhrases = []string{
	"looking for", "need a tracker", "need tracker", "anyone facing", "does anyone",
	"has anyone", "help with", "recommend", "our subscription", "clients looking",
	"we run", "i run", "my team", "our team", "we are looking", "i am looking",
	"we're looking", "i'm looking",
}

var buyerIntentRe = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:i|we)\s+(?:am|are|'m|'re)\s+(?:considering|evaluating|comparing)\b`),
	regexp.MustCompile(`(?i)\b(?:can|could)\s+(?:someone|anyone)\s+(?:help|recommend)\b`),
}

// HasBuyerIntentSignal reports first-person buyer language or outreach-worthy questions.
func HasBuyerIntentSignal(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	if buyerRoleRe.MatchString(lower) {
		return true
	}
	for _, phrase := range buyerIntentPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	for _, re := range buyerIntentRe {
		if re.MatchString(lower) {
			return true
		}
	}
	if strings.Contains(lower, "?") && len([]rune(lower)) >= 32 {
		return true
	}
	return false
}
