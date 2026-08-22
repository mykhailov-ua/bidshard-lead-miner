package forum

import (
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/seedcsv"
)

type ThreadSeed struct {
	URL   string
	Notes string
}

func LoadThreadURLs(path string) ([]string, error) {
	seeds, err := LoadThreadSeeds(path)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(seeds))
	for _, s := range seeds {
		urls = append(urls, s.URL)
	}
	return urls, nil
}

func LoadThreadSeeds(path string) ([]ThreadSeed, error) {
	records, err := seedcsv.ReadRecords(path)
	if err != nil {
		return nil, err
	}

	var seeds []ThreadSeed
	seen := map[string]struct{}{}
	for _, row := range records {
		if len(row) == 0 {
			continue
		}
		url := strings.TrimSpace(row[0])
		if url == "" || strings.EqualFold(url, "url") {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		notes := ""
		if len(row) > 1 {
			notes = strings.TrimSpace(row[1])
		}
		seeds = append(seeds, ThreadSeed{URL: url, Notes: notes})
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no forum urls in %s", path)
	}
	return seeds, nil
}
