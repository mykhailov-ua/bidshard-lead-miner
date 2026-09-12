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

func TestHarvestEmployerReverse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	empPath := filepath.Join(dir, "employers.json")
	jobPath := filepath.Join(dir, "jobs.json")
	tgPath := filepath.Join(dir, "tg.json")
	tgQueryPath := filepath.Join(dir, "employer_tg_queries.json")
	if _, err := jobboard.UpsertEmployer(empPath, jobboard.EmployerEntry{Name: "TapOK", Slug: "tapok"}); err != nil {
		t.Fatal(err)
	}

	html := `<html><body>
<a class="result__a" href="https://t.me/unlim_arbitrage">Unlim Arbitrage</a>
<a class="result__snippet">TapOK media buying team keitaro</a>
</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	crawler := NewCrawler(config.Config{HTTPTimeout: 5 * time.Second}, nil)
	crawler.SetBaseURL(server.URL)
	if err := crawler.HarvestEmployerReverse(context.Background(), EmployerReverseConfig{
		EmployerRegistryPath:  empPath,
		JobboardRegistryPath:  jobPath,
		TGChannelsPath:        tgPath,
		EmployerTGQueriesPath: tgQueryPath,
		MaxPerRun:             5,
		RescanDays:            7,
	}); err != nil {
		t.Fatal(err)
	}
	channels := readTGChannelFile(tgPath)
	if len(channels.Channels) == 0 {
		t.Fatal("expected telegram channels from employer reverse")
	}
	found := false
	for _, ch := range channels.Channels {
		if ch.Username == "unlim_arbitrage" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unlim_arbitrage channel, got %v", channels.Channels)
	}
	queries, err := jobboard.LoadEmployerTGQueries(tgQueryPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries.Queries) != 1 || queries.Queries[0].Name != "TapOK" {
		t.Fatalf("tg queries=%v", queries.Queries)
	}
}
