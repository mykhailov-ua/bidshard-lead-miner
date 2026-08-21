package pipeline

import (
	"context"
	"errors"

	"github.com/bidshard/parser/internal/metrics"
)

// ErrTaskChannelFull means taskCh is full and the item was not queued (non-blocking emit).
var ErrTaskChannelFull = errors.New("task channel full")

// TryEnqueue sends task without blocking. Respects ctx cancellation on emit path.
// On drop, increments task.Stats.Dropped and parser_tasks_dropped_total.
func TryEnqueue(ctx context.Context, taskCh chan<- Task, task Task) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case taskCh <- task:
		if task.Stats != nil {
			task.Stats.TrackTask()
		}
		return nil
	default:
		if task.Stats != nil {
			task.Stats.Dropped.Add(1)
		}
		source := task.Item.Source
		if source == "" {
			source = "unknown"
		}
		metrics.RecordTaskDropped(source)
		return ErrTaskChannelFull
	}
}
