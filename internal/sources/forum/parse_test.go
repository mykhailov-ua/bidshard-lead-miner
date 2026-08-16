package forum

import (
	"os"
	"testing"
)

func TestParsePostsFromHTML(t *testing.T) {
	t.Parallel()

	html, err := os.ReadFile("../../../testdata/forum/stm_thread.html")
	if err != nil {
		t.Fatal(err)
	}

	posts := ParsePostsFromHTML(string(html))
	if len(posts) < 2 {
		t.Fatalf("posts=%d want >=2", len(posts))
	}

	foundPain := false
	foundContact := false
	for _, post := range posts {
		if HasPainSignal(post.Body) {
			foundPain = true
		}
		if contains(post.Body, "ops@igaming-team.com") || contains(post.Body, "@buyer_mx") {
			foundContact = true
		}
	}
	if !foundPain {
		t.Fatal("expected pain signal in posts")
	}
	if !foundContact {
		t.Fatal("expected contact hints in posts")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || stringIndex(s, sub))
}

func stringIndex(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
