package config

import "testing"

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
