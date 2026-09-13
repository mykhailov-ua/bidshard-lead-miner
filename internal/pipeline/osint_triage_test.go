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

func TestProcessorForumTeamHiringAccept(t *testing.T) {
	reg := loadTestRegistry(t)
	p := &Processor{
		Registry:               reg,
		Seen:                   dedup.NewSeenCache(1000, 0),
		Store:                  sink.NewStubStore(),
		MX:                     validate.StubMX{OK: true},
		TelegramAcceptMinScore: 0,
	}
	out := p.Process(context.Background(), Task{
		RoundID: "r1",
		Item: model.RawItem{
			Source:   "forum:affiliatefix.com/hiring-thread",
			Raw:      "Hiring media buyer for paid social. DM ops@acme-media.com",
			Contact:  "forum:user/team_recruiter",
			Username: "team_recruiter",
			Title:    "Team expansion",
		},
	})
	if !out.Accepted {
		t.Fatalf("expected forum team hiring accept, reason=%s", out.RejectReason)
	}
	found := false
	for _, tag := range out.Lead.Tags {
		if tag == scoring.TagForumTeamHiring {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("tags=%v", out.Lead.Tags)
	}
}

func TestProcessorAdsTxtContactOutreachTag(t *testing.T) {
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
			Source:  "ads_txt:pub-example.com",
			Raw:     "CONTACT=ops@pub-example.com\nvoluum postback failing",
			Contact: "ops@pub-example.com",
		},
	})
	if !out.Accepted {
		t.Fatalf("expected accept, reason=%s", out.RejectReason)
	}
	hasContactTag := false
	for _, tag := range out.Lead.Tags {
		if tag == scoring.TagPublisherAdsTxtContact {
			hasContactTag = true
		}
	}
	if !hasContactTag {
		t.Fatalf("tags=%v", out.Lead.Tags)
	}
}
