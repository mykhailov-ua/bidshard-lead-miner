package serp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/parser/internal/config"
)

func TestSearchDorkPOST(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) == "" {
			http.Error(w, "empty body", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSERPHTML))
	}))
	defer ts.Close()

	cfg := config.Config{}
	crawler := NewCrawler(cfg, ts.Client())
	crawler.SetBaseURL(ts.URL)

	results, err := crawler.searchDork(context.Background(), "voluum alternative")
	if err != nil {
		t.Fatalf("searchDork: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected SERP results")
	}
}

func TestSearchDorkRetriesOn202(t *testing.T) {
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSERPHTML))
	}))
	defer ts.Close()

	cfg := config.Config{}
	crawler := NewCrawler(cfg, ts.Client())
	crawler.SetBaseURL(ts.URL)

	results, err := crawler.searchDork(context.Background(), "tracker")
	if err != nil {
		t.Fatalf("searchDork: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results after retry")
	}
	if calls < 3 {
		t.Fatalf("calls=%d want >=3 retries", calls)
	}
}
