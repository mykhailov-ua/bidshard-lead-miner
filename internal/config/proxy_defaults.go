package config

// defaultProxySources scopes residential/datacenter proxy to CF-heavy crawlers.
// SERP (DuckDuckGo), reddit API, github API, and supply seed fetch stay on direct egress.
var defaultProxySources = []string{"forum", "tgweb", "lander", "webpain"}

func applyProxyDefaults(cfg *Config) {
	if len(cfg.ProxyURLs) == 0 {
		return
	}
	if len(cfg.ProxySources) > 0 {
		return
	}
	cfg.ProxySources = append([]string{}, defaultProxySources...)
}

// DefaultProxySources returns the proxy scope applied when PARSER_PROXY_SOURCES is unset.
func DefaultProxySources() []string {
	return append([]string{}, defaultProxySources...)
}
