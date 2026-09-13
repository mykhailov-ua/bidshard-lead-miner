package lander

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestEnqueueHeadlessDedupes(t *testing.T) {
	ResetHeadlessPersonaBudgetForTest()
	t.Parallel()
	path := filepath.Join(t.TempDir(), "headless_queue.json")
	item := HeadlessQueueItem{
		URL:          "https://example.com/affiliate",
		SourceFamily: "tgweb",
		Reason:       "empty_extract",
	}
	if err := EnqueueHeadless(path, item); err != nil {
		t.Fatal(err)
	}
	if err := EnqueueHeadless(path, item); err != nil {
		t.Fatal(err)
	}
	pending, err := LoadPendingHeadless(path, 10, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending=%d want 1", len(pending))
	}
}

func TestPageFetcherDeferEnqueuesInsteadOfHeadless(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "headless_queue.json")
	fetcher := newHTTPFetcher(srv.Client(), "")
	pool := NewPlaywrightPoolFetcher(1, time.Second)
	pool.SetMockRunner(func(ctx context.Context, url string, params HeadlessFetchParams) (string, error) {
		t.Fatal("headless should not run in defer mode")
		return "", nil
	})
	pf := NewPageFetcher(fetcher, pool, PageFetchOptions{
		HeadlessDefer: true,
		QueuePath:     path,
		SourceFamily:  "lander",
	})
	_, meta, err := pf.FetchHTML(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected http error")
	}
	if meta.Stage != "headless_fallback_queued" {
		t.Fatalf("stage=%q want headless_fallback_queued", meta.Stage)
	}
	pending, err := LoadPendingHeadless(path, 5, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].URL != srv.URL {
		t.Fatalf("pending=%v", pending)
	}
	if pending[0].ProxyIndex != -1 {
		t.Fatalf("proxy_index=%d want -1 for direct httptest client", pending[0].ProxyIndex)
	}
}

func TestRawSourceLabel(t *testing.T) {
	t.Parallel()
	got := RawSourceLabel(HeadlessQueueItem{
		URL:          "https://affnet.com/join",
		SourceFamily: "tgweb",
	})
	if got != "tgweb:affnet.com" {
		t.Fatalf("got %q", got)
	}
}
