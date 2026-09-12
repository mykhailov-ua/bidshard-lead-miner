package filter

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/validate"
)

func TestSerpPlaceholderContact(t *testing.T) {
	t.Parallel()
	if !SerpPlaceholderContact("serp:digiexe.com") {
		t.Fatal("expected serp placeholder")
	}
	if SerpPlaceholderContact("telegram:@buyer") {
		t.Fatal("expected telegram contact to pass")
	}
}

func TestSerpHasReachableContact(t *testing.T) {
	t.Parallel()
	if SerpHasReachableContact("serp:example.com", nil) {
		t.Fatal("expected placeholder without contacts to fail")
	}
	contacts := []extract.Contact{{Type: "telegram", Value: "@buyer_mx"}}
	if !SerpHasReachableContact("telegram:@buyer_mx", contacts) {
		t.Fatal("expected telegram handle to pass")
	}
}

func TestSerpBlacklistedSource(t *testing.T) {
	t.Parallel()
	if err := validate.LoadBlacklistDomains("../../data/blacklist_domains.txt"); err != nil {
		t.Fatalf("load blacklist: %v", err)
	}
	if !SerpBlacklistedSource("serp:cloakingtool.com") {
		t.Fatal("expected competitor serp domain blocked")
	}
}

func TestRejectAffiliateNetworkSupply(t *testing.T) {
	t.Parallel()
	drop, _ := RejectAffiliateNetworkSupply(
		"Affiliate network manager. In-house media buying team. Contact manager @lanaaff",
		"",
	)
	if !drop {
		t.Fatal("expected affiliate network supply to drop")
	}
}
