package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/model"
)

func TestTryEnqueueDropsWhenFull(t *testing.T) {
	t.Parallel()

	taskCh := make(chan Task)
	state := &RoundState{}
	err := TryEnqueue(context.Background(), taskCh, Task{
		RoundID: "r1",
		Item:    model.RawItem{Contact: "x@y.com"},
		Stats:   state,
	})
	if !errors.Is(err, ErrTaskChannelFull) {
		t.Fatalf("err=%v want ErrTaskChannelFull", err)
	}
	if state.Dropped.Load() != 1 {
		t.Fatalf("dropped=%d want 1", state.Dropped.Load())
	}
}

func TestTryEnqueueRespectsContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	taskCh := make(chan Task, 1)
	err := TryEnqueue(ctx, taskCh, Task{RoundID: "r1"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
}

func TestTryEnqueueTracksTask(t *testing.T) {
	t.Parallel()

	taskCh := make(chan Task, 1)
	state := &RoundState{}
	if err := TryEnqueue(context.Background(), taskCh, Task{RoundID: "r1", Stats: state}); err != nil {
		t.Fatal(err)
	}
	state.FinishTask()
	done := make(chan struct{})
	go func() {
		state.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after FinishTask")
	}
}
