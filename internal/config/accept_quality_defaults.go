package config

import (
	"os"
	"strings"
)

// applyAcceptQualityDefaults turns on precision gates for defer+Gemini when env keys are unset.
func applyAcceptQualityDefaults(cfg *Config) {
	if cfg == nil || !precisionBundleEligible(*cfg) {
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

// AcceptQualityBundleMissing lists precision env toggles for defer+Gemini soak/prod handoff.
func AcceptQualityBundleMissing(cfg Config) []string {
	if !precisionBundleEligible(cfg) {
		return nil
	}
	var missing []string
	if !cfg.ParserWarmEmbedPrescan {
		missing = append(missing, "PARSER_WARM_EMBED_PRESCAN=true")
	}
	if !cfg.ParserWarmEmbedCluster {
		missing = append(missing, "PARSER_WARM_EMBED_CLUSTER=true")
	}
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

// AcceptQualityBundleErrors are hard config check failures for defer+Gemini without precision gates.
func AcceptQualityBundleErrors(cfg Config, prodProfile bool) []string {
	if !precisionBundleEligible(cfg) {
		return nil
	}
	if AcceptQualityBundleOK(cfg) {
		return nil
	}
	if !prodProfile {
		return nil
	}
	missing := AcceptQualityBundleMissing(cfg)
	if len(missing) == 0 {
		return nil
	}
	return []string{"prod accept-quality bundle incomplete: " + strings.Join(missing, ", ")}
}

// AcceptQualityGitHubErrors are hard config check failures when github is listed without opt-in.
func AcceptQualityGitHubErrors(cfg Config, prodProfile bool) []string {
	if !prodProfile {
		return nil
	}
	active := parseSourceNamesSimple(cfg.Source)
	if !containsSourceName(active, "github") || cfg.ParserGitHubEnabled {
		return nil
	}
	return []string{"prod: github in PARSER_SOURCE requires PARSER_GITHUB_ENABLED=true"}
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
	if cfg.ParserLanderOutreach {
		warnings = append(warnings, "prod: PARSER_LANDER_OUTREACH=true converts lander from intel-only to CRM leads; verify competitor domain blacklist is loaded")
	}
	if containsSourceName(active, "github") {
		if cfg.ParserGitHubEnabled {
			warnings = append(warnings, "prod: github enabled but noisy (CKAN/keitaroinc collisions); pain gate active in processor")
		} else {
			warnings = append(warnings, "prod: github in PARSER_SOURCE requires PARSER_GITHUB_ENABLED=true")
		}
	}
	if containsSourceName(active, "discord") && envDefaultEmpty("DISCORD_BOT_TOKEN") {
		warnings = append(warnings, "prod: discord in PARSER_SOURCE but DISCORD_BOT_TOKEN unset")
	}
	if containsSourceName(active, "forum") && !containsSourceName(active, "reddit") && len(cfg.ProxyURLs) > 0 && cfg.ProxyEnabledForSource("forum") {
		warnings = append(warnings, "prod: forum uses proxy but reddit is not in PARSER_SOURCE; reddit is direct-egress public coverage when proxies cool")
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

func precisionBundleEligible(cfg Config) bool {
	return cfg.ParserGeminiDefer && strings.TrimSpace(cfg.GeminiAPIKey) != ""
}
