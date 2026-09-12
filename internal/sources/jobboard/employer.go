package jobboard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/entity"
)

const DefaultEmployerRegistryPath = "data/runtime/discovered_employers.json"

// EmployerEntry is a media-buying team surfaced from job boards or crawl.
type EmployerEntry struct {
	Name          string   `json:"name"`
	Slug          string   `json:"slug,omitempty"`
	Normalized    string   `json:"normalized,omitempty"`
	Hosts         []string `json:"hosts,omitempty"`
	URLs          []string `json:"urls,omitempty"`
	Source        string   `json:"source,omitempty"`
	Query         string   `json:"query,omitempty"`
	At            string   `json:"at,omitempty"`
	LastReverseAt string   `json:"last_reverse_at,omitempty"`
}

type EmployerFile struct {
	Employers []EmployerEntry `json:"employers"`
}

func LoadEmployers(path string) (EmployerFile, error) {
	if path == "" {
		path = DefaultEmployerRegistryPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return EmployerFile{}, nil
		}
		return EmployerFile{}, err
	}
	var f EmployerFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return EmployerFile{}, err
	}
	return f, nil
}

func SaveEmployers(path string, f EmployerFile) error {
	if path == "" {
		path = DefaultEmployerRegistryPath
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

// UpsertEmployer merges employer metadata and returns true when a new row was added.
func UpsertEmployer(path string, in EmployerEntry) (bool, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return false, nil
	}
	normalized := strings.TrimSpace(in.Normalized)
	if normalized == "" {
		normalized = entity.NormalizeCompany(name)
	}
	if normalized == "" || !entity.IsOrgLikeName(name) {
		return false, nil
	}
	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = normalized
	}

	f, err := LoadEmployers(path)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	key := normalized
	byKey := make(map[string]int, len(f.Employers))
	for i, e := range f.Employers {
		k := strings.TrimSpace(e.Normalized)
		if k == "" {
			k = entity.NormalizeCompany(e.Name)
		}
		if k != "" {
			byKey[k] = i
		}
	}
	if idx, ok := byKey[key]; ok {
		existing := &f.Employers[idx]
		if existing.Name == "" || len(name) > len(existing.Name) {
			existing.Name = name
		}
		if slug != "" && existing.Slug == "" {
			existing.Slug = slug
		}
		existing.Hosts = mergeStrings(existing.Hosts, in.Hosts)
		existing.URLs = mergeStrings(existing.URLs, in.URLs)
		if in.Source != "" {
			existing.Source = in.Source
		}
		if in.Query != "" {
			existing.Query = in.Query
		}
		if existing.At == "" {
			existing.At = now
		}
		return false, SaveEmployers(path, f)
	}

	f.Employers = append(f.Employers, EmployerEntry{
		Name:       name,
		Slug:       slug,
		Normalized: normalized,
		Hosts:      dedupeStrings(in.Hosts),
		URLs:       dedupeStrings(in.URLs),
		Source:     in.Source,
		Query:      in.Query,
		At:         now,
	})
	return true, SaveEmployers(path, f)
}

// MarkEmployerReversed stamps last_reverse_at for normalized employer key.
func MarkEmployerReversed(path, normalized string, at time.Time) error {
	normalized = entity.NormalizeCompany(normalized)
	if normalized == "" {
		return nil
	}
	f, err := LoadEmployers(path)
	if err != nil {
		return err
	}
	stamp := at.UTC().Format(time.RFC3339)
	for i := range f.Employers {
		key := strings.TrimSpace(f.Employers[i].Normalized)
		if key == "" {
			key = entity.NormalizeCompany(f.Employers[i].Name)
		}
		if key == normalized {
			f.Employers[i].LastReverseAt = stamp
			return SaveEmployers(path, f)
		}
	}
	return nil
}

// EmployerFromPage builds an employer row from a parsed job-board page.
func EmployerFromPage(page Page, rawURL, source, query string) EmployerEntry {
	name := strings.TrimSpace(page.Company)
	if name == "" {
		name = employerNameFromSlug(page.Slug)
	}
	return EmployerEntry{
		Name:       name,
		Slug:       strings.TrimSpace(page.Slug),
		Normalized: entity.NormalizeCompany(name),
		Hosts:      []string{HostFromURL(rawURL)},
		URLs:       []string{NormalizeURL(rawURL)},
		Source:     source,
		Query:      query,
	}
}

// EmployerFromRegistryEntry infers employer metadata from a queued job URL.
func EmployerFromRegistryEntry(url, title, snippet, source, query string) EmployerEntry {
	name := EmployerNameFromTitle(title)
	slug := CompanySlug(url)
	if name == "" {
		name = employerNameFromSlug(slug)
	}
	return EmployerEntry{
		Name:       name,
		Slug:       slug,
		Normalized: entity.NormalizeCompany(name),
		Hosts:      []string{HostFromURL(url)},
		URLs:       []string{NormalizeURL(url)},
		Source:     source,
		Query:      query,
	}
}

// EmployerNameFromTitle extracts company names from DOU/Djinni SERP titles.
func EmployerNameFromTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	lower := strings.ToLower(title)
	for _, sep := range []string{" в ", " at ", " | dou", " – djinni", " - djinni"} {
		if idx := strings.LastIndex(lower, sep); idx >= 0 {
			name := strings.TrimSpace(title[:idx])
			name = strings.TrimSuffix(name, ",")
			if i := strings.LastIndex(name, " - "); i >= 0 {
				name = strings.TrimSpace(name[i+3:])
			}
			return strings.TrimSpace(name)
		}
	}
	return ""
}

// ReverseDorks builds open-web queries to find TG channels and public traces for a team.
func ReverseDorks(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	quoted := `"` + name + `"`
	return []string{
		"site:t.me " + quoted + " arbitrage",
		"site:t.me " + quoted + " media buying",
		"site:t.me " + quoted + " keitaro",
		quoted + " telegram media buyer",
		quoted + " media buying team",
	}
}

// EmployersDueForReverse returns employers not reverse-searched within rescanDays.
func EmployersDueForReverse(f EmployerFile, rescanDays int, limit int) []EmployerEntry {
	if limit <= 0 {
		limit = 15
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -rescanDays)
	var out []EmployerEntry
	for _, e := range f.Employers {
		if strings.TrimSpace(e.Name) == "" {
			continue
		}
		if e.LastReverseAt != "" {
			if ts, err := time.Parse(time.RFC3339, e.LastReverseAt); err == nil && ts.After(cutoff) {
				continue
			}
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out
}

// SyncEmployersFromJobRegistry upserts employers inferred from discovered_job_urls.json.
func SyncEmployersFromJobRegistry(jobRegistryPath, employerRegistryPath string) (added int, err error) {
	jobs, err := LoadRegistry(jobRegistryPath)
	if err != nil {
		return 0, err
	}
	for _, entry := range jobs.URLs {
		emp := EmployerFromRegistryEntry(entry.URL, entry.Title, entry.Snippet, entry.Source, entry.Query)
		if strings.TrimSpace(emp.Name) == "" {
			continue
		}
		ok, upsertErr := UpsertEmployer(employerRegistryPath, emp)
		if upsertErr != nil {
			return added, upsertErr
		}
		if ok {
			added++
		}
	}
	return added, nil
}

func employerNameFromSlug(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

func mergeStrings(existing, add []string) []string {
	return dedupeStrings(append(append([]string(nil), existing...), add...))
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
