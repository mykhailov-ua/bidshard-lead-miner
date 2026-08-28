package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestConfigCheckRejectsInvalidProxyList(t *testing.T) {
	chdirRepoRoot(t)
	t.Setenv("PARSER_SOURCE", "forum")
	t.Setenv("PARSER_PROXY_LIST", "http://:0")
	t.Setenv("PARSER_BG_TELEGRAM", "false")
	t.Setenv("TELEGRAM_API_ID", "")
	t.Setenv("TELEGRAM_API_HASH", "")
	t.Setenv("MONGO_URI", "")

	var out bytes.Buffer
	err := runConfigCheck(t.Context(), &out)
	if err == nil {
		t.Fatal("expected config check error for invalid PARSER_PROXY_LIST")
	}
	if !strings.Contains(err.Error(), "PARSER_PROXY_LIST") {
		t.Fatalf("err=%v", err)
	}
}

func TestConfigCheckProdRejectsFixtureForumSeed(t *testing.T) {
	chdirRepoRoot(t)
	t.Setenv("PARSER_SEED_PROFILE", "prod")
	t.Setenv("PARSER_SOURCE", "forum")
	t.Setenv("FORUM_SEED_PATH", "data/seeds/forum_threads.csv")
	t.Setenv("PARSER_BG_TELEGRAM", "false")
	t.Setenv("TELEGRAM_API_ID", "")
	t.Setenv("TELEGRAM_API_HASH", "")
	t.Setenv("MONGO_URI", "")

	var out bytes.Buffer
	err := runConfigCheck(t.Context(), &out)
	if err == nil {
		t.Fatal("expected config check error for fixture forum seed in prod profile")
	}
	if !strings.Contains(out.String(), "fixture marker") {
		t.Fatalf("output=%q", out.String())
	}
}

func TestConfigCheckProdAcceptsLiveForumSeed(t *testing.T) {
	chdirRepoRoot(t)
	t.Setenv("PARSER_SEED_PROFILE", "prod")
	t.Setenv("PARSER_SOURCE", "forum")
	t.Setenv("FORUM_SEED_PATH", "data/seeds/forum_threads.live.csv")
	t.Setenv("PARSER_ICP_CLASSIFY_TGWEB", "false")
	t.Setenv("PARSER_BG_TELEGRAM", "false")
	t.Setenv("TELEGRAM_API_ID", "")
	t.Setenv("TELEGRAM_API_HASH", "")
	t.Setenv("MONGO_URI", "")

	var out bytes.Buffer
	if err := runConfigCheck(t.Context(), &out); err != nil {
		t.Fatalf("config check: %v\n%s", err, out.String())
	}
}

func TestConfigCheckErrorsOnGeminiFlagWithoutKey(t *testing.T) {
	chdirRepoRoot(t)
	t.Setenv("PARSER_SOURCE", "tgweb")
	t.Setenv("PARSER_ICP_CLASSIFY_TGWEB", "true")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("PARSER_BG_TELEGRAM", "false")
	t.Setenv("TELEGRAM_API_ID", "")
	t.Setenv("TELEGRAM_API_HASH", "")
	t.Setenv("MONGO_URI", "")

	var out bytes.Buffer
	err := runConfigCheck(t.Context(), &out)
	if err == nil {
		t.Fatal("expected error when tgweb ICP enabled without GEMINI_API_KEY")
	}
	if !strings.Contains(out.String(), "GEMINI_API_KEY required") {
		t.Fatalf("output=%q", out.String())
	}
}

func TestConfigCheckProdAcceptsAutoDiscoverForumRegistry(t *testing.T) {
	chdirRepoRoot(t)
	dir := t.TempDir()
	registryPath := dir + "/discovered_forum_threads.json"
	if err := os.WriteFile(registryPath, []byte(`{
  "threads": [
    {
      "url": "https://affiliatefix.com/threads/voluum-postback.1001",
      "source": "serp",
      "at": "2026-08-22T12:00:00Z",
      "title": "Voluum postback failing",
      "snippet": "S2S tracker pain"
    }
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	csvPath := dir + "/forum_threads.csv"
	if err := os.WriteFile(csvPath, []byte("url,notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PARSER_SEED_PROFILE", "prod")
	t.Setenv("PARSER_SOURCE", "forum")
	t.Setenv("PARSER_AUTO_DISCOVER", "true")
	t.Setenv("FORUM_SEED_PATH", csvPath)
	t.Setenv("FORUM_REGISTRY_PATH", registryPath)
	t.Setenv("PARSER_ICP_CLASSIFY_TGWEB", "false")
	t.Setenv("PARSER_BG_TELEGRAM", "false")
	t.Setenv("TELEGRAM_API_ID", "")
	t.Setenv("TELEGRAM_API_HASH", "")
	t.Setenv("MONGO_URI", "")

	var out bytes.Buffer
	if err := runConfigCheck(t.Context(), &out); err != nil {
		t.Fatalf("config check: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "forum auto-discover registry") {
		t.Fatalf("output=%q", out.String())
	}
}
