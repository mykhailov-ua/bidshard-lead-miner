package tgweb

import (
	"strings"

	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sourceregistry"
)

// SyncToSourceRegistry upserts tgweb registry rows into the unified source registry.
// Cross-mention affiliate domains fan out to tgweb + supply crawlers.
func SyncToSourceRegistry(tgDomainsPath, registryPath string) (added int, err error) {
	f, err := LoadDomains(tgDomainsPath)
	if err != nil {
		return 0, err
	}
	for _, row := range f.Domains {
		domain := strings.ToLower(strings.TrimSpace(row.Domain))
		if domain == "" || !IsValidCrawlDomain(domain) {
			continue
		}
		discoveredBy := strings.TrimSpace(row.Source)
		if discoveredBy == "" {
			discoveredBy = "telegram"
		}
		types := append([]string(nil), sourceregistry.CascadeTypes...)
		if strings.Contains(strings.ToLower(discoveredBy), "cross") || strings.Contains(strings.ToLower(discoveredBy), "mention") {
			types = []string{sourceregistry.TypeSupply, sourceregistry.TypeTGWeb, sourceregistry.TypeLander}
		}
		ok, upsertErr := sourceregistry.Upsert(registryPath, sourceregistry.Entry{
			Domain:       domain,
			Types:        types,
			DiscoveredBy: discoveredBy,
			Channel:      row.Channel,
			Source:       row.Source,
		})
		if upsertErr != nil {
			return added, upsertErr
		}
		if ok {
			added++
		}
	}
	if added > 0 {
		metrics.RecordSourcesDiscovered("source_registry", added)
	}
	return added, nil
}
