package salesexport

import (
	"testing"

	"github.com/bidshard/parser/internal/sink"
)

func TestLeadCardFromDocRussianLabels(t *testing.T) {
	t.Parallel()
	card := LeadCardFromDoc(sink.LeadDoc{
		HashID:          "abc",
		Priority:        "high",
		Score:           80,
		HeatTier:        "hot",
		Source:          "forum:affiliatefix.com",
		OutreachChannel: "telegram",
		Outcome:         "pilot_started",
		PilotQualified:  true,
		PilotWhy:        "migration intent",
		Contacts:        []sink.StoredContact{{Type: "telegram", Value: "@buyer"}},
	})
	if card.Priority != "высокий" || card.HeatTier != "тёплый" {
		t.Fatalf("card=%+v", card)
	}
	if card.Outcome != "пилот запущен" {
		t.Fatalf("outcome=%q", card.Outcome)
	}
	if len(card.Contacts) != 1 || card.Contacts[0] != "telegram:@buyer" {
		t.Fatalf("contacts=%v", card.Contacts)
	}
}
