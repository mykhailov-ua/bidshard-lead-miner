package forum

import (
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/seedcsv"
)

func LoadThreadURLs(path string) ([]string, error) {
	records, err := seedcsv.ReadRecords(path)
	if err != nil {
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
	if len(urls) == 0 {
		return nil, fmt.Errorf("no forum urls in %s", path)
	}
	return urls, nil
}
