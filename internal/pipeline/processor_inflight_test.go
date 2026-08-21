package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

type slowExistsStore struct {
	inner        *sink.StubStore
	existsBlocks chan struct{}
	existsCalls  atomic.Int32
}

func (s *slowExistsStore) Exists(ctx context.Context, hashID string) (bool, error) {
	s.existsCalls.Add(1)
	select {
	case <-s.existsBlocks:
	case <-ctx.Done():
		return false, ctx.Err()
	}
	return s.inner.Exists(ctx, hashID)
}

func (s *slowExistsStore) Upsert(ctx context.Context, lead model.Lead) error {
	return s.inner.Upsert(ctx, lead)
}

func (s *slowExistsStore) UpdateStatus(ctx context.Context, hashID, status string) error {
	return s.inner.UpdateStatus(ctx, hashID, status)
}

func TestProcessorHashInflightDedup(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	inner := sink.NewStubStore()
	store := &slowExistsStore{
		inner:        inner,
		existsBlocks: make(chan struct{}),
	}
	proc := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	item := model.RawItem{
		Source:  "stub:test",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@igaming-team.com",
	}
	task := Task{RoundID: "r1", Item: item}

	var wg sync.WaitGroup
	results := make([]ProcessOutcome, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = proc.Process(context.Background(), task)
		}(i)
	}

	time.Sleep(50 * time.Millisecond)
	close(store.existsBlocks)
	wg.Wait()

	accepted := 0
	dedup := 0
	for _, out := range results {
		if out.Accepted {
			accepted++
		}
		if out.Dedup {
			dedup++
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted=%d want 1, results=%+v", accepted, results)
	}
	if dedup != 1 {
		t.Fatalf("dedup=%d want 1, results=%+v", dedup, results)
	}
	if store.existsCalls.Load() != 1 {
		t.Fatalf("exists calls=%d want 1", store.existsCalls.Load())
	}
}
