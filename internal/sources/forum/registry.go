package forum

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultRegistryPath = "data/runtime/discovered_forum_threads.json"

type ThreadEntry struct {
	URL     string `json:"url"`
	Host    string `json:"host,omitempty"`
	Source  string `json:"source"`
	Query   string `json:"query,omitempty"`
	At      string `json:"at"`
	Notes   string `json:"notes,omitempty"`
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// ThreadDiscovery is SERP or manual metadata stored before HTTP fetch.
type ThreadDiscovery struct {
	URL     string
	Title   string
	Snippet string
}

type ThreadFile struct {
	Threads []ThreadEntry `json:"threads"`
}

func LoadThreadRegistry(path string) (ThreadFile, error) {
	if path == "" {
		path = DefaultRegistryPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ThreadFile{}, nil
		}
		return ThreadFile{}, err
	}
	var f ThreadFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return ThreadFile{}, err
	}
	return f, nil
}

func SaveThreadRegistry(path string, f ThreadFile) error {
	if path == "" {
		path = DefaultRegistryPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AppendThreadURLs dedupes and appends forum thread URLs discovered by SERP or other jobs.
func AppendThreadURLs(path, source, query string, urls []string) (added int, err error) {
	items := make([]ThreadDiscovery, 0, len(urls))
	for _, u := range urls {
		items = append(items, ThreadDiscovery{URL: u})
	}
	return AppendThreadDiscoveries(path, source, query, items)
}

// AppendThreadDiscoveries dedupes and appends forum threads with optional SERP title/snippet.
func AppendThreadDiscoveries(path, source, query string, items []ThreadDiscovery) (added int, err error) {
	if len(items) == 0 {
		return 0, nil
	}
	f, err := LoadThreadRegistry(path)
	if err != nil {
		return 0, err
	}
	seen := make(map[string]struct{}, len(f.Threads))
	for _, e := range f.Threads {
		if u := normalizeThreadURL(e.URL); u != "" {
			seen[u] = struct{}{}
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, item := range items {
		u := normalizeThreadURL(item.URL)
		if u == "" || !IsForumThreadURL(u) {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		f.Threads = append(f.Threads, ThreadEntry{
			URL:     u,
			Host:    hostFromThreadURL(u),
			Source:  source,
			Query:   query,
			At:      now,
			Title:   strings.TrimSpace(item.Title),
			Snippet: strings.TrimSpace(item.Snippet),
		})
		added++
	}
	if added == 0 {
		return 0, nil
	}
	return added, SaveThreadRegistry(path, f)
}

// IsForumThreadURL reports whether rawURL points at a thread detail page on a known forum host.
func IsForumThreadURL(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Host)
	if !IsKnownForumHost(host) {
		return false
	}
	path := strings.ToLower(u.Path)
	query := strings.ToLower(u.RawQuery)
	switch {
	case strings.Contains(path, "/threads/"):
		return true
	case strings.Contains(path, "/t/"):
		return true
	case strings.Contains(path, "thread-"):
		return true
	case strings.Contains(query, "t="):
		return true
	case strings.Contains(host, "blackhatworld.com") && strings.Contains(path, "/seo/"):
		return true
	default:
		return false
	}
}

func normalizeThreadURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	u.Fragment = ""
	return strings.TrimSuffix(u.String(), "/")
}

func hostFromThreadURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}
