package forum

import "strings"

const (
	ThreadVerdictBuyerIntent = "igaming_buyer_intent"
	ThreadVerdictNoise       = "noise"
)

var reviewSkipTokens = []string{
	"review", "comparison", "compare", "vs-", "-vs-", "best-tracker",
	"top-tracker", "roundup", "alternatives-to",
}

var threadNoiseTokens = []string{
	"job posting", "we are hiring", "hiring media buyer", "vacancy",
	"coupon code", "promo code giveaway", "free spins", "casino bonus review",
	"wordpress plugin", "vpn review", "hosting review", "affiliate program review",
	"best trackers 20", "top 10 trackers",
}

var threadBuyerIntentTokens = []string{
	"voluum", "postback", "tracker", "keitaro", "binom", "redtrack",
	"self-hosted tracker", "s2s", "igaming affiliate", "media buy",
	"migration", "billing", "too expensive", "failing", "alternative",
}

// TriageThread classifies a forum thread from URL and SERP metadata before HTTP fetch.
func TriageThread(seed ThreadSeed) (fetch bool, verdict string) {
	text := strings.ToLower(strings.TrimSpace(
		seed.URL + " " + seed.Notes + " " + seed.Title + " " + seed.Snippet,
	))
	if text == "" {
		return true, ""
	}
	for _, tok := range reviewSkipTokens {
		if strings.Contains(text, tok) {
			return false, ThreadVerdictNoise
		}
	}
	for _, tok := range threadNoiseTokens {
		if strings.Contains(text, tok) {
			return false, ThreadVerdictNoise
		}
	}
	for _, tok := range threadBuyerIntentTokens {
		if strings.Contains(text, tok) {
			return true, ThreadVerdictBuyerIntent
		}
	}
	return true, ""
}

// ShouldCrawlSeed skips review/comparison threads before HTTP fetch.
func ShouldCrawlSeed(url, notes string) bool {
	fetch, _ := TriageThread(ThreadSeed{URL: url, Notes: notes})
	return fetch
}
