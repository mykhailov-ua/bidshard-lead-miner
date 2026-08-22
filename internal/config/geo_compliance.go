package config

import "strings"

// GeoComplianceWarnings reports RU/BY compliance gaps when CRM webhook is active.
func GeoComplianceWarnings(cfg Config) []string {
	if !crmWebhookActive(cfg) {
		return nil
	}
	var out []string
	if !cfg.ParserGeoClassify {
		out = append(out, "PARSER_CRM_WEBHOOK active but PARSER_GEO_CLASSIFY=false - RU/BY may reach CRM before warm-path geo")
		return out
	}
	hotPathCRM := !cfg.CRMWebhookAfterAnalysis || !cfg.ParserGeminiDefer
	// Hot-path webhook on accept: defer without sync geo leaves a gap until warm path runs.
	if hotPathCRM && cfg.ParserGeminiDefer && !cfg.ParserGeminiSyncGeo {
		out = append(out, "PARSER_GEMINI_DEFER=true without PARSER_GEMINI_SYNC_GEO=true - leads may reach CRM before geo; set PARSER_CRM_WEBHOOK_AFTER_ANALYSIS=true or enable sync geo")
	}
	return out
}

// SyncGeoGateConfigured reports whether Gemini geo runs on the hot path before Mongo write.
func SyncGeoGateConfigured(cfg Config) bool {
	if cfg.GeminiAPIKey == "" || !cfg.ParserGeoClassify {
		return false
	}
	return !cfg.ParserGeminiDefer || cfg.ParserGeminiSyncGeo
}

// GeoComplianceErrors are hard config check failures when CRM webhook risks RU/BY inbox leak.
func GeoComplianceErrors(cfg Config, prodProfile bool) []string {
	if !crmWebhookActive(cfg) {
		return nil
	}
	if !prodProfile && !cfg.CRMWebhookEnabled {
		return nil
	}
	var out []string
	if cfg.GeminiAPIKey == "" {
		out = append(out, "PARSER_CRM_WEBHOOK active but GEMINI_API_KEY empty - enable geo classify or disable webhook")
		return out
	}
	if !cfg.ParserGeoClassify {
		out = append(out, "PARSER_CRM_WEBHOOK active but PARSER_GEO_CLASSIFY=false")
	}
	if !cfg.ParserLeadStatusEnabled {
		out = append(out, "PARSER_CRM_WEBHOOK active but PARSER_LEAD_STATUS_ENABLED=false - CRM inbox needs status=new")
	}
	hotPathCRM := !cfg.CRMWebhookAfterAnalysis || !cfg.ParserGeminiDefer
	if hotPathCRM && cfg.ParserGeminiDefer && !cfg.ParserGeminiSyncGeo {
		out = append(out, "defer+CRM without sync geo or PARSER_CRM_WEBHOOK_AFTER_ANALYSIS=true")
	}
	if prodProfile && !SyncGeoGateConfigured(cfg) && (!cfg.ParserGeminiDefer || !cfg.CRMWebhookAfterAnalysis || !cfg.ParserGeoClassify) {
		out = append(out, "prod: no geo gate before CRM (enable sync geo or defer+after-analysis webhook)")
	}
	return out
}

func crmWebhookActive(cfg Config) bool {
	return cfg.CRMWebhookEnabled || strings.TrimSpace(cfg.CRMWebhookURL) != ""
}
