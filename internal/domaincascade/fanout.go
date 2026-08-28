package domaincascade

import (
	"strings"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/sourceregistry"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/sources/tgweb"
	"github.com/bidshard/parser/internal/sources/webpain"
)

// Config wires runtime registries for domain fan-out.
type Config struct {
	RegistryPath        string
	TelegramDomainsPath string
	ForumRegistryPath   string
	WebPainRegistryPath string
}

// FanOut registers one domain for supply, lander, and tgweb crawls.
func FanOut(cfg Config, domain, discoveredBy, source string) (registryAdded bool, tgwebQueued bool, err error) {
	domain = sourceregistry.NormalizeDomain(domain)
	if domain == "" || !sourceregistry.AcceptCascadeDomain(domain) {
		return false, false, nil
	}
	discoveredBy = strings.TrimSpace(discoveredBy)
	source = strings.TrimSpace(source)
	if source == "" {
		source = discoveredBy
	}

	registryAdded, err = sourceregistry.UpsertCascade(cfg.RegistryPath, domain, discoveredBy, source, "")
	if err != nil {
		return false, false, err
	}
	if cfg.TelegramDomainsPath != "" {
		tgwebQueued, err = tgweb.QueueDomain(cfg.TelegramDomainsPath, domain, source)
		if err != nil {
			return registryAdded, false, err
		}
	}
	return registryAdded, tgwebQueued, nil
}

// FanOutMany fans out deduped domains.
func FanOutMany(cfg Config, domains []string, discoveredBy, source string) (registryAdded int, tgwebQueued int, err error) {
	seen := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		domain = sourceregistry.NormalizeDomain(domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		regAdded, tgAdded, fanErr := FanOut(cfg, domain, discoveredBy, source)
		if fanErr != nil {
			return registryAdded, tgwebQueued, fanErr
		}
		if regAdded {
			registryAdded++
		}
		if tgAdded {
			tgwebQueued++
		}
	}
	return registryAdded, tgwebQueued, nil
}

// SyncDiscoveryRegistries cascades domains from web pain and forum SERP registries.
func SyncDiscoveryRegistries(cfg Config) (registryAdded int, tgwebQueued int, err error) {
	var domains []string

	if cfg.WebPainRegistryPath != "" {
		f, loadErr := webpain.LoadRegistry(cfg.WebPainRegistryPath)
		if loadErr != nil {
			return 0, 0, loadErr
		}
		for _, e := range f.URLs {
			if h := strings.TrimSpace(e.Host); h != "" {
				domains = append(domains, h)
			}
		}
	}

	if cfg.ForumRegistryPath != "" {
		tf, loadErr := forum.LoadThreadRegistry(cfg.ForumRegistryPath)
		if loadErr != nil {
			return 0, 0, loadErr
		}
		for _, e := range tf.Threads {
			domains = append(domains, extract.WebDomains(e.Snippet+" "+e.Title)...)
		}
	}

	return FanOutMany(cfg, domains, "serp_registry", "serp")
}
