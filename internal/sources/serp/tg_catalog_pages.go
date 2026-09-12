package serp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultTGCatalogPagesPath = "data/runtime/discovered_tg_catalog_pages.json"

type tgCatalogPageEntry struct {
	URL          string `json:"url"`
	Source       string `json:"source"`
	Query        string `json:"query,omitempty"`
	DiscoveredAt string `json:"discovered_at"`
	CrawledAt    string `json:"crawled_at,omitempty"`
	HandlesFound int    `json:"handles_found,omitempty"`
	LastError    string `json:"last_error,omitempty"`
}

type tgCatalogPageFile struct {
	Pages []tgCatalogPageEntry `json:"pages"`
}

func readTGCatalogPageFile(path string) tgCatalogPageFile {
	if path == "" {
		path = defaultTGCatalogPagesPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return tgCatalogPageFile{}
	}
	var f tgCatalogPageFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return tgCatalogPageFile{}
	}
	return f
}

func writeTGCatalogPageFile(path string, f tgCatalogPageFile) error {
	if path == "" {
		path = defaultTGCatalogPagesPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func appendTGCatalogPageDiscoveries(path string, dork string, urls []string) (int, error) {
	if len(urls) == 0 {
		return 0, nil
	}
	f := readTGCatalogPageFile(path)
	seen := make(map[string]struct{}, len(f.Pages))
	for _, p := range f.Pages {
		seen[normalizeCatalogURL(p.URL)] = struct{}{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	added := 0
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || !isTGCatalogCandidateURL(u) {
			continue
		}
		key := normalizeCatalogURL(u)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		f.Pages = append(f.Pages, tgCatalogPageEntry{
			URL:          u,
			Source:       "serp",
			Query:        dork,
			DiscoveredAt: now,
		})
		added++
	}
	if added == 0 {
		return 0, nil
	}
	if err := writeTGCatalogPageFile(path, f); err != nil {
		return 0, err
	}
	return added, nil
}

func normalizeCatalogURL(u string) string {
	return strings.ToLower(strings.TrimSpace(u))
}

// isTGCatalogCandidateURL filters SERP hits to pages likely listing Telegram channels/chats.
func isTGCatalogCandidateURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "t.me/") && !strings.Contains(lower, "t.me/+") {
		// Direct channel link; handled by telegram catalog harvest, not page crawl.
		return false
	}
	host := catalogHost(lower)
	if host == "" {
		return false
	}
	switch {
	case strings.Contains(host, "tgstat."):
		return true
	case strings.Contains(host, "telemetr."):
		return true
	case strings.Contains(host, "tlgrm."):
		return true
	case strings.Contains(host, "combot."):
		return true
	case strings.Contains(host, "affiliatefix."):
		return strings.Contains(lower, "telegram") || strings.Contains(lower, "t.me")
	case strings.Contains(host, "afflift."):
		return strings.Contains(lower, "telegram") || strings.Contains(lower, "t.me")
	case strings.Contains(host, "blackhatworld."):
		return strings.Contains(lower, "telegram") || strings.Contains(lower, "t.me")
	case strings.Contains(host, "stmforum."):
		return strings.Contains(lower, "telegram") || strings.Contains(lower, "t.me")
	case strings.Contains(host, "gpwa."):
		return strings.Contains(lower, "telegram") || strings.Contains(lower, "t.me")
	case strings.Contains(host, "reddit.com"):
		return strings.Contains(lower, "/r/") &&
			(strings.Contains(lower, "telegram") || strings.Contains(lower, "t.me"))
	default:
		return strings.Contains(lower, "telegram") &&
			(strings.Contains(lower, "channel") || strings.Contains(lower, "group") || strings.Contains(lower, "chat"))
	}
}

func catalogHost(lower string) string {
	if !strings.Contains(lower, "://") {
		if i := strings.Index(lower, "/"); i > 0 {
			return lower[:i]
		}
		return lower
	}
	rest := lower
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if j := strings.Index(rest, "/"); j >= 0 {
		rest = rest[:j]
	}
	if k := strings.Index(rest, "?"); k >= 0 {
		rest = rest[:k]
	}
	return rest
}

func listTGCatalogPagesDue(path string, max int, rescanDays int) []tgCatalogPageEntry {
	f := readTGCatalogPageFile(path)
	if max <= 0 {
		max = 25
	}
	if rescanDays <= 0 {
		rescanDays = 14
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -rescanDays)
	var due []tgCatalogPageEntry
	for _, p := range f.Pages {
		if strings.TrimSpace(p.CrawledAt) == "" {
			due = append(due, p)
			continue
		}
		t, err := time.Parse(time.RFC3339, p.CrawledAt)
		if err != nil || t.Before(cutoff) {
			due = append(due, p)
		}
	}
	if len(due) > max {
		due = due[:max]
	}
	return due
}

func markTGCatalogPageCrawled(path string, pageURL string, handles int, crawlErr string) error {
	f := readTGCatalogPageFile(path)
	now := time.Now().UTC().Format(time.RFC3339)
	key := normalizeCatalogURL(pageURL)
	for i := range f.Pages {
		if normalizeCatalogURL(f.Pages[i].URL) != key {
			continue
		}
		f.Pages[i].CrawledAt = now
		f.Pages[i].HandlesFound = handles
		f.Pages[i].LastError = strings.TrimSpace(crawlErr)
		return writeTGCatalogPageFile(path, f)
	}
	return nil
}
