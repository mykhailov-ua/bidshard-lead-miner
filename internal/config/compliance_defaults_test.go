package config

import "testing"

func TestApplyComplianceDefaultsCRMWebhook(t *testing.T) {
	t.Parallel()

	cfg := Config{
		CRMWebhookEnabled: true,
		GeminiAPIKey:      "key",
		ParserGeminiDefer: true,
	}
	applyComplianceDefaults(&cfg)
	if !cfg.ParserGeoClassify {
		t.Fatal("expected ParserGeoClassify default true with CRM webhook")
	}
	if !cfg.CRMWebhookAfterAnalysis {
		t.Fatal("expected CRMWebhookAfterAnalysis default true with defer+CRM")
	}
	if !cfg.ParserLeadStatusEnabled {
		t.Fatal("expected ParserLeadStatusEnabled default true with CRM webhook")
	}
	if cfg.ParserGeminiSyncGeo {
		t.Fatal("expected sync geo off when after-analysis webhook defaults on")
	}
	if cfg.CRMWebhookHeatMin != "warm" {
		t.Fatalf("heat_min=%q want warm", cfg.CRMWebhookHeatMin)
	}
}

func TestApplyComplianceDefaultsRespectsExplicitEnv(t *testing.T) {
	t.Setenv("PARSER_GEO_CLASSIFY", "false")
	t.Setenv("PARSER_CRM_WEBHOOK_AFTER_ANALYSIS", "false")
	t.Setenv("PARSER_LEAD_STATUS_ENABLED", "false")

	cfg := Config{
		CRMWebhookEnabled: true,
		GeminiAPIKey:      "key",
		ParserGeminiDefer: true,
	}
	applyComplianceDefaults(&cfg)
	if cfg.ParserGeoClassify {
		t.Fatal("expected explicit PARSER_GEO_CLASSIFY=false preserved")
	}
	if cfg.CRMWebhookAfterAnalysis {
		t.Fatal("expected explicit after-analysis=false preserved")
	}
	if cfg.ParserLeadStatusEnabled {
		t.Fatal("expected explicit lead status=false preserved")
	}
}

func TestGeoComplianceErrorsProdDeferAfterAnalysis(t *testing.T) {
	t.Parallel()

	cfg := Config{
		CRMWebhookEnabled:       true,
		GeminiAPIKey:            "key",
		ParserGeoClassify:       true,
		ParserGeminiDefer:       true,
		CRMWebhookAfterAnalysis: true,
		ParserLeadStatusEnabled: true,
	}
	if errs := GeoComplianceErrors(cfg, true); len(errs) != 0 {
		t.Fatalf("expected no errors for defer+after-analysis profile, got %v", errs)
	}
}

func TestGeoComplianceErrorsMissingGeo(t *testing.T) {
	t.Parallel()

	cfg := Config{
		CRMWebhookEnabled: true,
		GeminiAPIKey:      "key",
		ParserGeoClassify: false,
	}
	if errs := GeoComplianceErrors(cfg, false); len(errs) == 0 {
		t.Fatal("expected error when geo classify off with CRM webhook")
	}
}
