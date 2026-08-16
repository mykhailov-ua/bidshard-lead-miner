package sink

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// LeadHashID returns a stable dedup key from normalized contacts (not source/title).
func LeadHashID(contacts []StoredContact) string {
	var parts []string
	for _, c := range contacts {
		parts = append(parts, c.Type+":"+strings.ToLower(strings.TrimSpace(c.Value)))
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}
