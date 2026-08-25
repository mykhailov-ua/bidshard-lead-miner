package config

import "fmt"

// AutoDiscoverRegistryOK reports whether runtime registry satisfies prod seed check.
func AutoDiscoverRegistryOK(cfg Config, src string, registryCount int) (bool, string) {
	if !cfg.ParserAutoDiscover || registryCount <= 0 {
		return false, ""
	}
	switch src {
	case "forum":
		return true, fmt.Sprintf("forum auto-discover registry: %d threads", registryCount)
	case "supply":
		return true, fmt.Sprintf("supply source_registry: %d domains", registryCount)
	default:
		return false, ""
	}
}
