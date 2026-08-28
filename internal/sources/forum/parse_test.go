package forum

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseXenForoHTML(t *testing.T) {
	t.Parallel()

	html, err := os.ReadFile("../../../testdata/forum/xenforo_thread.html")
	if err != nil {
		t.Fatal(err)
	}

	posts := ParsePostsFromHTML(string(html))
	if len(posts) < 2 {
		t.Fatalf("posts=%d want >=2", len(posts))
	}
	if posts[0].Author != "media_buyer_mx" {
		t.Fatalf("author=%q want media_buyer_mx", posts[0].Author)
	}
	if !strings.Contains(strings.ToLower(posts[0].Body), "voluum alternative") {
		t.Fatalf("expected pain phrase in fixture body: %q", posts[0].Body)
	}
}

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
		if strings.Contains(strings.ToLower(post.Body), "voluum alternative") {
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

func TestParseWarriorPostContentHTML(t *testing.T) {
	t.Parallel()

	html := `<html><body>
	<time datetime="2025-06-01T12:00:00Z"></time>
	<div class="post-container">
		<a class="username">BuyerJohn</a>
		<div class="post-content">Looking for voluum alternative, contact me telegram:@buyerjohn</div>
	</div>
	</body></html>`

	posts := ParsePostsFromHTML(html)
	if len(posts) != 1 {
		t.Fatalf("posts=%d want 1", len(posts))
	}
	if posts[0].Author != "BuyerJohn" {
		t.Fatalf("author=%q", posts[0].Author)
	}
	want := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	if !posts[0].PostedAt.Equal(want) {
		t.Fatalf("posted_at=%v want %v", posts[0].PostedAt, want)
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
