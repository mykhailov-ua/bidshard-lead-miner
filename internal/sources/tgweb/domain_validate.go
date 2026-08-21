package tgweb

import "github.com/bidshard/parser/internal/extract"

// IsValidCrawlDomain reports whether a registry host is worth HTTP crawling.
func IsValidCrawlDomain(domain string) bool {
	return extract.IsValidWebDomain(domain)
}
