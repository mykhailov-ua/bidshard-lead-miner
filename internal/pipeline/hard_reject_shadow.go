package pipeline

import (
	"hash/fnv"
)

// ShouldSampleHardRejectShadow returns true for ~pct% of hard_reject rows (deterministic by text).
func ShouldSampleHardRejectShadow(text string, pct int) bool {
	if pct <= 0 {
		return false
	}
	if pct >= 100 {
		return true
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(text))
	return int(h.Sum32()%100) < pct
}
