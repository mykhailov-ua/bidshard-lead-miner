package bgworker

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSkipIfRunningDropsOverlap(t *testing.T) {
	t.Parallel()

	var runs atomic.Int32
	block := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	Run(ctx, &wg, []Job{
		{
			Name:          "slow",
			Interval:      20 * time.Millisecond,
			InitialDelay:  time.Millisecond,
			SkipIfRunning: true,
			Run: func(ctx context.Context) error {
				runs.Add(1)
				select {
				case <-block:
				case <-ctx.Done():
				}
				return nil
			},
		},
	})

	time.Sleep(60 * time.Millisecond)
	close(block)
	cancel()
	wg.Wait()

	if got := runs.Load(); got != 1 {
		t.Fatalf("runs=%d, want 1 (overlap ticks skipped)", got)
	}
}
