package scoring

import (
	"strings"

	"github.com/bidshard/parser/internal/validate"
)

// PrescanPasses reports whether text should enter contact extraction/scoring.
// tgweb leads from affiliate network sites may lack forum-style pain phrases in page copy;
// allow pain-context hints and affiliate-vertical signals from our crawl prefix.
func PrescanPasses(source string, reg *Registry, text string) bool {
	if reg != nil && reg.Prescan(text) {
		return true
	}
	if isTgWebSource(source) {
		return validate.HasPainContext(text) || validate.HasAffiliateNetworkContext(text)
	}
	return false
}

func isTgWebSource(source string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "tgweb:")
}
