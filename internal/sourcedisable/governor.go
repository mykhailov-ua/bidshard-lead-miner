package sourcedisable

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
)

const DefaultPath = "data/runtime/disabled_sources.json"

type file struct {
	Updated string   `json:"updated"`
	Sources []string `json:"sources"`
	Audit   []string `json:"audit,omitempty"`
}

// Evaluate returns crawl family keys to disable when accepted=0 and raw (accepted+junk) >= minRaw.
func Evaluate(stats []sink.SourceStatsDoc, minRaw int) []string {
	if minRaw <= 0 {
		minRaw = 100
	}
	var out []string
	for _, doc := range stats {
		key := strings.TrimSpace(doc.Source)
		if key == "" {
			continue
		}
		threshold := MinRawForFamily(key, minRaw)
		total := doc.Accepted + doc.Junk
		if total < threshold || doc.Accepted > 0 {
			continue
		}
		out = append(out, key)
	}
	return out
}

// MinRawForFamily lowers the governor bar for chronically noisy crawl families.
func MinRawForFamily(family string, defaultMin int) int {
	switch strings.ToLower(strings.TrimSpace(family)) {
	case "lander", "github", "webpain":
		if defaultMin > 40 {
			return 40
		}
	}
	if strings.HasPrefix(strings.ToLower(family), "telegram:invite") && defaultMin > 40 {
		return 40
	}
	return defaultMin
}

func Load(path string) map[string]struct{} {
	if path == "" {
		path = DefaultPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]struct{}{}
	}
	var f file
	if json.Unmarshal(raw, &f) != nil {
		return map[string]struct{}{}
	}
	seen := make(map[string]struct{}, len(f.Sources))
	for _, s := range f.Sources {
		s = strings.TrimSpace(s)
		if s != "" {
			seen[s] = struct{}{}
		}
	}
	return seen
}

func Save(path string, disabled []string, audit []string) error {
	if path == "" {
		path = DefaultPath
	}
	existing := Load(path)
	for _, s := range disabled {
		existing[strings.TrimSpace(s)] = struct{}{}
	}
	var sources []string
	for s := range existing {
		sources = append(sources, s)
	}
	f := file{
		Updated: time.Now().UTC().Format(time.RFC3339),
		Sources: sources,
		Audit:   audit,
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

// IsDisabled matches exact family names and prefixes (e.g. forum blocks forum:affiliatefix.com).
func IsDisabled(path, source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return false
	}
	for key := range Load(path) {
		key = strings.ToLower(key)
		if source == key || strings.HasPrefix(source, key+":") {
			return true
		}
	}
	return false
}
