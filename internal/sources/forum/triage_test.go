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

func TestTriageThreadTenURLsThreeRejectedWithoutHTTP(t *testing.T) {
	seeds := []ThreadSeed{
		{URL: "https://affiliatefix.com/threads/voluum-postback-failing.1", Title: "Postback failing on Voluum", Snippet: "S2S postback debug"},
		{URL: "https://stmforum.com/threads/keitaro-too-expensive.2", Title: "Keitaro too expensive", Snippet: "Self-hosted tracker migration"},
		{URL: "https://blackhatworld.com/seo/binom-self-hosted.3", Title: "Binom self-hosted setup", Snippet: "Media buy tracker pain"},
		{URL: "https://affiliatefix.com/threads/redtrack-billing.4", Title: "RedTrack billing issue", Snippet: "Igaming affiliate billing"},
		{URL: "https://stmforum.com/threads/voluum-alternative.5", Title: "Voluum alternative needed", Snippet: "Tracker migration thread"},
		{URL: "https://affiliatefix.com/threads/postback-s2s.6", Title: "S2S postback tracker", Snippet: "Failing postback on igaming campaign"},
		{URL: "https://blackhatworld.com/seo/media-buy-tracker.7", Title: "Media buy tracker stack", Snippet: "Igaming affiliate stack"},
		{URL: "https://affiliatefix.com/threads/best-voluum-alternatives-2026.8", Title: "Best Voluum alternatives 2026 review roundup", Snippet: "Top 10 trackers comparison"},
		{URL: "https://stmforum.com/threads/hiring-media-buyer.9", Title: "We are hiring media buyer", Snippet: "Job posting vacancy"},
		{URL: "https://affiliatefix.com/threads/free-spins-casino-bonus.10", Title: "Free spins casino bonus review", Snippet: "Affiliate program review promo"},
	}

	rejected := 0
	intent := 0
	for _, seed := range seeds {
		fetch, verdict := TriageThread(seed)
		if !fetch {
			rejected++
			if verdict != ThreadVerdictNoise {
				t.Fatalf("url=%s verdict=%q want noise", seed.URL, verdict)
			}
			continue
		}
		if verdict == ThreadVerdictBuyerIntent {
			intent++
		}
	}
	if rejected != 3 {
		t.Fatalf("rejected=%d want 3 without HTTP", rejected)
	}
	if intent < 7 {
		t.Fatalf("buyer_intent=%d want >=7 fetchable threads", intent)
	}
}
