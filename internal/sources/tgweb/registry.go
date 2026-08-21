package tgweb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const DefaultDomainsPath = "data/runtime/discovered_telegram_domains.json"

type DomainEntry struct {
	Domain  string `json:"domain"`
	Channel string `json:"channel,omitempty"`
	Source  string `json:"source"`
	// Kind is discover provenance from the Telethon sidecar (about vs message body).
	Kind string `json:"kind,omitempty"`
	// DiscoveredVia is the seed channel that surfaced the domain.
	DiscoveredVia string `json:"discovered_via,omitempty"`
	// ForwardedFrom is legacy JSON from Tier B beta; normalized to DiscoveredVia on load.
	ForwardedFrom string `json:"forwarded_from,omitempty"`
	At            string `json:"at"`
	CrawledAt     string `json:"crawled_at,omitempty"`
}

type DomainFile struct {
	Domains []DomainEntry `json:"domains"`
}

func LoadDomains(path string) (DomainFile, error) {
	if path == "" {
		path = DefaultDomainsPath
	}
	var f DomainFile
	err := withRegistrySharedLock(path, func() error {
		var loadErr error
		f, loadErr = loadDomainsUnlocked(path)
		return loadErr
	})
	return f, err
}

func loadDomainsUnlocked(path string) (DomainFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DomainFile{}, nil
		}
		return DomainFile{}, err
	}
	var f DomainFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return DomainFile{}, err
	}
	normalizeDomainFile(&f)
	return f, nil
}

func normalizeDomainFile(f *DomainFile) {
	for i := range f.Domains {
		if f.Domains[i].DiscoveredVia == "" && f.Domains[i].ForwardedFrom != "" {
			f.Domains[i].DiscoveredVia = f.Domains[i].ForwardedFrom
		}
		f.Domains[i].ForwardedFrom = ""
	}
}

// SelectDomains returns pending domains or an explicit allowlist (ignores crawled_at for allowlist).
func SelectDomains(f DomainFile, limit int, rescanDays int, only []string) []DomainEntry {
	if len(only) > 0 {
		// --domains flag: crawl listed hosts even when crawled_at is fresh; order follows registry file.
		return selectOnlyDomains(f, only, limit)
	}
	return PendingDomains(f, limit, rescanDays)
}

// PendingDomains skips domains crawled within rescanDays; malformed crawled_at is treated as stale.
// Higher-priority entries (channel provenance, recent discover at) are crawled first when limit applies.
func PendingDomains(f DomainFile, limit int, rescanDays int) []DomainEntry {
	if rescanDays <= 0 {
		rescanDays = 30
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -rescanDays)
	now := time.Now().UTC()
	var candidates []DomainEntry
	for _, e := range f.Domains {
		domain := strings.ToLower(strings.TrimSpace(e.Domain))
		if domain == "" {
			continue
		}
		if e.CrawledAt != "" {
			if t, err := time.Parse(time.RFC3339, e.CrawledAt); err == nil && t.After(cutoff) {
				continue
			}
		}
		e.Domain = domain
		if !IsValidCrawlDomain(domain) {
			continue
		}
		candidates = append(candidates, e)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return domainPendingScore(candidates[i], now) > domainPendingScore(candidates[j], now)
	})
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

// domainPendingScore ranks registry rows for crawl order (not lead scoring).
func domainPendingScore(e DomainEntry, now time.Time) int {
	score := 0
	if strings.TrimSpace(e.Channel) != "" {
		score += 100
	}
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(e.At)); err == nil {
		days := int(now.Sub(t).Hours() / 24)
		if days < 0 {
			days = 0
		}
		if days > 30 {
			days = 30
		}
		score += (30 - days) * 2
	}
	src := strings.ToLower(strings.TrimSpace(e.Source))
	if strings.Contains(src, "cross") || strings.Contains(src, "mention") {
		score += 10
	}
	switch strings.ToLower(strings.TrimSpace(e.Kind)) {
	case "mentioned_in_about":
		// Channel about links are higher-signal than one-off message URLs.
		score += 15
	case "mentioned_in_message":
		score += 5
	}
	if strings.TrimSpace(e.DiscoveredVia) != "" {
		// Seed channel provenance; small tie-break when kind scores tie.
		score += 5
	}
	return score
}

func selectOnlyDomains(f DomainFile, only []string, limit int) []DomainEntry {
	want := make(map[string]struct{}, len(only))
	for _, d := range only {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			want[d] = struct{}{}
		}
	}
	if len(want) == 0 {
		return nil
	}
	// Do not inject domains missing from the registry file; allowlist is a filter, not a seed.
	var out []DomainEntry
	for _, e := range f.Domains {
		domain := strings.ToLower(strings.TrimSpace(e.Domain))
		if _, ok := want[domain]; !ok {
			continue
		}
		if !IsValidCrawlDomain(domain) {
			continue
		}
		e.Domain = domain
		out = append(out, e)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// ClearCrawledAt removes crawled_at for the given domains.
func ClearCrawledAt(path string, domains []string) (int, error) {
	if path == "" {
		path = DefaultDomainsPath
	}
	want := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		if d != "" {
			want[d] = struct{}{}
		}
	}
	if len(want) == 0 {
		return 0, nil
	}
	var updated int
	err := withRegistryExclusiveLock(path, func() error {
		f, loadErr := loadDomainsUnlocked(path)
		if loadErr != nil {
			return loadErr
		}
		for i := range f.Domains {
			domain := strings.ToLower(strings.TrimSpace(f.Domains[i].Domain))
			if _, ok := want[domain]; !ok {
				continue
			}
			if f.Domains[i].CrawledAt != "" {
				f.Domains[i].CrawledAt = ""
				updated++
			}
		}
		if updated == 0 {
			return nil
		}
		return writeDomainsUnlocked(path, f)
	})
	if err != nil {
		return 0, err
	}
	return updated, nil
}

func MarkCrawled(path string, domain string) error {
	if path == "" {
		path = DefaultDomainsPath
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	return withRegistryExclusiveLock(path, func() error {
		f, err := loadDomainsUnlocked(path)
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		updated := false
		for i := range f.Domains {
			if strings.EqualFold(f.Domains[i].Domain, domain) {
				f.Domains[i].CrawledAt = now
				updated = true
				break
			}
		}
		if !updated {
			return nil
		}
		return writeDomainsUnlocked(path, f)
	})
}

func writeDomainsUnlocked(path string, f DomainFile) error {
	normalizeDomainFile(&f)
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
	// Atomic replace so readers never see a partial JSON object mid-write.
	return os.Rename(tmp, path)
}

// PruneInvalidDomains removes blocked, file-like, and invalid TLD hosts from the registry file.
func PruneInvalidDomains(path string) (kept, removed int, err error) {
	if path == "" {
		path = DefaultDomainsPath
	}
	err = withRegistryExclusiveLock(path, func() error {
		f, loadErr := loadDomainsUnlocked(path)
		if loadErr != nil {
			return loadErr
		}
		var valid []DomainEntry
		for _, e := range f.Domains {
			domain := strings.ToLower(strings.TrimSpace(e.Domain))
			if domain == "" || !IsValidCrawlDomain(domain) {
				removed++
				continue
			}
			e.Domain = domain
			valid = append(valid, e)
		}
		kept = len(valid)
		if removed == 0 {
			return nil
		}
		f.Domains = valid
		return writeDomainsUnlocked(path, f)
	})
	if err != nil {
		return 0, 0, err
	}
	return kept, removed, nil
}
