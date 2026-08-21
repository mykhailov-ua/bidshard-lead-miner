package config

import "os"

// applyComplianceDefaults fills geo/CRM safety when webhook is active and env keys are unset.
// Explicit env always wins (including PARSER_GEO_CLASSIFY=false).
func applyComplianceDefaults(cfg *Config) {
	if !crmWebhookActive(*cfg) {
		return
	}
	if envUnset("PARSER_LEAD_STATUS_ENABLED") {
		cfg.ParserLeadStatusEnabled = true
	}
	if cfg.GeminiAPIKey == "" {
		return
	}
	if envUnset("PARSER_GEO_CLASSIFY") {
		cfg.ParserGeoClassify = true
	}
	if !cfg.ParserGeminiDefer {
		return
	}
	if envUnset("PARSER_CRM_WEBHOOK_AFTER_ANALYSIS") {
		cfg.CRMWebhookAfterAnalysis = true
	}
	// Warm-path geo + after-analysis webhook: inline sync geo not required on accept.
	if !cfg.CRMWebhookAfterAnalysis && envUnset("PARSER_GEMINI_SYNC_GEO") {
		cfg.ParserGeminiSyncGeo = true
	}
	if envUnset("PARSER_CRM_WEBHOOK_HEAT_MIN") {
		// Staging-safe default; set hot in prod (.env.prod.example).
		cfg.CRMWebhookHeatMin = "warm"
	}
}

func envUnset(key string) bool {
	return os.Getenv(key) == ""
}
