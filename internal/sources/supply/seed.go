package supply

import (
	"fmt"
	"os"
	"strings"

	"github.com/bidshard/parser/internal/geo"
	"github.com/bidshard/parser/internal/seedcsv"
	"github.com/bidshard/parser/internal/sourceregistry"
)

func LoadSeedDomainsCombined(csvPath, registryPath string) ([]string, error) {
	seen := map[string]struct{}{}
	var domains []string

	appendDomain := func(domain string) {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			return
		}
		if _, ok := seen[domain]; ok {
			return
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}

	if registryPath != "" {
		regDomains, err := sourceregistry.ListDomainsByType(registryPath, sourceregistry.TypeSupply)
		if err != nil {
			return nil, err
		}
		for _, d := range regDomains {
			appendDomain(d)
		}
	}

	csvDomains, err := LoadSeedDomainsOptional(csvPath)
	if err != nil {
		return nil, err
	}
	for _, d := range csvDomains {
		appendDomain(d)
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains loaded (csv=%s registry=%s)", csvPath, registryPath)
	}
	return domains, nil
}

// LoadSeedDomainsOptional reads CSV seeds; missing or empty file returns nil without error.
func LoadSeedDomainsOptional(path string) ([]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	records, err := seedcsv.ReadRecords(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var domains []string
	seen := map[string]struct{}{}
	for i, row := range records {
		if len(row) == 0 {
			continue
		}
		domain := strings.TrimSpace(row[0])
		if domain == "" || domain == "domain" {
			continue
		}
		if i == 0 && strings.EqualFold(domain, "domain") {
			continue
		}
		if len(row) > 1 {
			geoTag := strings.ToLower(strings.TrimSpace(row[1]))
			if geoTag == "ru" || geoTag == "by" {
				continue
			}
		}
		domain = strings.TrimPrefix(strings.ToLower(domain), "https://")
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimSuffix(domain, "/")
		if strings.Contains(domain, "/") {
			domain = strings.Split(domain, "/")[0]
		}
		if res := geo.Filter(domain); !res.OK {
			continue
		}
		if geo.IsBlockedTLD(domain) {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	return domains, nil
}

func LoadSeedDomains(path string) ([]string, error) {
	domains, err := LoadSeedDomainsOptional(path)
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains loaded from %s", path)
	}
	return domains, nil
}
