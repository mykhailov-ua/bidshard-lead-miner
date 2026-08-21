package serp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

var mockSERPHTML = `
<div class="result">
	<a class="result__a" href="https://blackhatworld.com/seo/best-voluum-alternative.12345">Best Voluum Alternative for Affiliate Marketing</a>
	<a class="result__snippet">Looking for self-hosted tracker, Voluum is too expensive. Contact me on Telegram @aff_lead</a>
</div>
`

func TestSERPCrawler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockSERPHTML))
	}))
	defer ts.Close()

	cfg := config.Config{}
	crawler := NewCrawler(cfg, ts.Client())
	crawler.SetBaseURL(ts.URL)

	var emitted []model.RawItem
	emit := func(ctx context.Context, item model.RawItem) error {
		emitted = append(emitted, item)
		return nil
	}

	err := crawler.Collect(context.Background(), emit)
	if err != nil {
		t.Fatalf("unexpected error during collect: %v", err)
	}

	if len(emitted) == 0 {
		t.Fatalf("expected emitted items from SERP search")
	}
	if emitted[0].Contact != "telegram:@aff_lead" {
		t.Errorf("expected contact telegram:@aff_lead, got: %s", emitted[0].Contact)
	}
}
