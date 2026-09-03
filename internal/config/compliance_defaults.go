package config

import (
	"os"
	"strings"
)

// applyComplianceDefaults fills geo/CRM safety when webhook is active and env keys are unset.
// Explicit env always wins (including PARSER_GEO_CLASSIFY=false).
func applyComplianceDefaults(cfg *Config) {
	if crmWebhookActive(*cfg) {
		applyCRMComplianceDefaults(cfg)
		return
	}
	if deferPrecisionEligible(*cfg) {
		applyDeferPrecisionDefaults(cfg)
	}
}

func applyCRMComplianceDefaults(cfg *Config) {
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
	applyWarmEmbedPrecisionDefaults(cfg)
	if envUnset("PARSER_CRM_WEBHOOK_HEAT_MIN") {
		// Staging-safe default; set hot in prod (.env.prod.example).
		cfg.CRMWebhookHeatMin = "warm"
	}
	applyEmailQualityDefaults(cfg)
	applyAcceptQualityDefaults(cfg)
}

// applyDeferPrecisionDefaults turns on warm-path precision gates for defer+Gemini soak
// without requiring CRM webhook (explicit env still wins).
func applyDeferPrecisionDefaults(cfg *Config) {
	applyWarmEmbedPrecisionDefaults(cfg)
	applyAcceptQualityDefaults(cfg)
}

func applyWarmEmbedPrecisionDefaults(cfg *Config) {
	if envUnset("PARSER_WARM_EMBED_PRESCAN") {
		cfg.ParserWarmEmbedPrescan = true
	}
	if envUnset("PARSER_WARM_EMBED_CLUSTER") {
		cfg.ParserWarmEmbedCluster = true
	}
}

func deferPrecisionEligible(cfg Config) bool {
	return cfg.ParserGeminiDefer && strings.TrimSpace(cfg.GeminiAPIKey) != ""
}

// applyEmailQualityDefaults turns on MX gate for CRM handoff when env is unset.
// SMTP verify stays opt-in (slow, often blocked on residential egress).
func applyEmailQualityDefaults(cfg *Config) {
	if !crmWebhookActive(*cfg) {
		return
	}
	if envUnset("PARSER_MX_CHECK") {
		cfg.MXCheck = true
	}
}

func envUnset(key string) bool {
	return os.Getenv(key) == ""
}
