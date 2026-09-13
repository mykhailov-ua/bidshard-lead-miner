package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/validate"
)

func TestProcessorRejectsTelegramCPASupplyIntelOnly(t *testing.T) {
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
			Source:   "telegram:@cpa_network",
			LeadType: "cpa_supply",
			Raw:      "Join our CPA network. High converting offers. Register as affiliate.",
			Username: "am_manager",
			Contact:  "telegram:@am_manager",
		},
	})
	if out.Accepted {
		t.Fatal("expected cpa_supply intel_only reject")
	}
	if out.RejectReason != "intel_only" {
		t.Fatalf("reject_reason=%q want intel_only", out.RejectReason)
	}
}

func TestProcessorColdTeamSkipsKeitaroPainScore(t *testing.T) {
	reg := loadTestRegistry(t)
	p := &Processor{
		Registry:              reg,
		Seen:                  dedup.NewSeenCache(1000, 0),
		Store:                 sink.NewStubStore(),
		MX:                    validate.StubMX{OK: true},
		TelegramAcceptMinScore: 0,
	}
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:      "telegram:@mediabuy_team",
			LeadType:    "cold_team",
			Raw:         "Hiring media buyer for FB. We use keitaro internally.",
			Username:    "recruiter",
			Contact:     "telegram:@recruiter",
			CompanyHint: "Acme Media",
		},
	})
	if !out.Accepted {
		t.Fatalf("expected cold_team accept, reason=%s", out.RejectReason)
	}
	for _, m := range out.Lead.Matched {
		if strings.EqualFold(strings.TrimSpace(m), "keitaro") {
			t.Fatalf("cold_team should not match keitaro keyword: %v", out.Lead.Matched)
		}
	}
	if out.Lead.Score <= 0 {
		t.Fatalf("score=%d", out.Lead.Score)
	}
}
