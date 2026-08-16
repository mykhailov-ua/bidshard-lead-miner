package app

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/pipeline"
)

func TestDrainCompletesWithinTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	taskCh := make(chan pipeline.Task, 4)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-taskCh:
				if !ok {
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	taskCh <- pipeline.Task{}
	cancel()

	start := time.Now()
	Drain(cancel, &wg, taskCh, 2*time.Second)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("drain took %v, want <= 2s", elapsed)
	}
}

func TestRunScanOnceExits(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.Config{
		ScanOnce:          true,
		WorkerCount:       2,
		TaskBuffer:        8,
		SourceConcurrency: 2,
		ScanTimeout:       2 * time.Second,
		HTTPTimeout:       time.Second,
		ShutdownTimeout:   2 * time.Second,
		Output:            "quiet",
		KeywordsJSONPath:  "../../testdata/keywords.json",
		KeywordsGrayPath:  "../../data/keywords-gray.json",
	}

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, cfg)
	}()

	select {
	case err := <-done:
		if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("Run did not return within timeout")
	}
}
