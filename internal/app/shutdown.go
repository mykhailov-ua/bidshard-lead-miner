package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/pipeline"
)

func Drain(
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	taskCh chan pipeline.Task,
	timeout time.Duration,
) {
	cancel()

	close(taskCh)

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
	case <-time.After(timeout):
		slog.Warn("shutdown drain timeout", "timeout", timeout)
	}
}
