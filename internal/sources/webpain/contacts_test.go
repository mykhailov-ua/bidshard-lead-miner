package webpain

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

func TestPickPageLPROnDomainEmail(t *testing.T) {
	t.Parallel()
	page := extract.Result{Contacts: []extract.Contact{
		{Type: "email", Value: "partners@affiliate.example.com"},
		{Type: "telegram", Value: "@media"},
	}}
	primary, ok := PickPageLPR(page, "affiliate.example.com")
	if !ok || primary.Value != "partners@affiliate.example.com" {
		t.Fatalf("primary=%+v ok=%v", primary, ok)
	}
}

func TestPickPageLPRRejectsOffDomainEmail(t *testing.T) {
	t.Parallel()
	page := extract.Result{Contacts: []extract.Contact{
		{Type: "email", Value: "ops@other.example.com"},
		{Type: "telegram", Value: "@buyer_mx"},
	}}
	if _, ok := PickPageLPR(page, "affiliate.example.com"); ok {
		t.Fatal("expected off-domain email to fail")
	}
}

func TestPickPageLPRRejectsTelegramOnly(t *testing.T) {
	t.Parallel()
	page := extract.Result{Contacts: []extract.Contact{{Type: "telegram", Value: "@buyer_mx"}}}
	if _, ok := PickPageLPR(page, "affiliate.example.com"); ok {
		t.Fatal("expected telegram-only to fail")
	}
}

func TestFilterPipelineContactsDropsCSS(t *testing.T) {
	t.Parallel()
	out := FilterPipelineContacts("webpain:affiliate.example.com", "", []extract.Contact{
		{Type: "telegram", Value: "@media"},
		{Type: "email", Value: "partners@affiliate.example.com"},
	})
	if len(out) != 1 || out[0].Type != "email" {
		t.Fatalf("out=%v", out)
	}
}
