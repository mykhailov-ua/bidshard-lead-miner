package config

import (
	"strings"
	"testing"
)

func TestApplyAcceptQualityDefaultsCRMDefer(t *testing.T) {
	for _, key := range []string{
		"PARSER_ICP_CLASSIFY",
		"PARSER_EMBED_PRESCAN",
		"PARSER_SOURCE_DISABLE_GOVERNOR",
		"PARSER_CHANNEL_TRIAGE",
		"PARSER_SOURCE_PRIORITY",
		"PARSER_INTENT_CLASSIFY",
	} {
		t.Setenv(key, "")
	}

	cfg := Config{
		CRMWebhookEnabled: true,
		GeminiAPIKey:      "key",
		ParserGeminiDefer: true,
		BGTelegramEnabled: true,
	}
	applyAcceptQualityDefaults(&cfg)
	if !cfg.ParserICPClassify {
		t.Fatal("expected ICP classify default on")
	}
	if !cfg.ParserEmbedPrescan {
		t.Fatal("expected embed prescan default on")
	}
	if !cfg.ParserSourceDisableGovernor {
		t.Fatal("expected source disable governor default on")
	}
	if !cfg.ParserChannelTriage {
		t.Fatal("expected channel triage default on with bg telegram")
	}
	if !cfg.ParserSourcePriority {
		t.Fatal("expected source priority default on")
	}
	if !cfg.ParserIntentClassify {
		t.Fatal("expected intent classify default on")
	}
}

func TestAcceptQualitySourceWarningsLander(t *testing.T) {
	t.Parallel()
	w := AcceptQualitySourceWarnings(Config{Source: "forum,lander,reddit"}, true)
	if len(w) != 1 || w[0] == "" {
		t.Fatalf("warnings=%v", w)
	}
}

func TestAcceptQualitySourceWarningsLanderOutreachProd(t *testing.T) {
	t.Parallel()
	w := AcceptQualitySourceWarnings(Config{Source: "forum,lander", ParserLanderOutreach: true}, true)
	found := false
	for _, s := range w {
		if strings.Contains(s, "PARSER_LANDER_OUTREACH=true") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected lander outreach prod warning, got %v", w)
	}
}

func TestAcceptQualitySourceWarningsGitHubOptIn(t *testing.T) {
	t.Parallel()
	w := AcceptQualitySourceWarnings(Config{Source: "forum,github,reddit"}, true)
	found := false
	for _, s := range w {
		if strings.Contains(s, "PARSER_GITHUB_ENABLED=true") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected github opt-in warning, got %v", w)
	}
}

func TestAcceptQualityGitHubErrorsProdOnly(t *testing.T) {
	t.Parallel()
	cfg := Config{Source: "forum,github,reddit"}
	if devErrs := AcceptQualityGitHubErrors(cfg, false); len(devErrs) != 0 {
		t.Fatalf("dev errs=%v", devErrs)
	}
	prodErrs := AcceptQualityGitHubErrors(cfg, true)
	if len(prodErrs) != 1 {
		t.Fatalf("prod errs=%v", prodErrs)
	}
	cfg.ParserGitHubEnabled = true
	if prodErrs := AcceptQualityGitHubErrors(cfg, true); len(prodErrs) != 0 {
		t.Fatalf("enabled prod errs=%v", prodErrs)
	}
}

func TestAcceptQualityBundleMissing(t *testing.T) {
	t.Parallel()
	cfg := Config{
		CRMWebhookEnabled:           true,
		GeminiAPIKey:                "key",
		ParserGeminiDefer:           true,
		BGTelegramEnabled:           true,
		ParserICPClassify:           true,
		ParserEmbedPrescan:          true,
		ParserSourceDisableGovernor: true,
		ParserChannelTriage:         true,
		ParserSourcePriority:        true,
		ParserIntentClassify:        true,
	}
	if !AcceptQualityBundleOK(cfg) {
		t.Fatalf("missing=%v", AcceptQualityBundleMissing(cfg))
	}
}

func TestAcceptQualityBundleErrorsProdOnly(t *testing.T) {
	t.Parallel()
	incomplete := Config{
		CRMWebhookEnabled: true,
		GeminiAPIKey:      "key",
		ParserGeminiDefer: true,
	}
	if devErrs := AcceptQualityBundleErrors(incomplete, false); len(devErrs) != 0 {
		t.Fatalf("dev errs=%v", devErrs)
	}
	prodErrs := AcceptQualityBundleErrors(incomplete, true)
	if len(prodErrs) != 1 {
		t.Fatalf("prod errs=%v", prodErrs)
	}
}

func TestAcceptQualitySourceWarningsRedditFallback(t *testing.T) {
	t.Parallel()
	cfg := Config{Source: "forum", ProxyURLs: []string{"http://proxy:8080"}}
	w := AcceptQualitySourceWarnings(cfg, true)
	if len(w) != 1 {
		t.Fatalf("warnings=%v", w)
	}
	if !strings.Contains(w[0], "reddit") {
		t.Fatalf("expected reddit warning, got %q", w[0])
	}
	// Reddit active removes the warning; empty proxy list also removes it.
	cfg2 := Config{Source: "forum,reddit", ProxyURLs: []string{"http://proxy:8080"}}
	if w2 := AcceptQualitySourceWarnings(cfg2, true); len(w2) != 0 {
		t.Fatalf("expected no warning, got %v", w2)
	}
	cfg3 := Config{Source: "forum"}
	if w3 := AcceptQualitySourceWarnings(cfg3, true); len(w3) != 0 {
		t.Fatalf("expected no warning without proxy, got %v", w3)
	}
}
