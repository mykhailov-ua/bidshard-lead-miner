package config

import "testing"

func TestGeoComplianceWarnings(t *testing.T) {
	t.Parallel()

	base := Config{
		CRMWebhookEnabled:   true,
		ParserGeoClassify:   true,
		ParserGeminiDefer:   true,
		ParserGeminiSyncGeo: true,
		GeminiAPIKey:        "key",
	}
	if w := GeoComplianceWarnings(base); len(w) != 0 {
		t.Fatalf("expected no warnings, got %v", w)
	}

	noGeo := base
	noGeo.ParserGeoClassify = false
	if w := GeoComplianceWarnings(noGeo); len(w) != 1 {
		t.Fatalf("expected one warning for missing geo classify, got %v", w)
	}

	deferNoSync := base
	deferNoSync.ParserGeminiSyncGeo = false
	if w := GeoComplianceWarnings(deferNoSync); len(w) != 1 {
		t.Fatalf("expected defer/sync warning, got %v", w)
	}

	deferWebhook := deferNoSync
	deferWebhook.CRMWebhookAfterAnalysis = true
	if w := GeoComplianceWarnings(deferWebhook); len(w) != 0 {
		t.Fatalf("expected no warning with after-analysis webhook, got %v", w)
	}

	if w := GeoComplianceWarnings(Config{}); len(w) != 0 {
		t.Fatalf("expected no warnings without CRM, got %v", w)
	}
}

func TestSyncGeoGateConfigured(t *testing.T) {
	t.Parallel()

	if SyncGeoGateConfigured(Config{}) {
		t.Fatal("expected false without key and flag")
	}
	if !SyncGeoGateConfigured(Config{
		GeminiAPIKey:        "key",
		ParserGeoClassify:   true,
		ParserGeminiDefer:   true,
		ParserGeminiSyncGeo: true,
	}) {
		t.Fatal("expected true with defer+sync geo")
	}
	if !SyncGeoGateConfigured(Config{
		GeminiAPIKey:      "key",
		ParserGeoClassify: true,
		ParserGeminiDefer: false,
	}) {
		t.Fatal("expected true with inline geo (defer off)")
	}
	if SyncGeoGateConfigured(Config{
		GeminiAPIKey:        "key",
		ParserGeoClassify:   true,
		ParserGeminiDefer:   true,
		ParserGeminiSyncGeo: false,
	}) {
		t.Fatal("expected false with defer and no sync geo")
	}
}
