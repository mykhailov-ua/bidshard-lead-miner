package sourceregistry

import (
	"path/filepath"
	"testing"
)

func TestUpsertCascadeFansOutTypes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "source_registry.json")

	ok, err := UpsertCascade(path, "buylink.pro", "ads_txt", "supply", "")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected new entry")
	}

	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Sources) != 1 {
		t.Fatalf("sources=%d", len(f.Sources))
	}
	for _, typ := range CascadeTypes {
		if !hasType(f.Sources[0].Types, typ) {
			t.Fatalf("missing type %s in %+v", typ, f.Sources[0].Types)
		}
	}
}

func TestAcceptCascadeDomainSkipsNoiseHost(t *testing.T) {
	t.Parallel()

	if AcceptCascadeDomain("google.com") {
		t.Fatal("expected google.com rejected")
	}
	if !AcceptCascadeDomain("voluum.com") {
		t.Fatal("expected voluum.com accepted")
	}
}

func TestAcceptCascadePartnerDomainSkipsSSP(t *testing.T) {
	t.Parallel()

	for _, domain := range []string{"rubiconproject.com", "pubmatic.com", "ads.pubmatic.com"} {
		if AcceptCascadePartnerDomain(domain) {
			t.Fatalf("expected %s rejected for partner cascade", domain)
		}
	}
	if !AcceptCascadePartnerDomain("voluum.com") {
		t.Fatal("expected voluum.com accepted for partner cascade")
	}
	if !AcceptCascadeDomain("pubmatic.com") {
		t.Fatal("direct pubmatic seed crawl should still pass AcceptCascadeDomain")
	}
}

func TestCascadeDomainsDedupes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "source_registry.json")

	added, err := CascadeDomains(path, []string{"acme.com", "acme.com"}, "serp", "webpain")
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}
}
