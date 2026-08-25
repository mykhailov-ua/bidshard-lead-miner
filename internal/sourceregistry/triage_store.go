package sourceregistry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// LookupEntry returns a registry row by domain.
func LookupEntry(path, domain string) (Entry, bool) {
	f, err := Load(path)
	if err != nil {
		return Entry{}, false
	}
	domain = normalizeDomain(domain)
	for _, e := range f.Sources {
		if normalizeDomain(e.Domain) == domain {
			return e, true
		}
	}
	return Entry{}, false
}

// SetTriageStatus patches triage_status for one domain in the registry file.
func SetTriageStatus(path, domain, status string) error {
	f, err := Load(path)
	if err != nil {
		return err
	}
	domain = normalizeDomain(domain)
	updated := false
	for i := range f.Sources {
		if normalizeDomain(f.Sources[i].Domain) != domain {
			continue
		}
		setEntryTriageStatus(&f.Sources[i], status)
		updated = true
		break
	}
	if !updated {
		return nil
	}
	return Save(path, f)
}

func setEntryTriageStatus(entry *Entry, action string) {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case "keep", "drop", "defer":
		entry.TriageStatus = action
	default:
		entry.TriageStatus = "defer"
	}
}

// SetEntryTriageStatus updates triage_status on an in-memory entry.
func SetEntryTriageStatus(entry *Entry, action string) {
	setEntryTriageStatus(entry, action)
}

type triageCacheFile struct {
	Decisions map[string]string `json:"decisions"`
}

func ReadTriageCache(path string) triageCacheFile {
	raw, err := os.ReadFile(path)
	if err != nil {
		return triageCacheFile{Decisions: map[string]string{}}
	}
	var cache triageCacheFile
	if json.Unmarshal(raw, &cache) != nil || cache.Decisions == nil {
		return triageCacheFile{Decisions: map[string]string{}}
	}
	return cache
}

func WriteTriageCache(path string, cache triageCacheFile) error {
	if cache.Decisions == nil {
		cache.Decisions = map[string]string{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// ApplyTriageDecisions writes cache decisions onto registry entries; returns newly dropped count.
func ApplyTriageDecisions(path string, cache triageCacheFile) (int, error) {
	f, err := Load(path)
	if err != nil {
		return 0, err
	}
	dropped := 0
	for i := range f.Sources {
		domain := normalizeDomain(f.Sources[i].Domain)
		action, ok := cache.Decisions[domain]
		if !ok || action == "" {
			continue
		}
		prev := strings.ToLower(strings.TrimSpace(f.Sources[i].TriageStatus))
		setEntryTriageStatus(&f.Sources[i], action)
		if action == "drop" && prev != "drop" {
			dropped++
		}
	}
	if err := Save(path, f); err != nil {
		return 0, err
	}
	return dropped, nil
}

// PendingForGemini returns registry rows that still need Gemini triage.
func PendingForGemini(f File, cache triageCacheFile) []Entry {
	var out []Entry
	for _, entry := range f.Sources {
		domain := normalizeDomain(entry.Domain)
		if domain == "" {
			continue
		}
		meta := DomainMeta{
			Domain:       domain,
			Channel:      entry.Channel,
			Source:       entry.Source,
			DiscoveredBy: entry.DiscoveredBy,
		}
		if action, _, ok := HeuristicTriage(meta); ok {
			cache.Decisions[domain] = action
			continue
		}
		if cached, ok := cache.Decisions[domain]; ok && cached != "" && cached != "defer" {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(entry.TriageStatus))
		if status == "keep" || status == "drop" {
			continue
		}
		out = append(out, entry)
	}
	return out
}
