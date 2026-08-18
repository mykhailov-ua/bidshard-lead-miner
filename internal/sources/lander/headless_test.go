package lander

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
)

func TestPlaywrightPoolMockFetch(t *testing.T) {
	pool := NewPlaywrightPoolFetcher(2, 5*time.Second)
	pool.SetMockRunner(func(ctx context.Context, url string) (string, error) {
		return "<html><body><script id=\"__NEXT_DATA__\">{}</script><p>Rendered text with contact telegram:@lander_test</p></body></html>", nil
	})

	html, err := pool.Fetch(context.Background(), "https://example.com/lander")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if !strings.Contains(html, "telegram:@lander_test") {
		t.Errorf("expected rendered contact in returned HTML")
	}
}

func TestCrawlerHeadlessFallback(t *testing.T) {
	cfg := config.Config{
		LanderHeadless: true,
	}

	httpFetcher := NewHTTPFetcher(100*time.Millisecond, "")
	pool := NewPlaywrightPoolFetcher(1, 1*time.Second)
	pool.SetMockRunner(func(ctx context.Context, url string) (string, error) {
		return "<html><body><p>Headless contact telegram:@fallback_test</p></body></html>", nil
	})

	crawler := NewCrawler(cfg, httpFetcher, pool)
	if !crawler.headlessEnabled {
		t.Fatalf("expected headlessEnabled: true")
	}
}
