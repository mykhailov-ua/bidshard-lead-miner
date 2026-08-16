package dedup

import (
	"testing"
	"time"
)

func TestSeenCacheSkipsRepeat(t *testing.T) {
	t.Parallel()

	c := NewSeenCache(10, time.Hour)
	if c.Seen("abc") {
		t.Fatal("expected miss")
	}
	c.Mark("abc")
	if !c.Seen("abc") {
		t.Fatal("expected hit")
	}
}

func TestSeenCacheEvictsOldest(t *testing.T) {
	t.Parallel()

	c := NewSeenCache(2, time.Hour)
	c.Mark("a")
	c.Mark("b")
	c.Mark("c")
	if c.Seen("a") {
		t.Fatal("expected a evicted")
	}
	if !c.Seen("c") {
		t.Fatal("expected c present")
	}
}

func TestSeenCacheTTL(t *testing.T) {
	t.Parallel()

	c := NewSeenCache(10, 10*time.Millisecond)
	c.Mark("x")
	time.Sleep(15 * time.Millisecond)
	if c.Seen("x") {
		t.Fatal("expected ttl expiry")
	}
}
