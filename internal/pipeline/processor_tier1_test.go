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

func loadTestRegistry(t *testing.T) *scoring.Registry {
	t.Helper()
	reg := scoring.NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	return reg
}

func TestProcessorRejectsLanderIntelOnly(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	p := &Processor{
		Registry:              reg,
		Seen:                  dedup.NewSeenCache(1000, 0),
		Store:                 sink.NewStubStore(),
		MX:                    validate.StubMX{OK: true},
		LanderOutreachEnabled: false,
	}

	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "lander:affiliate-network.example.com",
			Raw:     "We are looking for a voluum alternative because postback is failing",
			Contact: "email:partners@affiliate-network.example.com",
		},
	})
	if out.Accepted {
		t.Fatal("expected lander intel-only reject")
	}
}

func TestProcessorRejectsLanderMarketingCopy(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	text := "voluum media buyer igaming affiliate s2s postback cost sync pricing adjust"
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "lander:affiliate-network.example.com",
			Raw:     text,
			Contact: "email:partners@affiliate-network.example.com",
		},
	})
	if out.Accepted {
		t.Fatal("expected lander marketing copy without buyer signal to reject")
	}
}

func TestProcessorRejectsLanderBoilerplate(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	text := "viewport width=device-width meta charset=UTF-8 x-ua-compatible theme-color initial-scale=1"
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "lander:affiliate-network.example.com",
			Raw:     text,
			Contact: "email:partners@affiliate-network.example.com",
		},
	})
	if out.Accepted {
		t.Fatal("expected lander html boilerplate reject")
	}
}

func TestProcessorRejectsLanderCompetitorDomain(t *testing.T) {
	t.Parallel()

	if err := validate.LoadBlacklistDomains("../../data/blacklist_domains.txt"); err != nil {
		t.Fatalf("load blacklist: %v", err)
	}

	reg := loadTestRegistry(t)
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "lander:voluum.com",
			Raw:     "voluum media buyer s2s postback cost sync",
			Contact: "email:support@voluum.com",
		},
	})
	if out.Accepted {
		t.Fatal("expected blacklisted competitor lander reject")
	}
}

func TestProcessorRejectsGitHubSSOTax(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "github:stopthessotax/sso-wall-of-shame",
			Raw:     "Companion resource: a self-hosted SSO tax tracker pricing parity",
			Contact: "github:iblessi",
		},
	})
	if out.Accepted {
		t.Fatal("expected github SSO infra reject")
	}
}

func TestProcessorRejectsLanderCSSContacts(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	store := sink.NewStubStore()
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	text := "voluum media buyer igaming affiliate voluumtrk charset=UTF-8 viewport width=device-width"
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "lander:voluum.com",
			Raw:     text,
			Contact: "telegram:@media",
		},
	})
	if out.Accepted {
		t.Fatal("expected lander CSS contact reject")
	}
}

func TestProcessorRejectsGitHubWithoutPain(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	store := sink.NewStubStore()
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "github:keitaroinc/docker-ckan",
			Raw:     "How is the envvars mechanism supposed to work in ckan docker?",
			Contact: "github:keitaroinc",
		},
	})
	if out.Accepted {
		t.Fatal("expected github without pain to reject")
	}
}

func TestProcessorRejectsGitHubVendorOrgEvenWithPain(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	store := sink.NewStubStore()
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "github:keitaroinc/docker-ckan",
			Raw:     "voluum postback failing s2s migration from keitaro docker-ckan envvars",
			Contact: "github:keitaroinc",
		},
	})
	if out.Accepted {
		t.Fatal("expected github vendor org to reject even with tracker pain words")
	}
}

func TestProcessorAcceptsRedditBuyerIntent(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	store := sink.NewStubStore()
	p := &Processor{
		Registry:          reg,
		Seen:              dedup.NewSeenCache(1000, 0),
		Store:             store,
		MX:                validate.StubMX{OK: true},
		LeadStatusEnabled: true,
		GeminiDefer:       true,
		PilotTagEnabled:   true,
	}

	text := "Starting to consider Voluum and BeMob alternatives for Awin postback tracking"
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "reddit:r/affiliatemarketing",
			Raw:     text,
			Contact: "reddit:buyer1",
		},
	})
	if !out.Accepted {
		t.Fatal("expected reddit buyer intent accept")
	}
	if out.Lead.PilotQualified {
		t.Fatal("defer mode must not set pilot_qualified on hot path")
	}
	for _, tag := range out.Lead.Tags {
		if tag == "pilot-qualified" {
			t.Fatal("defer mode must not tag pilot-qualified on hot path")
		}
	}
}

func TestProcessorRejectsTelegramChannelSelfBroadcast(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    sink.NewStubStore(),
		MX:       validate.StubMX{OK: true},
	}

	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "telegram:@igaming_news",
			Raw:     "Daily igaming news digest voluum keitaro market update for affiliates",
			Contact: "@igaming_news",
		},
	})
	if out.Accepted {
		t.Fatal("expected channel self broadcast reject")
	}
}

func TestProcessorRejectsJunkTelegramContactsOnly(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	store := sink.NewStubStore()
	p := &Processor{
		Registry: reg,
		Seen:     dedup.NewSeenCache(1000, 0),
		Store:    store,
		MX:       validate.StubMX{OK: true},
	}

	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:  "stub:test",
			Raw:     "voluum alternative postback failing @media @keyframes",
			Contact: "telegram:@media",
		},
	})
	if out.Accepted {
		t.Fatal("expected junk telegram contact only to reject")
	}
}

func TestProcessorKeitaroIncLowScore(t *testing.T) {
	t.Parallel()

	reg := loadTestRegistry(t)
	body := "keitaroinc docker-ckan helm chart failing"
	result := scoring.AnalyzeWithRegistry(reg, body)
	_, _, _, _, mediumMin := reg.Snapshot()
	score := scoring.ApplySpendGate(result.Score, body, mediumMin)
	if score >= mediumMin {
		t.Fatalf("score=%d want below mediumMin=%d without keitaro keyword hit", score, mediumMin)
	}
}
