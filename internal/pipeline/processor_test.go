package pipeline

import (
	"context"
	"testing"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

type stubGeo struct {
	result gemini.GeoResult
	err    error
}

func (s stubGeo) ClassifyGeo(_ context.Context, _ string, _ []string, _ []string) (gemini.GeoResult, error) {
	return s.result, s.err
}

type stubICP struct{}

func (stubICP) ClassifyICP(_ context.Context, _ string) (gemini.ICPResult, error) {
	return gemini.ICPResult{ICP: "starter", SpendTier: "unknown"}, nil
}

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

func TestProcessorHardReject(t *testing.T) {
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
			Source:  "stub:test",
			Raw:     "cheapest tracker for beginner? voluum alternative",
			Contact: "buyer@igaming-team.com",
		},
	})
	if out.Accepted {
		t.Fatal("expected hard reject")
	}
	if !out.HardRejected {
		t.Fatal("expected HardRejected flag")
	}
}

func TestProcessorRejectsEmailWithoutPainContext(t *testing.T) {
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
			Source:  "stub:test",
			Raw:     "ops@igaming-team.com",
			Contact: "ops@igaming-team.com",
		},
	})
	if out.Accepted {
		t.Fatal("expected email-only reject")
	}
}

func TestProcessorRejectsGeminiGeo(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	proc := &Processor{
		Registry:          reg,
		Store:             sink.NewStubStore(),
		MX:                validate.StubMX{OK: true},
		GeoEnabled:        true,
		GeoBlockCountries: []string{"RU", "BY"},
		Geo: stubGeo{result: gemini.GeoResult{
			Blocked:        true,
			Confidence:     "high",
			CompanyCountry: "RU",
			CompanyName:    "ООО Медиа",
			Why:            "Moscow registration",
		}},
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "stub:test",
			Raw:     "voluum alternative postback failing self-hosted tracker",
			Contact: "buyer@igaming-team.com",
		},
	})
	if !out.RejectedGeo {
		t.Fatal("expected gemini geo reject")
	}
}

func TestProcessorStoresGeoOnAccept(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	store := sink.NewStubStore()
	proc := &Processor{
		Registry:          reg,
		Store:             store,
		MX:                validate.StubMX{OK: true},
		GeoEnabled:        true,
		GeoBlockCountries: []string{"RU", "BY"},
		Geo: stubGeo{result: gemini.GeoResult{
			Blocked:        false,
			Confidence:     "medium",
			PersonCountry:  "US",
			CompanyCountry: "CY",
			CompanyName:    "MediaBuy Ltd",
		}},
		ICPEnabled: true,
		ICP:        stubICP{},
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "stub:test",
			Raw:     "voluum alternative with postback failing",
			Contact: "ops@igaming-team.com",
		},
	})
	if !out.Accepted {
		t.Fatalf("expected accepted, outcome=%+v", out)
	}
	if out.Lead.CompanyCountry != "CY" || out.Lead.GeoCountry != "US" {
		t.Fatalf("geo fields not stored: %+v", out.Lead)
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

func TestProcessorAcceptsDomainContact(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	store := sink.NewStubStore()
	proc := &Processor{
		Registry: reg,
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "ct:example.com",
			Raw:     "voluum alternative postback failing self-hosted tracker",
			Contact: "domain:buyer-team.com",
		},
	})
	if !out.Accepted {
		t.Fatalf("expected domain contact accepted, outcome=%+v", out)
	}
	if out.Lead.HashID == "" {
		t.Fatal("expected hash_id")
	}
}
