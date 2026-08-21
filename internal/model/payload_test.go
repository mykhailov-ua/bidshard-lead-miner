package model

import "testing"

func TestLimitCrawlHTML(t *testing.T) {
	t.Parallel()

	short := "voluum alternative"
	if got := LimitCrawlHTML(short); got != short {
		t.Fatalf("got %q", got)
	}

	long := make([]byte, MaxCrawlHTMLBytes+100)
	for i := range long {
		long[i] = 'a'
	}
	got := LimitCrawlHTML(string(long))
	if len(got) != MaxCrawlHTMLBytes {
		t.Fatalf("len=%d want %d", len(got), MaxCrawlHTMLBytes)
	}
}
