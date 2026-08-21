package sink

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/bidshard/parser/internal/extract"
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

// LeadHashIDWithScope mixes a dedup scope (e.g. tgweb site domain) into the contact hash.
// Prevents the same email on two affiliate sites from deduping to one lead.
func LeadHashIDWithScope(scope string, contacts []extract.Contact) string {
	base := LeadHashIDFromExtract(contacts)
	if base == "" {
		return ""
	}
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return base
	}
	sum := sha256.Sum256([]byte("scope:" + scope + "|" + base))
	return hex.EncodeToString(sum[:16])
}
