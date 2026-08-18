package ct

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestCTCrawlerJSON(t *testing.T) {
	jsonFixture := `[
		{"name_value": "track.example.com\nclick.example.com"},
		{"name_value": "bad.ru"},
		{"name_value": "go.tracker.io"}
	]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(jsonFixture))
	}))
	defer ts.Close()

	cfg := config.Config{
		CTQueries:    []string{"track"},
		CTMaxResults: 10,
	}

	crawler := NewCrawler(cfg, ts.Client())
	crawler.baseURL = ts.URL
	crawler.rateTicker = time.NewTicker(1 * time.Millisecond)

	var emitted []model.RawItem
	err := crawler.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		emitted = append(emitted, item)
		return nil
	})

	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if len(emitted) == 0 {
		t.Fatalf("expected emitted items, got 0")
	}

	foundBad := false
	foundTrack := false
	for _, item := range emitted {
		if item.Title == "bad.ru" {
			foundBad = true
		}
		if item.Title == "track.example.com" {
			foundTrack = true
		}
	}

	if foundBad {
		t.Errorf(".ru domain should be blocked by TLD filter")
	}
	if !foundTrack {
		t.Errorf("expected track.example.com in emitted items")
	}
}
