package config

import (
	"os"
	"strings"
)

// applyAcceptQualityDefaults turns on precision gates for defer+CRM when env keys are unset.
func applyAcceptQualityDefaults(cfg *Config) {
	if cfg == nil || !crmWebhookActive(*cfg) || strings.TrimSpace(cfg.GeminiAPIKey) == "" {
		return
	}
	if envUnset("PARSER_ICP_CLASSIFY") {
		cfg.ParserICPClassify = true
	}
	if envUnset("PARSER_EMBED_PRESCAN") {
		cfg.ParserEmbedPrescan = true
	}
	if envUnset("PARSER_SOURCE_DISABLE_GOVERNOR") {
		cfg.ParserSourceDisableGovernor = true
	}
	if cfg.BGTelegramEnabled && envUnset("PARSER_CHANNEL_TRIAGE") {
		cfg.ParserChannelTriage = true
	}
	if envUnset("PARSER_SOURCE_PRIORITY") {
		cfg.ParserSourcePriority = true
	}
	if envUnset("PARSER_INTENT_CLASSIFY") {
		cfg.ParserIntentClassify = true
	}
}

// AcceptQualityBundleMissing lists precision env toggles for defer+CRM prod handoff.
func AcceptQualityBundleMissing(cfg Config) []string {
	if !crmWebhookActive(cfg) || !cfg.ParserGeminiDefer || strings.TrimSpace(cfg.GeminiAPIKey) == "" {
		return nil
	}
	var missing []string
	if !cfg.ParserICPClassify {
		missing = append(missing, "PARSER_ICP_CLASSIFY=true")
	}
	if !cfg.ParserEmbedPrescan {
		missing = append(missing, "PARSER_EMBED_PRESCAN=true")
	}
	if !cfg.ParserSourceDisableGovernor {
		missing = append(missing, "PARSER_SOURCE_DISABLE_GOVERNOR=true")
	}
	if cfg.BGTelegramEnabled && !cfg.ParserChannelTriage {
		missing = append(missing, "PARSER_CHANNEL_TRIAGE=true")
	}
	if !cfg.ParserSourcePriority {
		missing = append(missing, "PARSER_SOURCE_PRIORITY=true")
	}
	if !cfg.ParserIntentClassify {
		missing = append(missing, "PARSER_INTENT_CLASSIFY=true")
	}
	return missing
}

// AcceptQualityBundleOK reports hot-path ICP/embed prescan + junk feedback loops for CRM.
func AcceptQualityBundleOK(cfg Config) bool {
	return len(AcceptQualityBundleMissing(cfg)) == 0
}

// AcceptQualitySourceWarnings flags high-junk source mixes on prod profiles.
func AcceptQualitySourceWarnings(cfg Config, prodProfile bool) []string {
	if !prodProfile {
		return nil
	}
	active := parseSourceNamesSimple(cfg.Source)
	var warnings []string
	if containsSourceName(active, "lander") {
		warnings = append(warnings, "prod: lander in PARSER_SOURCE is intel-only unless PARSER_LANDER_OUTREACH=true")
	}
	if containsSourceName(active, "github") {
		warnings = append(warnings, "prod: github is opt-in and noisy (CKAN/keitaroinc collisions); prefer forum,reddit,serp or deploy github pain gate")
	}
	if containsSourceName(active, "discord") && envDefaultEmpty("DISCORD_BOT_TOKEN") {
		warnings = append(warnings, "prod: discord in PARSER_SOURCE but DISCORD_BOT_TOKEN unset")
	}
	return warnings
}

func parseSourceNamesSimple(raw string) []string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil
	}
	if raw == "all" {
		return []string{"forum", "supply", "reddit", "discord", "serp"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "warrior" {
			p = "forum"
		}
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func containsSourceName(names []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, n := range names {
		if strings.EqualFold(n, want) {
			return true
		}
	}
	return false
}

func envDefaultEmpty(key string) bool {
	return os.Getenv(key) == ""
}
