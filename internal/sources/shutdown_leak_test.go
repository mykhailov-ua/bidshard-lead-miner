package sources

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
)

func TestCoordinatorShutdownNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t)

	cfg := config.Config{
		WorkerCount:       1,
		TaskBuffer:        8,
		SourceConcurrency: 1,
		ScanTimeout:       2 * time.Second,
		HTTPTimeout:       time.Second,
	}

	coordinator := NewCoordinator(cfg, []Source{
		NewStubSource("stub:test", []model.RawItem{
			{Raw: "voluum alternative", Contact: "x@y.com"},
		}),
	})

	ctx, cancel := context.WithCancel(context.Background())
	taskCh := make(chan pipeline.Task, 8)
	scanCh := make(chan struct{}, 1)
	statsCh := make(chan pipeline.RoundStats, 1)

	var wg sync.WaitGroup
	coordinator.Run(ctx, &wg, scanCh, taskCh, statsCh)

	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for {
			select {
			case task, ok := <-taskCh:
				if !ok {
					return
				}
				task.Stats.FinishTask()
			case <-ctx.Done():
				return
			}
		}
	}()

	scanCh <- struct{}{}
	select {
	case stats := <-statsCh:
		if stats.SourcesOK != 1 {
			t.Fatalf("sources_ok=%d want 1", stats.SourcesOK)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("round stats timeout")
	}

	cancel()
	wg.Wait()
	<-drainDone
}
