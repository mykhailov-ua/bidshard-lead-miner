package supply

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
)

func TestCrawlerEmitsSellerContact(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/ads.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("voluum.com, 123, DIRECT\n"))
	})
	mux.HandleFunc("/sellers.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"contact_email":"ops@igaming-team.com"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	seedFile := writeTempSeed(t, "crawler-test.example")
	cfg := config.Config{
		SupplySeedPath:   seedFile,
		SupplyMaxDomains: 1,
		SupplyHostRPS:    10,
		HTTPTimeout:      5 * time.Second,
		SupplyBaseURL:    server.URL,
	}

	fetcher := NewFetcher(cfg.HTTPTimeout, cfg.SupplyHostRPS, server.URL)
	crawler := NewCrawler(cfg, fetcher)

	var items []model.RawItem
	err := crawler.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].Contact != "ops@igaming-team.com" {
		t.Fatalf("contact=%q", items[0].Contact)
	}
	if items[0].Source != "ads_txt:crawler-test.example" {
		t.Fatalf("source=%q", items[0].Source)
	}
}

func writeTempSeed(t *testing.T, domain string) string {
	t.Helper()
	path := t.TempDir() + "/domains.csv"
	content := "domain,geo\n" + domain + ",global\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
