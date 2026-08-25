package lander

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type landerPathsCacheFile struct {
	Domains map[string]landerPathsCacheEntry `json:"domains"`
}

type landerPathsCacheEntry struct {
	Paths        []string `json:"paths"`
	DiscoveredAt string   `json:"discovered_at"`
}

var landerPathsCacheMu sync.Mutex

func loadLanderPathsCache(path, domain string) ([]string, bool) {
	landerPathsCacheMu.Lock()
	defer landerPathsCacheMu.Unlock()

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var file landerPathsCacheFile
	if err := json.Unmarshal(raw, &file); err != nil || file.Domains == nil {
		return nil, false
	}
	entry, ok := file.Domains[domain]
	if !ok || len(entry.Paths) == 0 {
		return nil, false
	}
	if t, err := time.Parse(time.RFC3339, entry.DiscoveredAt); err == nil {
		if time.Since(t) > 7*24*time.Hour {
			return nil, false
		}
	}
	return append([]string(nil), entry.Paths...), true
}

func saveLanderPathsCache(path, domain string, paths []string) error {
	if domain == "" || len(paths) == 0 {
		return nil
	}
	landerPathsCacheMu.Lock()
	defer landerPathsCacheMu.Unlock()

	file := landerPathsCacheFile{Domains: map[string]landerPathsCacheEntry{}}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &file)
	}
	if file.Domains == nil {
		file.Domains = map[string]landerPathsCacheEntry{}
	}
	file.Domains[domain] = landerPathsCacheEntry{
		Paths:        append([]string(nil), paths...),
		DiscoveredAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}
