package scoring

import (
	"strings"
	"sync"
)

// SourceReputation tracks per-source accepted/junk ratios and applies boost weights.
type SourceReputation struct {
	mu       sync.RWMutex
	counts   map[string]sourceCounts
	mongo    SourceStatsWriter
	maxBoost int
}

type sourceCounts struct {
	Accepted int
	Junk     int
}

type SourceStatsWriter interface {
	RecordAccepted(source string)
	RecordJunk(source string)
	Boost(source string) int
}

func NewSourceReputation(mongo SourceStatsWriter) *SourceReputation {
	return &SourceReputation{
		counts:   make(map[string]sourceCounts),
		mongo:    mongo,
		maxBoost: 8,
	}
}

func (r *SourceReputation) RecordAccepted(source string) {
	if r == nil {
		return
	}
	key := normalizeSourceKey(source)
	r.mu.Lock()
	c := r.counts[key]
	c.Accepted++
	r.counts[key] = c
	r.mu.Unlock()
	if r.mongo != nil {
		r.mongo.RecordAccepted(key)
	}
}

func (r *SourceReputation) RecordJunk(source string) {
	if r == nil {
		return
	}
	key := normalizeSourceKey(source)
	r.mu.Lock()
	c := r.counts[key]
	c.Junk++
	r.counts[key] = c
	r.mu.Unlock()
	if r.mongo != nil {
		r.mongo.RecordJunk(key)
	}
}

func (r *SourceReputation) Boost(source string) int {
	base := StaticSourceBoost(source)
	dynamic := 0
	if r != nil {
		r.mu.RLock()
		c := r.counts[normalizeSourceKey(source)]
		r.mu.RUnlock()
		dynamic = dynamicBoost(c, r.maxBoost)
		if r.mongo != nil {
			if b := r.mongo.Boost(normalizeSourceKey(source)); b > dynamic {
				dynamic = b
			}
		}
	}
	return base + dynamic
}

func StaticSourceBoost(source string) int {
	lower := strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.HasPrefix(lower, "telegram:") && strings.Contains(lower, "igaming"):
		return 5
	case strings.HasPrefix(lower, "forum:") && strings.Contains(lower, "stm"):
		return 10
	case strings.HasPrefix(lower, "ads_txt:"):
		return 8
	case strings.HasPrefix(lower, "reddit:"):
		return 3
	case strings.HasPrefix(lower, "discord:"):
		return 6
	case strings.HasPrefix(lower, "lander:"):
		return 0
	default:
		return 0
	}
}

func dynamicBoost(c sourceCounts, max int) int {
	total := c.Accepted + c.Junk
	if total < 20 {
		return 0
	}
	ratio := float64(c.Accepted) / float64(total)
	switch {
	case ratio >= 0.35:
		return max
	case ratio >= 0.2:
		return max / 2
	case ratio < 0.08:
		return -max / 2
	default:
		return 0
	}
}

func normalizeSourceKey(source string) string {
	lower := strings.ToLower(strings.TrimSpace(source))
	if idx := strings.Index(lower, "/"); idx > 0 {
		return lower[:idx]
	}
	return lower
}
