package entity

import "strings"

// Key kind constants ordered by resolution priority (strongest first).
const (
	KindCompany         = "company"
	KindDomain          = "domain"
	KindForumUID        = "forum_uid"
	KindForumUser       = "forum_user"
	KindTelegram        = "telegram"
	KindTelegramChannel = "channel"
)

// EntityKey is one normalized identity atom for cross-source linking.
type EntityKey struct {
	Kind  string
	Value string
}

// Canonical returns a stable lookup token kind:value.
func (k EntityKey) Canonical() string {
	return k.Kind + ":" + k.Value
}

var keyPriority = map[string]int{
	KindCompany:         0,
	KindDomain:          1,
	KindForumUID:        2,
	KindForumUser:       3,
	KindTelegram:        4,
	KindTelegramChannel: 5,
}

func keyLess(a, b EntityKey) bool {
	pa, pb := keyPriority[a.Kind], keyPriority[b.Kind]
	if pa != pb {
		return pa < pb
	}
	return a.Value < b.Value
}

func dedupeKeys(keys []EntityKey) []EntityKey {
	if len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]EntityKey, 0, len(keys))
	for _, k := range keys {
		if k.Kind == "" || k.Value == "" {
			continue
		}
		canon := k.Canonical()
		if _, ok := seen[canon]; ok {
			continue
		}
		seen[canon] = struct{}{}
		out = append(out, k)
	}
	return out
}

func sortKeys(keys []EntityKey) {
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keyLess(keys[j], keys[i]) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
}

func isOrgLikeName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 3 {
		return false
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, "@") {
		return false
	}
	// Skip obvious personal names: two short title-case tokens only.
	parts := strings.Fields(name)
	if len(parts) == 2 {
		if len(parts[0]) <= 12 && len(parts[1]) <= 12 {
			allTitle := true
			for _, p := range parts {
				if len(p) == 0 {
					allTitle = false
					break
				}
				r := []rune(p)
				if len(r) > 0 && r[0] >= 'a' && r[0] <= 'z' {
					allTitle = false
					break
				}
			}
			if allTitle && !strings.Contains(lower, "network") &&
				!strings.Contains(lower, "media") &&
				!strings.Contains(lower, "affiliate") &&
				!strings.Contains(lower, "cpa") {
				return false
			}
		}
	}
	return true
}
