package filter

import "strings"

// Affiliate network manager / supply-side outreach (not buyer pain).
var affiliateNetworkSupplyPhrases = []string{
	"affiliate network",
	"in-house media buying",
	"in-house buying",
	"partnership manager",
	"cpa network",
	"direct advertiser",
	"we offer traffic",
	"our offers",
	"manager @",
	"aff manager",
	"affiliate manager",
}

// RejectAffiliateNetworkSupply drops partner program promos without buyer voice.
func RejectAffiliateNetworkSupply(text, title string) (bool, string) {
	combined := strings.ToLower(strings.TrimSpace(title + " " + text))
	if combined == "" {
		return false, ""
	}
	if HasCommercialPainIntent(combined) || HasBuyerQuestionPattern(combined) {
		return false, ""
	}
	hits := 0
	for _, phrase := range affiliateNetworkSupplyPhrases {
		if strings.Contains(combined, phrase) {
			hits++
		}
	}
	if hits >= 2 {
		return true, "affiliate network supply"
	}
	if hits >= 1 && (strings.Contains(combined, "manager") || strings.Contains(combined, "network")) {
		return true, "affiliate network supply"
	}
	return false, ""
}
