package pipeline

import (
	"context"
	"testing"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

func TestProcessorSeenCacheSkipsMongoExists(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	store := sink.NewStubStore()
	seen := dedup.NewSeenCache(1000, 0)
	proc := &Processor{
		Registry: reg,
		Seen:     seen,
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	item := model.RawItem{
		Source:  "stub:test",
		Raw:     "voluum alternative with postback failing",
		Contact: "ops@igaming-team.com",
	}
	task := Task{RoundID: "r1", Item: item}

	out1 := proc.Process(context.Background(), task)
	if !out1.Accepted {
		t.Fatalf("expected accepted first pass, outcome=%+v", out1)
	}
	if store.ExistsCalls != 1 {
		t.Fatalf("exists calls=%d want 1", store.ExistsCalls)
	}

	out2 := proc.Process(context.Background(), task)
	if !out2.Dedup {
		t.Fatalf("expected dedup on second pass")
	}
	if store.ExistsCalls != 1 {
		t.Fatalf("exists calls=%d want 1 (seen cache)", store.ExistsCalls)
	}
}

func TestProcessorRejectsGeoBeforeScoring(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	proc := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(10, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "stub:ru",
			Raw:     "voluum alternative postback failing",
			Contact: "buyer@team.ru",
		},
	})
	if !out.RejectedGeo {
		t.Fatal("expected geo reject")
	}
}

func TestProcessorRejectsLinkedIn(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	proc := &Processor{
		Registry: reg,
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source: "stub:li",
			Raw:    "voluum alternative https://linkedin.com/in/buyer postback failing",
		},
	})
	if out.Accepted {
		t.Fatal("expected linkedin reject")
	}
}
