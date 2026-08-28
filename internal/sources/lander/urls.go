package lander

import (
	"fmt"
	"os"
	"strings"

	"github.com/bidshard/parser/internal/seedcsv"
	"github.com/bidshard/parser/internal/sourceregistry"
)

func LoadURLs(path string) ([]string, error) {
	urls, err := LoadURLsOptional(path)
	if err != nil {
		return nil, err
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no lander urls in %s", path)
	}
	return urls, nil
}

// LoadURLsOptional reads CSV seeds; missing or empty file returns nil without error.
func LoadURLsOptional(path string) ([]string, error) {
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

	var urls []string
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
		urls = append(urls, url)
	}
	return urls, nil
}

// LoadURLsCombined merges source_registry lander domains with CSV seed URLs.
func LoadURLsCombined(csvPath, registryPath string) ([]string, error) {
	seen := map[string]struct{}{}
	var urls []string
	appendURL := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		if _, ok := seen[raw]; ok {
			return
		}
		seen[raw] = struct{}{}
		urls = append(urls, raw)
	}

	if registryPath != "" {
		domains, err := sourceregistry.ListDomainsByType(registryPath, sourceregistry.TypeLander)
		if err != nil {
			return nil, err
		}
		for _, domain := range domains {
			appendURL(PrimarySeedURL(domain))
		}
	}

	csvURLs, err := LoadURLsOptional(csvPath)
	if err != nil {
		return nil, err
	}
	for _, u := range csvURLs {
		appendURL(u)
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no lander urls (csv=%s registry=%s)", csvPath, registryPath)
	}
	return urls, nil
}

// PrimarySeedURL picks an affiliate LPR path for registry-sourced lander crawls.
func PrimarySeedURL(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return ""
	}
	for _, p := range NetworkLandingPaths() {
		if strings.Contains(p, "affiliate") || strings.Contains(p, "partner") {
			return "https://" + domain + p
		}
	}
	return "https://" + domain + "/"
}
