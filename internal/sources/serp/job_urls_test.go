package serp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/jobboard"
)

func TestExtractJobboardDiscoveries(t *testing.T) {
	t.Parallel()
	results := []SERPResult{
		{URL: "https://jobs.dou.ua/companies/tapok/vacancies/118950/", Title: "Media Buyer"},
		{URL: "https://djinni.co/jobs/796247-dsp-media-buying-team-lead/", Title: "Team Lead"},
		{URL: "https://jobs.dou.ua/vacancies/?category=Marketing", Title: "Listing"},
	}
	got := ExtractJobboardDiscoveries(results)
	if len(got) != 2 {
		t.Fatalf("discoveries=%d want 2", len(got))
	}
}

func TestHarvestJobboardURLs(t *testing.T) {
	t.Parallel()
	customHTML := `<html><body>
<a class="result__a" href="https://jobs.dou.ua/companies/tapok/vacancies/118950/">Media Buyer TapOK</a>
<a class="result__snippet">Keitaro gambling media buyer</a>
</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(customHTML))
	}))
	defer server.Close()

	regPath := filepath.Join(t.TempDir(), "jobs.json")
	crawler := NewCrawler(config.Config{HTTPTimeout: 5 * time.Second}, nil)
	crawler.SetBaseURL(server.URL)

	if err := crawler.HarvestJobboardURLs(context.Background(), regPath); err != nil {
		t.Fatal(err)
	}
	f, err := jobboard.LoadRegistry(regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.URLs) == 0 {
		t.Fatal("expected registry entries")
	}
}

func TestSerpHarvestJobboardDorks(t *testing.T) {
	t.Parallel()
	got := serpHarvestJobboardDorks([]string{
		`site:jobs.dou.ua media buyer`,
		`site:t.me media buying`,
		`site:djinni.co arbitrage`,
	})
	if len(got) != 2 {
		t.Fatalf("dorks=%v", got)
	}
}
