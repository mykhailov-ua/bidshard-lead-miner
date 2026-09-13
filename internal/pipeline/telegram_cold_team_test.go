package pipeline

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
)

func TestTelegramColdTeamContacts_prefersPoster(t *testing.T) {
	item := model.RawItem{
		LeadType: "cold_team",
		Username: "hr_recruiter",
		Source:   "telegram:@jobchannel",
	}
	contacts := []extract.Contact{
		{Type: "telegram", Value: "telegram:@jobchannel"},
		{Type: "email", Value: "jobs@example.com"},
	}
	out := telegramColdTeamContacts(item, contacts)
	if len(out) < 2 {
		t.Fatalf("contacts=%v", out)
	}
	if out[0].Value != "telegram:@hr_recruiter" {
		t.Errorf("first contact=%q want poster", out[0].Value)
	}
	for _, c := range out {
		if c.Value == "telegram:@jobchannel" {
			t.Error("channel contact should be dropped when poster present")
		}
	}
}
