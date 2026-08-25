package sourceregistry

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/validate"
)

// DomainMeta is registry metadata used for heuristic and Gemini triage.
type DomainMeta struct {
	Domain       string
	Channel      string
	Source       string
	DiscoveredBy string
	Kind         string
}

var heuristicDropHosts = map[string]struct{}{
	"facebook.com":   {},
	"instagram.com":  {},
	"twitter.com":    {},
	"x.com":          {},
	"youtube.com":    {},
	"tiktok.com":     {},
	"linkedin.com":   {},
	"t.me":           {},
	"telegram.me":    {},
	"telegram.org":   {},
	"google.com":     {},
	"bing.com":       {},
	"duckduckgo.com": {},
	"bit.ly":         {},
	"tinyurl.com":    {},
	"goo.gl":         {},
	"cloudflare.com": {},
	"amazonaws.com":  {},
	"github.com":     {},
	"medium.com":     {},
	"wordpress.com":  {},
	"blogspot.com":   {},
	"wikipedia.org":  {},
}

// HeuristicTriage returns action keep|drop when metadata is enough to decide without Gemini.
func HeuristicTriage(meta DomainMeta) (action string, why string, decided bool) {
	domain := normalizeDomain(meta.Domain)
	if domain == "" {
		return "drop", "empty_domain", true
	}
	if validate.IsBlacklistedDomain(domain) {
		return "drop", "blacklist", true
	}
	if geo.IsBlockedTLD(domain) {
		return "drop", "blocked_tld", true
	}
	if !extract.IsValidWebDomain(domain) {
		return "drop", "invalid_host", true
	}
	host := domain
	if idx := strings.Index(host, "."); idx > 0 {
		host = host[idx+1:]
	}
	if _, ok := heuristicDropHosts[domain]; ok {
		return "drop", "noise_host", true
	}
	if _, ok := heuristicDropHosts[host]; ok {
		return "drop", "noise_host", true
	}
	for blocked := range heuristicDropHosts {
		if domain == blocked || strings.HasSuffix(domain, "."+blocked) {
			return "drop", "noise_host", true
		}
	}
	return "", "", false
}
