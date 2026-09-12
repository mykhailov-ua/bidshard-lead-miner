package gemini

import (
	"testing"
	"time"
)

func TestBackoffDuration_hasJitter(t *testing.T) {
	c := &Client{limits: ModelLimits{RetryBase: time.Second, RetryMax: 30 * time.Second}}
	seen := make(map[time.Duration]struct{})
	for i := 0; i < 20; i++ {
		d := c.backoffDuration(1, 0)
		if d < 2*time.Second || d > 3*time.Second {
			t.Fatalf("unexpected backoff %v", d)
		}
		seen[d] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("expected jitter variance, got single duration %v", seen)
	}
}
