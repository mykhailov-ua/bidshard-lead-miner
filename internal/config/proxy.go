package config

import "strings"

// ProxyURLsForSource returns PARSER_PROXY_LIST when the source is allowed to use proxy egress.
// Empty PARSER_PROXY_SOURCES keeps legacy behavior: all HTTP crawlers use the proxy list when set.
func (c Config) ProxyURLsForSource(sourceID string) []string {
	if len(c.ProxyURLs) == 0 {
		return nil
	}
	if !c.ProxyEnabledForSource(sourceID) {
		return nil
	}
	return c.ProxyURLs
}

// ProxyEnabledForSource reports whether sourceID may use PARSER_PROXY_LIST.
func (c Config) ProxyEnabledForSource(sourceID string) bool {
	sourceID = strings.ToLower(strings.TrimSpace(sourceID))
	if sourceID == "" {
		return false
	}
	if len(c.ProxyURLs) == 0 {
		return false
	}
	if len(c.ProxySources) == 0 {
		return true
	}
	for _, s := range c.ProxySources {
		if strings.EqualFold(s, sourceID) {
			return true
		}
	}
	return false
}
