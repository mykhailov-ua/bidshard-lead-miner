package config

import (
	"fmt"
	"strings"
)

// KeywordOverlayPaths returns keyword overlay JSON paths from KEYWORDS_LOCALE_PATH or KEYWORDS_LOCALE.
// KEYWORDS_LOCALE supports comma-separated locales (es,pt,de) mapped to data/keywords-{locale}.json.
func KeywordOverlayPaths(locale, localePath string) []string {
	if path := strings.TrimSpace(localePath); path != "" {
		return []string{path}
	}
	raw := strings.TrimSpace(locale)
	if raw == "" {
		return nil
	}
	var out []string
	seen := map[string]struct{}{}
	for _, loc := range parseCSV(raw) {
		loc = strings.ToLower(strings.TrimSpace(loc))
		if loc == "" {
			continue
		}
		path := fmt.Sprintf("data/keywords-%s.json", loc)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}
