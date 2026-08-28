package enrich

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/sources/forum"
)

func TestForumProfileURL(t *testing.T) {
	t.Parallel()

	got := ForumProfileURL("affiliatefix.com", "buyer_mx", "42")
	if got != "https://affiliatefix.com/members/buyer_mx.42/" {
		t.Fatalf("got %q", got)
	}
}

func TestProfileEnricherForumProfile(t *testing.T) {
	t.Parallel()

	html := `<html><body><p>voluum alternative pain. Email ops@igaming-team.com telegram @buyer_mx</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/members/media_buyer.99/" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	fetcher := forum.NewFetcher(5*time.Second, server.URL)
	pe := NewProfileEnricherForTest(fetcher, server.Client(), "")

	out := pe.MergeContacts(context.Background(), "forum:affiliatefix.com/thread-a", []extract.Contact{
		{Type: "forum_user", Value: "media_buyer"},
	}, "media_buyer", "99")

	if !extract.HasReachableContact(out) {
		t.Fatalf("expected reachable contact, got %+v", out)
	}
}

func TestProfileEnricherGitHubAPI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/buyer42" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"bio":"tracker pain voluum alternative","blog":"https://buyer.io","email":"ops@igaming-team.com"}`))
	}))
	defer server.Close()

	client := server.Client()
	pe := NewProfileEnricherForTest(nil, client, "")
	pe.SetTestAPIBases(server.URL, "")

	out := pe.MergeContacts(context.Background(), "github:repo/issue", []extract.Contact{
		{Type: "github", Value: "buyer42"},
	}, "", "")

	if !extract.HasReachableContact(out) {
		t.Fatalf("expected email from github profile, got %+v", out)
	}
}

func TestProfileEnricherRedditAbout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/media_buyer/about.json" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"subreddit":{"public_description":"postback failing. telegram @nutra_buyer"}}}`))
	}))
	defer server.Close()

	client := server.Client()
	pe := NewProfileEnricherForTest(nil, client, "")
	pe.SetTestAPIBases("", server.URL)

	out := pe.MergeContacts(context.Background(), "reddit:igaming", []extract.Contact{
		{Type: "reddit", Value: "u/media_buyer"},
	}, "", "")

	if !extract.HasReachableContact(out) {
		t.Fatalf("expected telegram from reddit profile, got %+v", out)
	}
}
