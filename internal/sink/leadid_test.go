package sink

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
)

func TestLeadHashIDByContacts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		contacts []extract.Contact
		want     string
	}{
		{
			name: "email+telegram",
			contacts: []extract.Contact{
				{Type: "email", Value: "ops@igaming-team.com"},
				{Type: "telegram", Value: "@buyer_mx"},
			},
			want: "3f808a82172da0775afc4b32c77b4460",
		},
		{
			name: "email only",
			contacts: []extract.Contact{
				{Type: "email", Value: "ops@igaming-team.com"},
			},
			want: "9c89321514bc7047daba5997fe13742b",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := LeadHashIDFromExtract(tc.contacts)
			if got != tc.want {
				t.Fatalf("hash=%s want=%s", got, tc.want)
			}
		})
	}
}

func TestLeadHashIDIgnoresSourceAndTitle(t *testing.T) {
	t.Parallel()

	a := LeadHashIDFromExtract([]extract.Contact{{Type: "email", Value: "ops@igaming-team.com"}})
	b := LeadHashIDFromExtract([]extract.Contact{{Type: "email", Value: "OPS@igaming-team.com"}})
	if a != b {
		t.Fatalf("case-insensitive email hash mismatch: %s vs %s", a, b)
	}
}

func TestLeadHashIDOrderIndependent(t *testing.T) {
	t.Parallel()

	a := LeadHashIDFromExtract([]extract.Contact{
		{Type: "email", Value: "ops@igaming-team.com"},
		{Type: "telegram", Value: "@buyer_mx"},
	})
	b := LeadHashIDFromExtract([]extract.Contact{
		{Type: "telegram", Value: "@buyer_mx"},
		{Type: "email", Value: "ops@igaming-team.com"},
	})
	if a != b {
		t.Fatalf("hash order mismatch: %s vs %s", a, b)
	}
}

func TestToLeadDocStoresNormalizedContacts(t *testing.T) {
	t.Parallel()

	doc := ToLeadDoc(modelLead(
		[]string{"ops@igaming-team.com", "telegram:@buyer_mx"},
		"telegram:foo",
	))
	if len(doc.Contacts) != 2 {
		t.Fatalf("contacts=%d want 2", len(doc.Contacts))
	}
	if doc.Contacts[0].Value != "ops@igaming-team.com" || doc.Contacts[1].Value != "@buyer_mx" {
		t.Fatalf("contacts=%+v", doc.Contacts)
	}
	if doc.Source != "telegram:foo" {
		t.Fatalf("source=%q", doc.Source)
	}
}

func modelLead(contacts []string, source string) model.Lead {
	return model.Lead{
		HashID:   LeadHashIDFromExtract(ParseFormattedContacts(contacts)),
		Source:   source,
		Contacts: contacts,
		Priority: "High",
		Score:    10,
		Matched:  []string{"voluum"},
		Snippet:  "voluum alternative",
		Title:    "ignored for hash",
	}
}
