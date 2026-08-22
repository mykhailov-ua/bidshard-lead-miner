package entity

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// SemanticClusterID groups entities by shared pain vocabulary for ops audit.
func SemanticClusterID(matched []string, unifiedPain string) string {
	pain := strings.TrimSpace(strings.ToLower(unifiedPain))
	if pain != "" {
		sum := sha256.Sum256([]byte("pain:" + pain))
		return hex.EncodeToString(sum[:8])
	}
	if len(matched) == 0 {
		return ""
	}
	keys := append([]string(nil), matched...)
	sort.Strings(keys)
	if len(keys) > 5 {
		keys = keys[:5]
	}
	sum := sha256.Sum256([]byte("kw:" + strings.Join(keys, "|")))
	return hex.EncodeToString(sum[:8])
}
