package supply

import (
	"strings"

	"github.com/bidshard/parser/internal/sourceregistry"
)

// PartnerDomainsFromCrawl extracts ads.txt partner hosts for cascade; SSP/CDN rows are skipped.
func PartnerDomainsFromCrawl(crawlDomain string, lines []AdsTxtLine, sellers []SellerContact) []string {
	crawlDomain = sourceregistry.NormalizeDomain(crawlDomain)
	seen := make(map[string]struct{})
	var out []string
	add := func(domain string) {
		domain = sourceregistry.NormalizeDomain(domain)
		if domain == "" || domain == crawlDomain {
			return
		}
		if !sourceregistry.AcceptCascadePartnerDomain(domain) {
			return
		}
		if _, ok := seen[domain]; ok {
			return
		}
		seen[domain] = struct{}{}
		out = append(out, domain)
	}

	for _, line := range lines {
		add(line.Domain)
	}
	for _, s := range sellers {
		add(s.Domain)
		if email := strings.TrimSpace(s.ContactEmail); email != "" {
			parts := strings.Split(strings.ToLower(email), "@")
			if len(parts) == 2 {
				add(parts[1])
			}
		}
	}
	return out
}
