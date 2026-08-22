package entity

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

func TestResolveKeysForumUser(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		Source:    "forum:affiliatefix.com/voluum-thread",
		ForumUser: "BuyerJohn",
		ForumUID:  "12345",
		Contacts: []extract.Contact{
			{Type: "email", Value: "buyer@affnet.com"},
		},
	})
	found := map[string]bool{}
	for _, k := range keys {
		found[k.Canonical()] = true
	}
	for _, want := range []string{
		"domain:affnet.com",
		"forum_uid:affiliatefix.com:12345",
		"forum_user:affiliatefix.com:buyerjohn",
	} {
		if !found[want] {
			t.Fatalf("missing key %q in %v", want, keys)
		}
	}
}

func TestResolveKeysForumUserFromDisplayName(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		Source:      "warrior:binom-tracker-thread",
		DisplayName: "MediaBuyer42",
	})
	found := map[string]bool{}
	for _, k := range keys {
		found[k.Canonical()] = true
	}
	if !found["forum_user:warriorforum:mediabuyer42"] {
		t.Fatalf("missing warrior forum user key in %v", keys)
	}
}

func TestResolveKeysSkipsForumUserOnNonForumSource(t *testing.T) {
	keys := ResolveKeys(ResolveInput{
		Source:    "reddit:igaming",
		ForumUser: "BuyerJohn",
	})
	for _, k := range keys {
		if k.Kind == KindForumUser || k.Kind == KindForumUID {
			t.Fatalf("unexpected forum key on reddit: %+v", k)
		}
	}
}

func TestNormalizeForumUser(t *testing.T) {
	cases := []struct {
		host, user, want string
	}{
		{"affiliatefix.com", "BuyerJohn", "affiliatefix.com:buyerjohn"},
		{"affiliatefix.com", "@BuyerJohn", "affiliatefix.com:buyerjohn"},
		{"affiliatefix.com", "anonymous", ""},
		{"affiliatefix.com", "", ""},
	}
	for _, tc := range cases {
		got := NormalizeForumUser(tc.host, tc.user)
		if got != tc.want {
			t.Fatalf("NormalizeForumUser(%q, %q)=%q want %q", tc.host, tc.user, got, tc.want)
		}
	}
}

func TestMemoryStoreForumUserDifferentHostsStaySeparate(t *testing.T) {
	store := NewMemoryStore()

	first, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			Source:    "forum:affiliatefix.com/thread-a",
			ForumUser: "BuyerJohn",
		},
		HashID:  "hash-af",
		Matched: []string{"voluum"},
		Text:    "voluum alternative",
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			Source:    "forum:blackhatworld.com/thread-b",
			ForumUser: "BuyerJohn",
		},
		HashID:  "hash-bhw",
		Matched: []string{"postback"},
		Text:    "postback failing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.EntityID == first.EntityID {
		t.Fatalf("same username on different hosts must not merge: %q", second.EntityID)
	}
}

func TestMemoryStoreForumAndRedditMergeOnEmail(t *testing.T) {
	store := NewMemoryStore()

	first, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			Source:    "forum:affiliatefix.com/thread-a",
			ForumUser: "BuyerJohn",
			Contacts: []extract.Contact{
				{Type: "email", Value: "buyer@affnet.com"},
			},
		},
		HashID:  "hash-af",
		Matched: []string{"voluum"},
		Text:    "voluum alternative",
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := store.RecordSighting(t.Context(), SightingInput{
		ResolveInput: ResolveInput{
			Source: "reddit:igaming",
			Contacts: []extract.Contact{
				{Type: "email", Value: "buyer@affnet.com"},
			},
		},
		HashID:  "hash-reddit",
		Matched: []string{"postback"},
		Text:    "postback failing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.EntityID != first.EntityID {
		t.Fatalf("expected merge via shared email domain, got %q vs %q", second.EntityID, first.EntityID)
	}
	if second.SourceCount < 2 {
		t.Fatalf("source_count=%d want >= 2", second.SourceCount)
	}
}

func TestForumHostFromSource(t *testing.T) {
	cases := map[string]string{
		"forum:affiliatefix.com/slug": "affiliatefix.com",
		"forum:www.blackhatworld.com": "blackhatworld.com",
		"warrior:binom-thread":        "warriorforum",
		"reddit:igaming":              "",
	}
	for in, want := range cases {
		if got := ForumHostFromSource(in); got != want {
			t.Fatalf("ForumHostFromSource(%q)=%q want %q", in, got, want)
		}
	}
}

func TestEnrichForumIdentity(t *testing.T) {
	in := EnrichForumIdentity(ResolveInput{}, "ForumAuthor", "Thread title must not become user", "999")
	if in.ForumUser != "ForumAuthor" {
		t.Fatalf("ForumUser=%q", in.ForumUser)
	}
	if in.ForumUID != "999" {
		t.Fatalf("ForumUID=%q", in.ForumUID)
	}
}

func TestEnrichForumIdentityIgnoresTitleWithoutUsername(t *testing.T) {
	in := EnrichForumIdentity(ResolveInput{Source: "forum:affiliatefix.com/thread-a"}, "", "Voluum billing help", "")
	if in.ForumUser != "" {
		t.Fatalf("ForumUser=%q want empty when username missing", in.ForumUser)
	}
	keys := ResolveKeys(in)
	for _, k := range keys {
		if k.Kind == KindForumUser {
			t.Fatalf("unexpected forum_user key from title: %+v", k)
		}
	}
}
