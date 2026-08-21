package tgweb

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

func TestPickSiteLPRRequiresEmailOrSkype(t *testing.T) {
	t.Parallel()

	page := extract.Result{
		Contacts: []extract.Contact{
			{Type: "telegram", Value: "@wooden_blog"},
		},
	}
	if _, ok := pickSiteLPR(page, "wooden_blog", "bojoko.com"); ok {
		t.Fatal("expected channel telegram only to be rejected")
	}
}

func TestPickSiteLPRAcceptsSiteEmail(t *testing.T) {
	t.Parallel()

	page := extract.Result{
		Contacts: []extract.Contact{
			{Type: "telegram", Value: "@wooden_blog"},
			{Type: "email", Value: "partners@bojoko.com"},
		},
	}
	primary, ok := pickSiteLPR(page, "wooden_blog", "bojoko.com")
	if !ok {
		t.Fatal("expected site email")
	}
	if primary.Type != "email" || primary.Value != "partners@bojoko.com" {
		t.Fatalf("got %+v", primary)
	}
}

func TestPickSiteLPRAcceptsSiteEmailSubdomain(t *testing.T) {
	t.Parallel()

	page := extract.Result{
		Contacts: []extract.Contact{
			{Type: "email", Value: "affiliates@mail.bojoko.com"},
		},
	}
	primary, ok := pickSiteLPR(page, "", "bojoko.com")
	if !ok {
		t.Fatal("expected subdomain email")
	}
	if primary.Value != "affiliates@mail.bojoko.com" {
		t.Fatalf("got %+v", primary)
	}
}

func TestPickSiteLPRRejectsOffDomainEmail(t *testing.T) {
	t.Parallel()

	page := extract.Result{
		Contacts: []extract.Contact{
			{Type: "email", Value: "affiliates@other-network.com"},
		},
	}
	if _, ok := pickSiteLPR(page, "", "bojoko.com"); ok {
		t.Fatal("expected off-domain email to be rejected")
	}
}

func TestPickSiteLPRRejectsSentryIngest(t *testing.T) {
	t.Parallel()

	page := extract.Result{
		Contacts: []extract.Contact{
			{Type: "email", Value: "7@o451079.ingest.de.sentry.io"},
			{Type: "telegram", Value: "@wooden_blog"},
		},
	}
	if _, ok := pickSiteLPR(page, "wooden_blog", "bojoko.com"); ok {
		t.Fatal("expected sentry ingest without site email to be rejected")
	}
}

func TestEmailMatchesSite(t *testing.T) {
	t.Parallel()

	if !emailMatchesSite("a@example.com", "example.com") {
		t.Fatal("exact match")
	}
	if !emailMatchesSite("a@mail.example.com", "example.com") {
		t.Fatal("subdomain match")
	}
	if emailMatchesSite("a@notexample.com", "example.com") {
		t.Fatal("suffix false positive")
	}
}

func TestIsValidCrawlDomain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		domain string
		ok     bool
	}{
		{"bojoko.com", true},
		{"13-02-2023-1.jpg", false},
		{"claude-opus-4.7.txt", false},
		{"durov.gram", false},
		{"google.co.uk", false},
		{"site.com", false},
	}
	for _, tc := range cases {
		if got := IsValidCrawlDomain(tc.domain); got != tc.ok {
			t.Errorf("domain %s got %v want %v", tc.domain, got, tc.ok)
		}
	}
}

func TestFilterPipelineContactsDropsCSSHandles(t *testing.T) {
	t.Parallel()

	source := "tgweb:@aff_net:searchenginejournal.com"
	extracted := []extract.Contact{
		{Type: "telegram", Value: "@media"},
		{Type: "telegram", Value: "@keyframes"},
		{Type: "email", Value: "partners@searchenginejournal.com"},
	}
	out := FilterPipelineContacts(source, "partners@searchenginejournal.com", extracted)
	if len(out) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(out))
	}
	if out[0].Type != "email" || out[0].Value != "partners@searchenginejournal.com" {
		t.Fatalf("got %+v", out[0])
	}
}

func TestFilterPipelineContactsSkypeFromHint(t *testing.T) {
	t.Parallel()

	source := "tgweb:@buylinkpro:buylink.pro"
	out := FilterPipelineContacts(source, "skype:aff.manager", nil)
	if len(out) != 1 || out[0].Type != "skype" {
		t.Fatalf("got %+v", out)
	}
}

func TestFilterPipelineContactsPrefersPartnersEmail(t *testing.T) {
	t.Parallel()

	source := "tgweb:@aff:example.com"
	extracted := []extract.Contact{
		{Type: "email", Value: "info@example.com"},
		{Type: "email", Value: "partners@example.com"},
	}
	out := FilterPipelineContacts(source, "", extracted)
	if len(out) != 1 || out[0].Value != "partners@example.com" {
		t.Fatalf("got %+v", out)
	}
}

func TestFilterPipelineContactsRejectsRoleOnly(t *testing.T) {
	t.Parallel()

	source := "tgweb:@aff:example.com"
	extracted := []extract.Contact{
		{Type: "email", Value: "info@example.com"},
		{Type: "email", Value: "support@example.com"},
	}
	out := FilterPipelineContacts(source, "", extracted)
	if len(out) != 0 {
		t.Fatalf("expected role-only reject, got %+v", out)
	}
}

func TestPickSiteLPRRejectsJunkSkype(t *testing.T) {
	t.Parallel()

	page := extract.Result{
		Contacts: []extract.Contact{
			{Type: "skype", Value: "skype"},
			{Type: "skype", Value: "live"},
		},
	}
	if _, ok := pickSiteLPR(page, "", "example.com"); ok {
		t.Fatal("expected junk skype to be rejected")
	}
}
