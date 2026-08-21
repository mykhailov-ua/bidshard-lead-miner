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

type stubICPNone struct{}

func (stubICPNone) ClassifyICP(_ context.Context, _ string) (gemini.ICPResult, error) {
	return gemini.ICPResult{ICP: "none", Why: "not a media buyer"}, nil
}

type stubEngage struct {
	result gemini.EngagementResult
	err    error
}

func (s stubEngage) ClassifyEngagement(_ context.Context, _ gemini.EngagementInput) (gemini.EngagementResult, error) {
	return s.result, s.err
}

func TestProcessorSkipsFixtureSources(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	store := sink.NewStubStore()
	proc := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(10, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	item := model.RawItem{
		Source:  "fixture:telegram:@affiliate_latam_en",
		Raw:     "voluum alternative with postback failing",
		Contact: "telegram:@media_buyer_0",
	}
	out := proc.Process(context.Background(), Task{RoundID: "r1", Item: item})
	if out.Accepted {
		t.Fatal("expected fixture source to be skipped")
	}
	if store.ExistsCalls != 0 {
		t.Fatalf("store exists calls=%d want 0", store.ExistsCalls)
	}
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

func TestProcessorTgWebPassesAffiliatePrescan(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	proc := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(10, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "tgweb:@aff_net:buylink.pro",
			Title:   "site buylink.pro via telegram @aff_net",
			Raw:     "Voluum alternative for igaming affiliate program and media buying. Partner program with postback integration. Contact partnerships@buylink.pro",
			Contact: "partnerships@buylink.pro",
		},
	})
	if !out.Accepted {
		t.Fatal("expected tgweb affiliate context to pass prescan and accept")
	}
}

func TestProcessorTgWebAggressiveThinPageSkype(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	proc := &Processor{
		Registry:         reg,
		Seen:             dedup.NewSeenCache(10, 0),
		Store:            sink.NewStubStore(),
		MX:               validate.StubMX{OK: true},
		TgWebPrescanMode: scoring.TgWebPrescanAggressive,
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "tgweb:@buylinkpro:buylink.pro",
			Title:   "site buylink.pro via telegram @buylinkpro",
			Raw:     "__next static chunk noise without affiliate keywords",
			Contact: "skype:aff.manager",
		},
	})
	if out.Accepted {
		t.Fatal("expected aggressive tgweb skype lpr without tracker pain to reject")
	}
}

func TestProcessorTgWebHardRejectBypassForSiteLPR(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	proc := &Processor{
		Registry:         reg,
		Seen:             dedup.NewSeenCache(10, 0),
		Store:            sink.NewStubStore(),
		MX:               validate.StubMX{OK: true},
		TgWebPrescanMode: scoring.TgWebPrescanAggressive,
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "tgweb:@blask:blask.com",
			Title:   "site blask.com via telegram @blask",
			Raw:     "Blask analytics for beginners in igaming affiliate marketing. Partner program with postback.",
			Contact: "partnerships@blask.com",
		},
	})
	if out.HardRejected {
		t.Fatal("expected hard reject bypass for tgweb site email lpr")
	}
	if !out.Accepted {
		t.Fatal("expected tgweb marketing page with site email to accept")
	}
}

func TestProcessorTgWebStrictRejectsThinPage(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	proc := &Processor{
		Registry:         reg,
		Seen:             dedup.NewSeenCache(10, 0),
		Store:            sink.NewStubStore(),
		MX:               validate.StubMX{OK: true},
		TgWebPrescanMode: scoring.TgWebPrescanStrict,
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "tgweb:@buylinkpro:buylink.pro",
			Title:   "site buylink.pro via telegram @buylinkpro",
			Raw:     "__next static chunk noise without affiliate keywords",
			Contact: "skype:aff.manager",
		},
	})
	if out.Accepted {
		t.Fatal("expected strict mode to reject thin page without affiliate context")
	}
}

func TestProcessorTgWebRejectsICPNone(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	proc := &Processor{
		Registry:        reg,
		Seen:            dedup.NewSeenCache(10, 0),
		Store:           sink.NewStubStore(),
		MX:              validate.StubMX{OK: true},
		ICP:             stubICPNone{},
		ICPTgWebEnabled: true,
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "tgweb:@aff_net:buylink.pro",
			Title:   "site buylink.pro via telegram @aff_net",
			Raw:     "Voluum alternative for igaming affiliate program and media buying. Partner program with postback integration. Contact partnerships@buylink.pro",
			Contact: "partnerships@buylink.pro",
		},
	})
	if out.Accepted {
		t.Fatal("expected tgweb lead with icp=none to be rejected")
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

func TestProcessorStoresEngagementOnAccept(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	store := sink.NewStubStore()
	proc := &Processor{
		Registry:        reg,
		Store:           store,
		MX:              validate.StubMX{OK: true},
		PilotTagEnabled: true,
		EngageEnabled:   true,
		Engage: stubEngage{result: gemini.EngagementResult{
			PilotSignals:    []string{"migration_intent", "competitor_stack", "tracker_pain"},
			PilotQualified:  true,
			PilotWhy:        "voluum migration pain",
			OutreachChannel: "email",
			OutreachAngle:   "Postback reliability",
			OutreachDraft:   "Hi - noticed postback issues while switching trackers.",
		}},
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
	if !out.Lead.PilotQualified {
		t.Fatal("expected pilot qualified")
	}
	if out.Lead.OutreachChannel != "email" {
		t.Fatalf("outreach_channel=%q", out.Lead.OutreachChannel)
	}
	if out.Lead.OutreachDraft == "" {
		t.Fatal("expected outreach draft")
	}
}

func TestProcessorEmbedPrescanPass(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	prescan := &recordingPrescan{painMatch: true, painScore: 0.95}
	proc := &Processor{
		Registry:       reg,
		Store:          sink.NewStubStore(),
		MX:             validate.StubMX{OK: true},
		PrescanEnabled: true,
		Prescan:        prescan,
	}

	_ = proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "stub:test",
			Raw:     "plain unrelated text without registry keywords",
			Contact: "telegram:@buyer_mx",
		},
	})
	if !prescan.painCalled {
		t.Fatal("expected embed prescan pain evaluation")
	}
}

type recordingPrescan struct {
	painCalled bool
	spamCalled bool
	painMatch  bool
	painScore  float64
}

func (r *recordingPrescan) EvaluatePain(_ context.Context, _ string) (gemini.PrescanVerdict, error) {
	r.painCalled = true
	return gemini.PrescanVerdict{PainMatch: r.painMatch, PainScore: r.painScore}, nil
}

func (r *recordingPrescan) EvaluateSpam(_ context.Context, _ string) (gemini.PrescanVerdict, error) {
	r.spamCalled = true
	return gemini.PrescanVerdict{}, nil
}

type stubLeadCluster struct {
	dup       bool
	clusterOf string
}

func (s stubLeadCluster) CheckDuplicate(_ context.Context, _, _ string) (bool, string, error) {
	return s.dup, s.clusterOf, nil
}

func (s stubLeadCluster) Record(_ context.Context, _, _ string) error {
	return nil
}

func TestProcessorSemanticLeadDedup(t *testing.T) {
	t.Parallel()

	reg := scoring.NewRegistry("../../testdata/keywords.json")
	_ = reg.Load(context.Background())

	proc := &Processor{
		Registry:           reg,
		Store:              sink.NewStubStore(),
		MX:                 validate.StubMX{OK: true},
		LeadClusterEnabled: true,
		LeadCluster: stubLeadCluster{
			dup:       true,
			clusterOf: "existing-hash",
		},
	}

	out := proc.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "stub:test",
			Raw:     "voluum alternative with postback failing",
			Contact: "ops@igaming-team.com",
		},
	})
	if !out.Dedup {
		t.Fatal("expected semantic dedup")
	}
	if out.Accepted {
		t.Fatal("expected not accepted")
	}
}
