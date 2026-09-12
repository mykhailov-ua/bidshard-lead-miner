package jobboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sources/forum"
)

func TestAdapterEmitsDouVacancy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(douVacancyFixture))
	}))
	defer server.Close()

	regPath := writeRegistry(t, "https://jobs.dou.ua/companies/tapok/vacancies/118950/", "Media Buyer TapOK", "keitaro gambling")
	cfg := config.Config{
		JobboardRegistryPath: regPath,
		HTTPTimeout:          5 * time.Second,
	}

	adapter := NewAdapter(cfg, forum.NewFetcher(5*time.Second, server.URL))
	var items []model.RawItem
	err := adapter.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].Contact != "jobboard:company/tapok" {
		t.Fatalf("contact=%q", items[0].Contact)
	}
	if items[0].Username != "TapOK" {
		t.Fatalf("company=%q", items[0].Username)
	}
}

func writeRegistry(t *testing.T, url, title, snippet string) string {
	t.Helper()
	path := t.TempDir() + "/jobs.json"
	raw := `{"urls":[{"url":"` + url + `","title":"` + title + `","snippet":"` + snippet + `","source":"test"}]}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
