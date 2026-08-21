package main

import (
	"bytes"
	"strings"
	"testing"
)

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
