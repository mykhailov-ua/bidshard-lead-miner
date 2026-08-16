package limit

import (
	"context"
	"testing"
	"time"
)

func TestHostLimiterWaits(t *testing.T) {
	t.Parallel()

	lim := NewHostLimiters(5, 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	_ = lim.Wait(ctx, "example.com")
	_ = lim.Wait(ctx, "example.com")
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("expected rate limit wait, got %v", elapsed)
	}
}
