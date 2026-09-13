package lander

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestPageFetcherCFBlockDefersToQueue(t *testing.T) {
	t.Parallel()
	InitHeadlessPersonaBudget(0, "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("CF-Ray", "test-ray")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("just a moment"))
	}))
	defer server.Close()

	queuePath := filepath.Join(t.TempDir(), "headless_queue.json")
	fetcher := newHTTPFetcher(server.Client(), "")
	pool := NewPlaywrightPoolFetcher(1, time.Second)
	pool.SetMockRunner(func(ctx context.Context, url string, params HeadlessFetchParams) (string, error) {
		t.Fatal("headless should not run in defer mode")
		return "", nil
	})

	pf := NewPageFetcher(fetcher, pool, PageFetchOptions{
		HeadlessDefer: true,
		QueuePath:     queuePath,
		SourceFamily:  "tgweb",
	})
	_, meta, status, err := pf.FetchForCrawl(context.Background(), server.URL, true)
	if err == nil {
		t.Fatal("expected http error")
	}
	if status != http.StatusForbidden {
		t.Fatalf("status=%d", status)
	}
	if meta.Stage != "cf_http_block_queued" {
		t.Fatalf("stage=%q", meta.Stage)
	}
	pending, err := LoadPendingHeadless(queuePath, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending=%d", len(pending))
	}
}
