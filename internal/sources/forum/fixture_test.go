package forum

import (
	"context"
	"strings"
	"testing"
)

func TestFetcherFixtureHost(t *testing.T) {
	t.Parallel()

	f := NewFetcher(0, "")
	html, err := f.Get(context.Background(), "https://forum-fixture.test/threads/voluum-alternative.1001/")
	if err != nil {
		t.Fatal(err)
	}
	if html == "" {
		t.Fatal("expected fixture html")
	}
	posts := ParsePostsFromHTML(html)
	if len(posts) == 0 {
		t.Fatal("expected posts from fixture")
	}
	if !strings.Contains(strings.ToLower(posts[0].Body), "voluum") {
		t.Fatalf("expected pain phrase in fixture body: %q", posts[0].Body)
	}
}
