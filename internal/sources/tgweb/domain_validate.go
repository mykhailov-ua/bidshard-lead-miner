package tgweb

import (
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/geo"
)

// IsValidCrawlDomain reports whether a registry host is worth HTTP crawling.
func IsValidCrawlDomain(domain string) bool {
	if geo.IsBlockedTLD(domain) {
		return false
	}
	return extract.IsValidWebDomain(domain)
}
