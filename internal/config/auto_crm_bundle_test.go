package config

import (
	"strings"
	"testing"
)

func TestAutoCRMBundleOK(t *testing.T) {
	t.Parallel()

	ok := Config{
		CRMWebhookEnabled:       true,
		ParserGeminiDefer:       true,
		CRMWebhookAfterAnalysis: true,
		ParserGeoClassify:       true,
		ParserLeadStatusEnabled: true,
		GeminiAPIKey:            "key",
	}
	if !AutoCRMBundleOK(ok) {
		t.Fatal("expected full auto-crm bundle")
	}

	missing := ok
	missing.CRMWebhookAfterAnalysis = false
	if AutoCRMBundleOK(missing) {
		t.Fatal("expected missing after-analysis webhook")
	}
	if got := AutoCRMBundleMissing(missing); len(got) != 1 || got[0] != "PARSER_CRM_WEBHOOK_AFTER_ANALYSIS=true" {
		t.Fatalf("missing=%v", got)
	}
}

func TestAutoCRMBundleErrorsProdOnly(t *testing.T) {
	t.Parallel()

	incomplete := Config{
		CRMWebhookEnabled: true,
		ParserGeminiDefer: true,
		GeminiAPIKey:      "key",
	}
	if devErrs := AutoCRMBundleErrors(incomplete, false); len(devErrs) != 0 {
		t.Fatalf("dev profile should warn only: %v", devErrs)
	}
	prodErrs := AutoCRMBundleErrors(incomplete, true)
	if len(prodErrs) != 1 {
		t.Fatalf("prod errors=%v", prodErrs)
	}
	if !strings.Contains(prodErrs[0], "PARSER_CRM_WEBHOOK_AFTER_ANALYSIS") {
		t.Fatalf("error=%q", prodErrs[0])
	}
}
