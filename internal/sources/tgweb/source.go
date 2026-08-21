package tgweb

import "strings"

// SiteDomainFromSource returns the crawled site hostname from a tgweb source label.
// Parse tgweb:domain and tgweb:@channel:domain via the last colon segment.
func SiteDomainFromSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if !strings.HasPrefix(source, "tgweb:") {
		return ""
	}
	rest := strings.TrimPrefix(source, "tgweb:")
	if rest == "" {
		return ""
	}
	if idx := strings.LastIndex(rest, ":"); idx >= 0 {
		// tgweb:@channel:example.com -> example.com
		return strings.TrimSpace(rest[idx+1:])
	}
	return strings.TrimSpace(rest)
}
