package forum

import (
	"context"
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
	if !HasPainSignal(posts[0].Body) {
		t.Fatalf("expected pain signal in fixture body: %q", posts[0].Body)
	}
}
