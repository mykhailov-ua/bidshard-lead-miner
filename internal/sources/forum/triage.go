package forum

import "strings"

var reviewSkipTokens = []string{
	"review", "comparison", "compare", "vs-", "-vs-", "best-tracker",
	"top-tracker", "roundup", "alternatives-to",
}

// ShouldCrawlSeed skips review/comparison threads before HTTP fetch.
func ShouldCrawlSeed(url, notes string) bool {
	text := strings.ToLower(strings.TrimSpace(url + " " + notes))
	for _, tok := range reviewSkipTokens {
		if strings.Contains(text, tok) {
			return false
		}
	}
	return true
}
