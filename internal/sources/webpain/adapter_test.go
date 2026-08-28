package webpain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sources/forum"
)

func TestAdapterEmitsFromRegistryPage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	regPath := filepath.Join(dir, "discovered_web_pain.json")
	_, err := AppendDiscoveries(regPath, "serp", `"redtrack alternative"`, []Discovery{
		{
			URL:     "https://blog-affiliate.example/tracker-migration",
			Title:   "Migration pain",
			Snippet: "Need redtrack alternative before renewal",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	html := `<html><body>
<p>We need redtrack alternative with flat pricing. Contact partners@blog-affiliate.example</p>
</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	cfg := config.Config{
		WebPainRegistryPath: regPath,
		HTTPTimeout:         5 * time.Second,
	}
	adapter := NewAdapter(cfg, forum.NewFetcher(cfg.HTTPTimeout, server.URL))

	var items []model.RawItem
	if err := adapter.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].Contact != "partners@blog-affiliate.example" {
		t.Fatalf("contact=%q", items[0].Contact)
	}
	if items[0].CrawlHTML == "" {
		t.Fatal("expected crawl html preserved")
	}
	if !strings.Contains(items[0].Raw, "redtrack alternative") {
		t.Fatalf("raw=%q", items[0].Raw)
	}
}

func TestAdapterRejectsOffDomainEmail(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	regPath := filepath.Join(dir, "discovered_web_pain.json")
	_, err := AppendDiscoveries(regPath, "serp", `"voluum alternative"`, []Discovery{
		{URL: "https://blog-affiliate.example/pain", Snippet: "Need voluum alternative"},
	})
	if err != nil {
		t.Fatal(err)
	}

	html := `<html><body><p>Contact ops@other-network.com @media screen</p></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	adapter := NewAdapter(config.Config{WebPainRegistryPath: regPath, HTTPTimeout: 5 * time.Second}, forum.NewFetcher(5*time.Second, server.URL))
	var items []model.RawItem
	if err := adapter.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items=%d want 0", len(items))
	}
}

func TestAdapterRejectsBoilerplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	regPath := filepath.Join(dir, "discovered_web_pain.json")
	_, err := AppendDiscoveries(regPath, "serp", "q", []Discovery{
		{URL: "https://blog-affiliate.example/pain", Snippet: "tracker pain"},
	})
	if err != nil {
		t.Fatal(err)
	}

	html := `<html><head><meta charset=UTF-8><meta name=viewport content="width=device-width, initial-scale=1"></head>
<body>theme-color initial-scale=1 x-ua-compatible</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	adapter := NewAdapter(config.Config{WebPainRegistryPath: regPath, HTTPTimeout: 5 * time.Second}, forum.NewFetcher(5*time.Second, server.URL))
	var items []model.RawItem
	_ = adapter.Collect(context.Background(), func(ctx context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	})
	if len(items) != 0 {
		t.Fatalf("items=%d want 0", len(items))
	}
}

func TestAppendDiscoveriesDedupes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "web_pain.json")
	added, err := AppendDiscoveries(path, "serp", "q1", []Discovery{
		{URL: "https://blog.example/pain", Snippet: "a"},
		{URL: "https://blog.example/pain/", Snippet: "dup"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if added != 1 {
		t.Fatalf("added=%d want 1", added)
	}
	reg, err := LoadRegistry(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.URLs) != 1 {
		t.Fatalf("urls=%d", len(reg.URLs))
	}
}
