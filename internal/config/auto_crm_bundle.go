package config

import "strings"

// AutoCRMBundleOK reports defer + CRM-after-analysis + geo + lead status for full auto CRM.
func AutoCRMBundleOK(cfg Config) bool {
	return len(AutoCRMBundleMissing(cfg)) == 0
}

// AutoCRMBundleMissing lists env toggles required for the auto-crm preset.
func AutoCRMBundleMissing(cfg Config) []string {
	if !crmWebhookActive(cfg) {
		return nil
	}
	var missing []string
	if !cfg.ParserGeminiDefer {
		missing = append(missing, "PARSER_GEMINI_DEFER=true")
	}
	if !cfg.CRMWebhookAfterAnalysis {
		missing = append(missing, "PARSER_CRM_WEBHOOK_AFTER_ANALYSIS=true")
	}
	if !cfg.ParserGeoClassify {
		missing = append(missing, "PARSER_GEO_CLASSIFY=true")
	}
	if !cfg.ParserLeadStatusEnabled {
		missing = append(missing, "PARSER_LEAD_STATUS_ENABLED=true")
	}
	if strings.TrimSpace(cfg.GeminiAPIKey) == "" {
		missing = append(missing, "GEMINI_API_KEY set")
	}
	return missing
}

// AutoCRMBundleErrors are hard config check failures for defer+CRM without the full bundle.
func AutoCRMBundleErrors(cfg Config, prodProfile bool) []string {
	if !crmWebhookActive(cfg) || !cfg.ParserGeminiDefer {
		return nil
	}
	if AutoCRMBundleOK(cfg) {
		return nil
	}
	if !prodProfile {
		return nil
	}
	missing := AutoCRMBundleMissing(cfg)
	if len(missing) == 0 {
		return nil
	}
	return []string{"prod auto-crm bundle incomplete: " + strings.Join(missing, ", ")}
}
