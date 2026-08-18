package scoring

import "strings"

// KeywordIDsFromMatched resolves keyword ids from scored match summary strings.
func (r *Registry) KeywordIDsFromMatched(matched []string) []string {
	if r == nil || len(matched) == 0 {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := map[string]struct{}{}
	var ids []string
	for _, m := range matched {
		phrase := matchedPhrase(m)
		id, ok := r.phraseSet[NormalizePhrase(phrase)]
		if !ok || id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func matchedPhrase(s string) string {
	if i := strings.LastIndex(s, "(+"); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	if i := strings.LastIndex(s, "(-"); i > 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
