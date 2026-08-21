package sources

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
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

func TestCoordinatorCollectUsesScanTimeout(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		WorkerCount:       1,
		TaskBuffer:        8,
		SourceConcurrency: 1,
		ScanTimeout:       400 * time.Millisecond,
		HTTPTimeout:       50 * time.Millisecond,
	}

	slow := &StubSource{
		name:  "stub:slow",
		delay: 150 * time.Millisecond,
		items: []model.RawItem{{Raw: "voluum alternative", Contact: "x@y.com"}},
	}
	coordinator := NewCoordinator(cfg, []Source{slow})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	taskCh := make(chan pipeline.Task, 8)
	statsCh := make(chan pipeline.RoundStats, 1)

	var drainWG sync.WaitGroup
	drainWG.Add(1)
	go func() {
		defer drainWG.Done()
		for task := range taskCh {
			task.Stats.FinishTask()
		}
	}()

	coordinator.runRound(ctx, taskCh, statsCh)
	close(taskCh)
	drainWG.Wait()

	stats := <-statsCh
	if stats.SourcesFail != 0 {
		t.Fatalf("sources_fail=%d, want 0 (collect should outlive HTTPTimeout)", stats.SourcesFail)
	}
	if stats.SourcesOK != 1 {
		t.Fatalf("sources_ok=%d, want 1", stats.SourcesOK)
	}
}

func TestCoordinatorDropsOnFullTaskChannel(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		TaskBuffer:        0,
		SourceConcurrency: 1,
		ScanTimeout:       time.Second,
		HTTPTimeout:       time.Second,
	}

	coordinator := NewCoordinator(cfg, []Source{
		NewStubSource("stub:test", []model.RawItem{
			{Raw: "voluum alternative", Contact: "x@y.com"},
		}),
	})
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
