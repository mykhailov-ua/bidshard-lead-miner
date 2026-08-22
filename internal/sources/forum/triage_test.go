package forum

import "testing"

func TestShouldCrawlSeedSkipsReview(t *testing.T) {
	if ShouldCrawlSeed("https://forum.example/threads/voluum-review-2026", "") {
		t.Fatal("expected review thread skipped")
	}
}

func TestShouldCrawlSeedAllowsBuyerPain(t *testing.T) {
	if !ShouldCrawlSeed("https://forum.example/threads/postback-failing-voluum", "buyer switching tracker") {
		t.Fatal("expected buyer pain thread crawled")
	}
	if !ShouldCrawlSeed("https://forum.example/threads/voluum-alternative.123/", "") {
		t.Fatal("expected non-review slug crawled")
	}
}
