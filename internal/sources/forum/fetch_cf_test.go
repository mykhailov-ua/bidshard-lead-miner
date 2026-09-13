package forum

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/parser/internal/limit"
)

func TestForumFetcherCFBlockEnqueuesHeadless(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("CF-Ray", "forum-ray")
		http.Error(w, "blocked", http.StatusForbidden)
	}))
	defer server.Close()

	var enqueuedURL string
	f := &Fetcher{
		client:        server.Client(),
		limiters:      limit.NewHostLimiters(100, 8),
		headlessDefer: true,
	}
	f.SetCFHeadlessEnqueue(func(fetchURL string, proxyIndex int) error {
		enqueuedURL = fetchURL
		return nil
	})

	_, err := f.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if enqueuedURL != server.URL {
		t.Fatalf("enqueued=%q want %s", enqueuedURL, server.URL)
	}
}
