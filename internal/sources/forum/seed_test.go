package forum

import (
	"testing"
)

func TestLoadThreadURLsSkipsComments(t *testing.T) {
	t.Parallel()

	urls, err := LoadThreadURLs("../../../data/seeds/forum_threads.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) < 50 {
		t.Fatalf("urls=%d want >=50", len(urls))
	}
}
