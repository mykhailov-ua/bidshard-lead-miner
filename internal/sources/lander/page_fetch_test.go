package lander

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
)

func TestPageFetcherRSCMerge(t *testing.T) {
	t.Parallel()

	var rscRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("RSC") == "1" {
			rscRequest = true
			_, _ = w.Write([]byte(`a:["$","p",null,{"children":"affiliate program contact affiliates@example.com"}]`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><div id="__next"></div></body></html>`))
	}))
	defer server.Close()

	fetcher := newHTTPFetcher(server.Client(), "")
	pf := NewPageFetcher(fetcher, DisabledHeadless{}, PageFetchOptions{})
	html, meta, err := pf.FetchHTML(context.Background(), server.URL+"/affiliate")
	if err != nil {
		t.Fatal(err)
	}
	if !meta.RSCFetched {
		t.Fatal("expected RSC fetch")
	}
	if meta.Stage != "http_get+rsc" {
		t.Fatalf("stage=%q want http_get+rsc", meta.Stage)
	}
	if !rscRequest {
		t.Fatal("expected RSC HTTP request")
	}
	text, method := TextForContactExtract(html)
	if !strings.Contains(text, "affiliates@example.com") {
		t.Fatalf("missing email in %q method=%q", text, method)
	}
}

func TestPageFetcherHeadlessOnHTTPFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	fetcher := newHTTPFetcher(server.Client(), "")
	pool := NewPlaywrightPoolFetcher(1, time.Second)
	pool.SetMockRunner(func(ctx context.Context, url string) (string, error) {
		return "<html><body><footer>partnerships@headless.example.com</footer></body></html>", nil
	})

	pf := NewPageFetcher(fetcher, pool, PageFetchOptions{HeadlessEnabled: true})
	html, meta, err := pf.FetchHTML(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Stage != "headless_fallback" {
		t.Fatalf("stage=%q want headless_fallback", meta.Stage)
	}
	text := ExtractStaticLandingText(html)
	if !strings.Contains(text, "partnerships@headless.example.com") {
		t.Fatalf("html=%q", text)
	}
}

func TestPageFetcherHeadlessOnEmptyAfterRSC(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("RSC") == "1" {
			_, _ = w.Write([]byte(""))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body><div id="__next"></div></body></html>`))
	}))
	defer server.Close()

	fetcher := newHTTPFetcher(server.Client(), "")
	pool := NewPlaywrightPoolFetcher(1, time.Second)
	pool.SetMockRunner(func(ctx context.Context, url string) (string, error) {
		return "<html><body><a href=\"mailto:affiliates@headless.example.com\">contact</a></body></html>", nil
	})

	pf := NewPageFetcher(fetcher, pool, PageFetchOptions{HeadlessEnabled: true})
	html, meta, err := pf.FetchHTML(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Stage != "headless_empty" {
		t.Fatalf("stage=%q want headless_empty", meta.Stage)
	}
	text, _ := TextForContactExtract(html)
	if !strings.Contains(text, "affiliates@headless.example.com") {
		t.Fatalf("text=%q", text)
	}
}

func TestNewHTTPFetcherFromConfigWithoutProxies(t *testing.T) {
	t.Parallel()

	fetcher, err := NewHTTPFetcherFromConfig(config.Config{
		HTTPTimeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if fetcher == nil {
		t.Fatal("expected fetcher")
	}
}
