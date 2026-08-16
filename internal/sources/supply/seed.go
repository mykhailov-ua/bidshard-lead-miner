package supply

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/bidshard/parser/internal/geo"
)

func LoadSeedDomains(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
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
		domain = strings.TrimPrefix(strings.ToLower(domain), "https://")
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimSuffix(domain, "/")
		if strings.Contains(domain, "/") {
			domain = strings.Split(domain, "/")[0]
		}
		if res := geo.Filter(domain); !res.OK {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains loaded from %s", path)
	}
	return domains, nil
}
