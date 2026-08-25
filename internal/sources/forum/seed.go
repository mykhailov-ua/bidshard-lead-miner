package forum

import (
	"fmt"
	"os"
	"strings"

	"github.com/bidshard/parser/internal/seedcsv"
)

type ThreadSeed struct {
	URL     string
	Notes   string
	Title   string
	Snippet string
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
	seeds, err := loadThreadSeedsFromCSV(path)
	if err != nil {
		return nil, err
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no forum urls in %s", path)
	}
	return seeds, nil
}

// LoadThreadSeedsOptional reads CSV seeds; missing or empty file returns nil without error.
func LoadThreadSeedsOptional(path string) ([]ThreadSeed, error) {
	return loadThreadSeedsFromCSV(path)
}

func loadThreadSeedsFromCSV(path string) ([]ThreadSeed, error) {
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
	return seeds, nil
}

// LoadThreadSeedsCombined merges runtime registry threads with optional CSV seeds.
func LoadThreadSeedsCombined(csvPath, registryPath string) ([]ThreadSeed, error) {
	seen := map[string]struct{}{}
	var seeds []ThreadSeed

	if registryPath != "" {
		reg, err := LoadThreadRegistry(registryPath)
		if err != nil {
			return nil, err
		}
		for _, e := range reg.Threads {
			url := strings.TrimSpace(e.URL)
			if url == "" {
				continue
			}
			if _, ok := seen[url]; ok {
				continue
			}
			seen[url] = struct{}{}
			notes := strings.TrimSpace(e.Notes)
			if notes == "" && e.Query != "" {
				notes = "discover:" + e.Query
			}
			seeds = append(seeds, ThreadSeed{
				URL:     url,
				Notes:   notes,
				Title:   strings.TrimSpace(e.Title),
				Snippet: strings.TrimSpace(e.Snippet),
			})
		}
	}

	csvSeeds, err := LoadThreadSeedsOptional(csvPath)
	if err != nil {
		return nil, err
	}
	for _, s := range csvSeeds {
		if _, ok := seen[s.URL]; ok {
			continue
		}
		seen[s.URL] = struct{}{}
		seeds = append(seeds, s)
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no forum thread seeds (csv=%s registry=%s)", csvPath, registryPath)
	}
	return seeds, nil
}
