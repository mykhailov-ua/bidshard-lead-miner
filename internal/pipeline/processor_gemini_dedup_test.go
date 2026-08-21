package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

type countingGeo struct {
	calls atomic.Int32
}

func (c *countingGeo) ClassifyGeo(_ context.Context, _ string, _ []string, _ []string) (gemini.GeoResult, error) {
	c.calls.Add(1)
	return gemini.GeoResult{PersonCountry: "US"}, nil
}

func TestProcessorSeenReserveSkipsDuplicateGemini(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	geo := &countingGeo{}
	proc := &Processor{
		Registry:   reg,
		Seen:       dedup.NewSeenCache(1000, 0),
		Store:      sink.NewStubStore(),
		MX:         validate.StubMX{OK: true},
		Geo:        geo,
		GeoEnabled: true,
	}

	item := model.RawItem{
		Source:  "stub:test",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@igaming-team.com",
	}
	task := Task{RoundID: "r1", Item: item}

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			proc.Process(context.Background(), task)
		}()
	}
	wg.Wait()

	if got := geo.calls.Load(); got != 1 {
		t.Fatalf("geo classify calls=%d want 1", got)
	}
}
