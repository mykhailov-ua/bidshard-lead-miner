package app

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sink"
)

func testDepsConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		KeywordsJSONPath: "../../data/keywords.json",
		KeywordsGrayPath: "../../data/keywords-gray.json",
		HTTPTimeout:      2 * time.Second,
		WriteSlots:       4,
	}
}

func TestBuildDepsHotPathCRMWebhook(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := testDepsConfig(t)
	cfg.ExportJSONPath = filepath.Join(t.TempDir(), "leads.jsonl")
	cfg.CRMWebhookEnabled = true
	cfg.CRMWebhookURL = srv.URL

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if deps.bulkStore == nil {
		t.Fatal("expected bulk store")
	}
	if !sink.StoreNotifiesCRM(deps.bulkStore) {
		t.Fatal("expected hot-path CRM webhook on store")
	}
	if deps.warmPath != nil {
		t.Fatal("expected no warm path without gemini defer")
	}
}

func TestBuildDepsDeferredCRMWebhook(t *testing.T) {
	if !mongoReachable() {
		t.Skip("mongo not reachable on 127.0.0.1:27017")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := testDepsConfig(t)
	cfg.MongoURI = "mongodb://127.0.0.1:27017"
	cfg.MongoDB = "parser_deps_webhook_test"
	cfg.MongoCollection = "leads"
	cfg.GeminiAPIKey = "deps-test-key"
	cfg.ParserGeminiDefer = true
	cfg.ParserGeoClassify = true
	cfg.CRMWebhookEnabled = true
	cfg.CRMWebhookURL = srv.URL
	cfg.CRMWebhookAfterAnalysis = true

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if deps.bulkStore == nil {
		t.Fatal("expected bulk store")
	}
	if sink.StoreNotifiesCRM(deps.bulkStore) {
		t.Fatal("expected hot-path CRM webhook disabled when after-analysis defer is on")
	}
	if deps.warmPath == nil {
		t.Fatal("expected warm path with gemini defer")
	}
	if !deps.warmPath.DeferredCRMWebhook() {
		t.Fatal("expected deferred CRM webhook on warm path")
	}
}

func mongoReachable() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:27017", 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
