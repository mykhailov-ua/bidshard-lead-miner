package filter

import (
	"testing"

	"github.com/bidshard/parser/internal/extract"
)

func TestForumHasForumUserContact(t *testing.T) {
	t.Parallel()
	if ForumHasForumUserContact(nil) {
		t.Fatal("expected nil contacts to fail")
	}
	if ForumHasForumUserContact([]extract.Contact{{Type: "email", Value: "ops@team.com"}}) {
		t.Fatal("expected email-only to fail")
	}
	if !ForumHasForumUserContact([]extract.Contact{{Type: "forum_user", Value: "media_buyer"}}) {
		t.Fatal("expected forum_user to pass")
	}
}

func TestIsForumSource(t *testing.T) {
	t.Parallel()
	if !IsForumSource("forum:affiliatefix.com/thread-a") {
		t.Fatal("expected forum source")
	}
	if IsForumSource("serp:affiliatefix.com") {
		t.Fatal("expected serp to fail")
	}
}
