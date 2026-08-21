package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCrawlClientFallsBackToShared(t *testing.T) {
	httpclientReset(t)

	client := CrawlClient(5*time.Second, []string{"://bad"})
	shared := Shared(5 * time.Second)
	if client != shared {
		t.Fatal("expected shared fallback on invalid proxy list")
	}
}

func TestDoBytesReturnsStatusAndCloses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, "tea")
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, status, err := DoBytes(http.DefaultClient, req, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusTeapot {
		t.Fatalf("status=%d want 418", status)
	}
	if string(body) != "tea" {
		t.Fatalf("body=%q", body)
	}
}
