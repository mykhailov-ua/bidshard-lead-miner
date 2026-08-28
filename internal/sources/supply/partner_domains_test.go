package supply

import "testing"

func TestPartnerDomainsFromCrawlSkipsSSPPartners(t *testing.T) {
	t.Parallel()

	lines := []AdsTxtLine{
		{Domain: "voluum.com", PubID: "1", Relation: "DIRECT"},
		{Domain: "rubiconproject.com", PubID: "2", Relation: "RESELLER"},
	}
	got := PartnerDomainsFromCrawl("publisher.com", lines, nil)
	if len(got) != 1 || got[0] != "voluum.com" {
		t.Fatalf("got %v want [voluum.com]", got)
	}
}

func TestPartnerDomainsFromCrawlSkipsSelfAndNoise(t *testing.T) {
	t.Parallel()

	lines := []AdsTxtLine{
		{Domain: "voluum.com", PubID: "1", Relation: "DIRECT"},
		{Domain: "google.com", PubID: "2", Relation: "RESELLER"},
	}
	sellers := []SellerContact{
		{Domain: "acme.com", ContactEmail: "ads@acme.com"},
	}
	got := PartnerDomainsFromCrawl("publisher.com", lines, sellers)
	if len(got) != 2 {
		t.Fatalf("got %v want voluum.com acme.com", got)
	}
}

func TestPartnerDomainsFromCrawlSkipsCrawlDomain(t *testing.T) {
	t.Parallel()

	lines := []AdsTxtLine{{Domain: "publisher.com", PubID: "1", Relation: "DIRECT"}}
	got := PartnerDomainsFromCrawl("publisher.com", lines, nil)
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
