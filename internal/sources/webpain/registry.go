package webpain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultRegistryPath = "data/runtime/discovered_web_pain.json"

// Entry is a SERP hit queued for generic page fetch (non-forum thread URL).
type Entry struct {
	URL     string `json:"url"`
	Host    string `json:"host,omitempty"`
	Source  string `json:"source"`
	Query   string `json:"query,omitempty"`
	At      string `json:"at"`
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// Discovery is SERP metadata before registry write.
type Discovery struct {
	URL     string
	Title   string
	Snippet string
}

type File struct {
	URLs []Entry `json:"urls"`
}

func LoadRegistry(path string) (File, error) {
	if path == "" {
		path = DefaultRegistryPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(raw, &f); err != nil {
		return File{}, err
	}
	return f, nil
}

func SaveRegistry(path string, f File) error {
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

// AppendDiscoveries dedupes and appends open-web SERP hits.
func AppendDiscoveries(path, source, query string, items []Discovery) (added int, err error) {
	if len(items) == 0 {
		return 0, nil
	}
	f, err := LoadRegistry(path)
	if err != nil {
		return 0, err
	}
	seen := make(map[string]struct{}, len(f.URLs))
	for _, e := range f.URLs {
		if u := normalizeURL(e.URL); u != "" {
			seen[u] = struct{}{}
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, item := range items {
		u := normalizeURL(item.URL)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		f.URLs = append(f.URLs, Entry{
			URL:     u,
			Host:    hostFromURL(u),
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
	return added, SaveRegistry(path, f)
}

func normalizeURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	rawURL = strings.TrimSuffix(rawURL, "/")
	return rawURL
}

func hostFromURL(rawURL string) string {
	// Avoid importing net/url in hot path tests; adapter uses full url.Parse.
	idx := strings.Index(rawURL, "//")
	if idx < 0 {
		return ""
	}
	rest := rawURL[idx+2:]
	if slash := strings.Index(rest, "/"); slash >= 0 {
		rest = rest[:slash]
	}
	return strings.ToLower(strings.TrimPrefix(rest, "www."))
}
