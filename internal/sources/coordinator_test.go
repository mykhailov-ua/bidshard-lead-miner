package sources

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/pipeline"
)

func TestCoordinatorCancelsPreviousRound(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		WorkerCount:       1,
		TaskBuffer:        4,
		SourceConcurrency: 1,
		ScanTimeout:       5 * time.Second,
		HTTPTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
	}

	blocking := NewBlockingStubSource("stub:blocking")
	coordinator := NewCoordinator(cfg, []Source{blocking})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	taskCh := make(chan pipeline.Task, 4)
	scanCh := make(chan struct{}, 1)
	statsCh := make(chan pipeline.RoundStats, 4)

	var wg sync.WaitGroup
	coordinator.Run(ctx, &wg, scanCh, taskCh, statsCh)

	scanCh <- struct{}{}
	time.Sleep(50 * time.Millisecond)

	roundCancel := coordinator.ActiveRoundCancel()
	if roundCancel == nil {
		t.Fatal("expected active round cancel function")
	}

	scanCh <- struct{}{}
	time.Sleep(50 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		roundCancel()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("previous round cancel did not return after new scan")
	}

	cancel()
	wg.Wait()
}

func TestCoordinatorDropsOnFullTaskChannel(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		TaskBuffer:        0,
		SourceConcurrency: 1,
		ScanTimeout:       time.Second,
		HTTPTimeout:       time.Second,
	}

	coordinator := NewCoordinator(cfg, DefaultStubs())
	taskCh := make(chan pipeline.Task)
	state := &pipeline.RoundState{}

	err := coordinator.emit(context.Background(), "abc123", taskCh, state, model.RawItem{
		Source:  "stub:test",
		Raw:     "test",
		Contact: "x@y.com",
	})
	if err != nil {
		t.Fatalf("emit returned %v", err)
	}
	if state.Dropped.Load() != 1 {
		t.Fatalf("dropped=%d, want 1", state.Dropped.Load())
	}
}
