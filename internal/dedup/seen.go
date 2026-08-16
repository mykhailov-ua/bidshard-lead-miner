package dedup

import (
	"sync"
	"time"
)

type entry struct {
	seenAt time.Time
}

type SeenCache struct {
	mu      sync.Mutex
	items   map[string]entry
	order   []string
	maxSize int
	ttl     time.Duration
}

func NewSeenCache(maxSize int, ttl time.Duration) *SeenCache {
	if maxSize <= 0 {
		maxSize = 50_000
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &SeenCache{
		items:   make(map[string]entry, 1024),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

func (c *SeenCache) Seen(hashID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.evictExpiredLocked(time.Now())

	e, ok := c.items[hashID]
	return ok && time.Since(e.seenAt) <= c.ttl
}

func (c *SeenCache) Mark(hashID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.evictExpiredLocked(now)

	if _, ok := c.items[hashID]; !ok {
		c.order = append(c.order, hashID)
	}
	c.items[hashID] = entry{seenAt: now}

	for len(c.items) > c.maxSize {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.items, oldest)
	}
}

func (c *SeenCache) evictExpiredLocked(now time.Time) {
	if len(c.items) == 0 {
		return
	}
	kept := c.order[:0]
	for _, id := range c.order {
		e, ok := c.items[id]
		if !ok {
			continue
		}
		if now.Sub(e.seenAt) > c.ttl {
			delete(c.items, id)
			continue
		}
		kept = append(kept, id)
	}
	c.order = kept
}
