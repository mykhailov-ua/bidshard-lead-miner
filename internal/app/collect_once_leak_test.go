package app

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestCollectOnceShutdownNoLeak(t *testing.T) {
	defer goleak.VerifyNone(t)

	cfg := config.Config{
		WorkerCount:          2,
		TaskBuffer:           8,
		ScanTimeout:          5 * time.Second,
		ProcessorTaskTimeout: 2 * time.Second,
	}
	deps := &runtimeDeps{}

	err := runCollectOnce(context.Background(), cfg, deps, "stub", func(ctx context.Context, emit func(ctx context.Context, item model.RawItem) error) error {
		return emit(ctx, model.RawItem{
			Source:  "stub:test",
			Raw:     "voluum alternative postback failing",
			Contact: "x@example.com",
		})
	})
	if err != nil {
		t.Fatalf("runCollectOnce: %v", err)
	}
}
