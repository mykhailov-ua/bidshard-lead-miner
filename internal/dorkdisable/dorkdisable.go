package dorkdisable

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultPath = "data/runtime/disabled_dorks.json"

type file struct {
	Updated string   `json:"updated"`
	Dorks   []string `json:"dorks"`
	Audit   []string `json:"audit,omitempty"`
}

// Load returns disabled SERP dork queries (case-fold keys).
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
	seen := make(map[string]struct{}, len(f.Dorks))
	for _, d := range f.Dorks {
		key := normalizeDorkKey(d)
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	return seen
}

// Save merges new disabled dorks into the registry file.
func Save(path string, disabled []string, audit []string) error {
	if path == "" {
		path = DefaultPath
	}
	existing := Load(path)
	for _, d := range disabled {
		key := normalizeDorkKey(d)
		if key != "" {
			existing[key] = struct{}{}
		}
	}
	var dorks []string
	for key := range existing {
		dorks = append(dorks, key)
	}
	f := file{
		Updated: time.Now().UTC().Format(time.RFC3339),
		Dorks:   dorks,
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

// IsDisabled reports whether a SERP dork query is disabled.
func IsDisabled(path, dork string) bool {
	key := normalizeDorkKey(dork)
	if key == "" {
		return false
	}
	_, ok := Load(path)[key]
	return ok
}

// CountDisabled returns how many dorks are disabled in the registry file.
func CountDisabled(path string) int {
	return len(Load(path))
}

// FilterActiveDorks drops disabled queries from a harvest list.
func FilterActiveDorks(path string, dorks []string) []string {
	if len(dorks) == 0 {
		return nil
	}
	disabled := Load(path)
	if len(disabled) == 0 {
		return dorks
	}
	out := make([]string, 0, len(dorks))
	for _, dork := range dorks {
		if _, off := disabled[normalizeDorkKey(dork)]; off {
			continue
		}
		out = append(out, dork)
	}
	return out
}

func normalizeDorkKey(dork string) string {
	return strings.ToLower(strings.TrimSpace(dork))
}
