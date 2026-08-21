package lander

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPFetcherGetRSCHeaders(t *testing.T) {
	t.Parallel()

	var gotRSC, gotNextURL, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRSC = r.Header.Get("RSC")
		gotNextURL = r.Header.Get("Next-Url")
		gotAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`a:["$","p",null,{"children":"voluum alternative rsc"}]`))
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher(time.Second, "")
	body, err := fetcher.GetRSC(context.Background(), server.URL+"/offer/preview?id=1")
	if err != nil {
		t.Fatal(err)
	}
	if gotRSC != "1" {
		t.Fatalf("RSC header=%q want 1", gotRSC)
	}
	if gotNextURL != "/offer/preview?id=1" {
		t.Fatalf("Next-Url=%q", gotNextURL)
	}
	if gotAccept != "text/x-component" {
		t.Fatalf("Accept=%q", gotAccept)
	}
	if !strings.Contains(body, "voluum alternative") {
		t.Fatalf("body=%q", body)
	}
}

func TestExtractFlightWireText(t *testing.T) {
	t.Parallel()

	payload := `a:["$","div",null,{"children":"voluum alternative wire"}]
b:["$","p",null,{"children":"postback failing rsc"}]`

	text := ExtractFlightWireText(payload)
	if !strings.Contains(strings.ToLower(text), "voluum alternative") {
		t.Fatalf("missing keyword in %q", text)
	}
	if !strings.Contains(text, "postback failing") {
		t.Fatalf("missing pain text in %q", text)
	}
}

func TestShouldFetchRSC(t *testing.T) {
	t.Parallel()

	html := `<html><body><div id="__next"></div><script src="/_next/static/chunks/app/page.js"></script></body></html>`
	if !ShouldFetchRSC(html) {
		t.Fatal("expected RSC fetch for app shell")
	}

	pagesRouter := `<script id="__NEXT_DATA__" type="application/json">{"props":{}}</script>`
	if ShouldFetchRSC(pagesRouter) {
		t.Fatal("pages router should not trigger RSC fetch")
	}
}
