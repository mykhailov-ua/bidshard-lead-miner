package entity

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
)

func TestResolveKeysCompanyAndDomain(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		CompanyName: "AffNet Media LLC",
		Source:      "telegram:@affnet_official",
		Contacts: []extract.Contact{
			{Type: "email", Value: "ops@affnet.com"},
			{Type: "telegram", Value: "@buyer_mx"},
		},
	})

	if len(keys) < 3 {
		t.Fatalf("expected at least 3 keys, got %v", keys)
	}
	if keys[0].Kind != KindCompany || keys[0].Value != "affnet media" {
		t.Fatalf("primary key: got %+v", keys[0])
	}

	found := map[string]bool{}
	for _, k := range keys {
		found[k.Canonical()] = true
	}
	for _, want := range []string{
		"company:affnet media",
		"domain:affnet.com",
		"telegram:@buyer_mx",
		"channel:@affnet_official",
	} {
		if !found[want] {
			t.Fatalf("missing key %q in %v", want, keys)
		}
	}
}

func TestResolveKeysSkipsFreeMailDomain(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		Contacts: []extract.Contact{
			{Type: "email", Value: "buyer@gmail.com"},
			{Type: "telegram", Value: "@buyer_mx"},
		},
	})
	for _, k := range keys {
		if k.Kind == KindDomain {
			t.Fatalf("unexpected domain key from gmail: %+v", k)
		}
	}
	if len(keys) != 1 || keys[0].Kind != KindTelegram {
		t.Fatalf("got %v", keys)
	}
}

func TestResolveKeysTgWebDomain(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		Source: "tgweb:@wooden_blog:bojoko.com",
		Contacts: []extract.Contact{
			{Type: "email", Value: "partners@bojoko.com"},
		},
	})
	if len(keys) != 1 || keys[0].Value != "bojoko.com" {
		t.Fatalf("got %v", keys)
	}
}

func TestEntityIDStableFromPrimary(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		CompanyName: "Voluum Refugees Network",
		Contacts: []extract.Contact{
			{Type: "email", Value: "ceo@voluum-refugees.io"},
		},
	})
	id1 := EntityID(keys)
	id2 := EntityID(keys)
	if id1 == "" || id1 != id2 {
		t.Fatalf("entity id not stable: %q %q", id1, id2)
	}

	keysOnlyDomain := ResolveKeys(ResolveInput{
		Contacts: []extract.Contact{
			{Type: "email", Value: "ceo@voluum-refugees.io"},
		},
	})
	if EntityID(keysOnlyDomain) == id1 {
		t.Fatal("expected different entity id when primary key kind changes")
	}
}

func TestResolveInputFromLead(t *testing.T) {
	lead := model.Lead{
		CompanyName: "BidShard Partners",
		Source:      "reddit:igaming",
		DisplayName: "Ops Team",
	}
	keys := ResolveKeys(ResolveInputFromLead(lead, []extract.Contact{
		{Type: "email", Value: "ops@bidshard.io"},
	}))
	pk, ok := PrimaryKey(keys)
	if !ok || pk.Kind != KindCompany || pk.Value != "bidshard partners" {
		t.Fatalf("primary=%+v ok=%v keys=%v", pk, ok, keys)
	}
}

func TestNormalizeCompany(t *testing.T) {
	cases := map[string]string{
		"AffNet Media LLC":     "affnet media",
		"CPA Kings | iGaming":  "cpa kings",
		"  Voluum   Network  ": "voluum network",
	}
	for in, want := range cases {
		if got := NormalizeCompany(in); got != want {
			t.Fatalf("NormalizeCompany(%q)=%q want %q", in, got, want)
		}
	}
}

func TestResolveKeysAdsTxtDomain(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		Source: "ads_txt:example-pub.com",
		Contacts: []extract.Contact{
			{Type: "email", Value: "ops@example-pub.com"},
		},
	})
	found := map[string]bool{}
	for _, k := range keys {
		found[k.Canonical()] = true
	}
	if !found["domain:example-pub.com"] {
		t.Fatalf("missing ads_txt domain key in %v", keys)
	}
}

func TestSupplyDomainFromSource(t *testing.T) {
	if got := SupplyDomainFromSource("ads_txt:Example-Pub.COM"); got != "example-pub.com" {
		t.Fatalf("got %q", got)
	}
	if got := SupplyDomainFromSource("telegram:@foo"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestSourceFamily(t *testing.T) {
	cases := map[string]string{
		"telegram:@affnet":            "telegram",
		"tgweb:site.com":              "tgweb",
		"reddit:igaming":              "reddit",
		"forum:affiliatefix.com/slug": "forum",
		"warrior:binom-thread":        "warrior",
		"fixture:telegram:@foo":       "fixture",
		"ads_txt:example.com":         "supply",
	}
	for in, want := range cases {
		if got := SourceFamily(in); got != want {
			t.Fatalf("SourceFamily(%q)=%q want %q", in, got, want)
		}
	}
}

func TestAliasTokens(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		CompanyName: "Net Media",
		Contacts:    []extract.Contact{{Type: "telegram", Value: "@net_buyer"}},
	})
	aliases := AliasTokens(keys)
	if len(aliases) != 2 {
		t.Fatalf("aliases=%v", aliases)
	}
	if aliases[0] != "company:net media" {
		t.Fatalf("aliases=%v", aliases)
	}
}

func TestResolveKeysTelegramUserID(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		Source: "telegram:@affnet",
		Contacts: []extract.Contact{
			{Type: "telegram_user_id", Value: "99887766"},
		},
	})
	if len(keys) != 2 {
		t.Fatalf("keys=%v want channel + user_id", keys)
	}
	found := false
	for _, k := range keys {
		if k.Kind == KindTelegramUserID && k.Value == "99887766" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing telegram_user_id key in %v", keys)
	}
}

func TestResolveKeysEmpty(t *testing.T) {
	if keys := ResolveKeys(ResolveInput{}); len(keys) != 0 {
		t.Fatalf("expected no keys, got %v", keys)
	}
	if id := EntityID(nil); id != "" {
		t.Fatalf("expected empty entity id, got %q", id)
	}
}
