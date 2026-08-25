package sourceregistry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const DefaultPath = "data/runtime/source_registry.json"

const (
	TypeTGWeb  = "tgweb"
	TypeSupply = "supply"
	TypeLander = "lander"
)

// Entry is one discovered domain with crawl routing metadata.
type Entry struct {
	Domain       string   `json:"domain"`
	Types        []string `json:"types"`
	DiscoveredBy string   `json:"discovered_by"`
	LastSeen     string   `json:"last_seen"`
	TriageStatus string   `json:"triage_status,omitempty"`
	Channel      string   `json:"channel,omitempty"`
	Source       string   `json:"source,omitempty"`
	LastTriageAt string   `json:"last_triage_at,omitempty"`
	TriageWhy    string   `json:"triage_why,omitempty"`
}

type File struct {
	Sources []Entry `json:"sources"`
}

func Load(path string) (File, error) {
	if path == "" {
		path = DefaultPath
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

func Save(path string, f File) error {
	if path == "" {
		path = DefaultPath
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

// Upsert merges types and provenance for domain; skipped when triage_status=drop.
func Upsert(path string, in Entry) (added bool, err error) {
	domain := normalizeDomain(in.Domain)
	if domain == "" {
		return false, nil
	}
	f, err := Load(path)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	types := normalizeTypes(in.Types)
	if len(types) == 0 {
		types = []string{TypeTGWeb}
	}

	byDomain := make(map[string]int, len(f.Sources))
	for i, e := range f.Sources {
		byDomain[normalizeDomain(e.Domain)] = i
	}
	if idx, ok := byDomain[domain]; ok {
		existing := &f.Sources[idx]
		if strings.EqualFold(strings.TrimSpace(existing.TriageStatus), "drop") {
			return false, nil
		}
		merged := mergeTypes(existing.Types, types)
		existing.Types = merged
		existing.LastSeen = now
		if in.DiscoveredBy != "" {
			existing.DiscoveredBy = in.DiscoveredBy
		}
		if in.Channel != "" && existing.Channel == "" {
			existing.Channel = in.Channel
		}
		if in.Source != "" && existing.Source == "" {
			existing.Source = in.Source
		}
		if in.TriageStatus != "" {
			existing.TriageStatus = in.TriageStatus
		}
		return false, Save(path, f)
	}

	f.Sources = append(f.Sources, Entry{
		Domain:       domain,
		Types:        types,
		DiscoveredBy: in.DiscoveredBy,
		LastSeen:     now,
		TriageStatus: in.TriageStatus,
		Channel:      in.Channel,
		Source:       in.Source,
	})
	return true, Save(path, f)
}

// ListDomainsByType returns deduped domains registered for crawl type t (skip triage drop).
func ListDomainsByType(path, typ string) ([]string, error) {
	f, err := Load(path)
	if err != nil {
		return nil, err
	}
	typ = strings.ToLower(strings.TrimSpace(typ))
	seen := make(map[string]struct{})
	var out []string
	for _, e := range f.Sources {
		if strings.EqualFold(strings.TrimSpace(e.TriageStatus), "drop") {
			continue
		}
		if !hasType(e.Types, typ) {
			continue
		}
		domain := normalizeDomain(e.Domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		out = append(out, domain)
	}
	sort.Strings(out)
	return out, nil
}

func CountByType(path, typ string) (int, error) {
	domains, err := ListDomainsByType(path, typ)
	return len(domains), err
}

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")
	if idx := strings.Index(domain, "/"); idx >= 0 {
		domain = domain[:idx]
	}
	return domain
}

// NormalizeDomain lowercases and strips scheme/path for registry keys.
func NormalizeDomain(domain string) string {
	return normalizeDomain(domain)
}

func normalizeTypes(types []string) []string {
	seen := make(map[string]struct{}, len(types))
	var out []string
	for _, t := range types {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func mergeTypes(existing, add []string) []string {
	seen := make(map[string]struct{})
	for _, t := range existing {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			seen[t] = struct{}{}
		}
	}
	for _, t := range add {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			seen[t] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func hasType(types []string, want string) bool {
	for _, t := range types {
		if strings.EqualFold(strings.TrimSpace(t), want) {
			return true
		}
	}
	return false
}
