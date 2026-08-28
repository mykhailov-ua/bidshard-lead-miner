package gemini

import "strings"

// DefaultModel is the stable Flash model for new Gemini API keys (Google AI Studio / deprecations).
// See https://ai.google.dev/gemini-api/docs/models and gemini-2.5-flash shutdown Oct 2026.
const DefaultModel = "gemini-3.6-flash"

// DeprecatedModelWarning returns a config-check warning when model is blocked for new API keys.
func DeprecatedModelWarning(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return ""
	}
	if strings.Contains(m, "2.5-flash") || strings.Contains(m, "2.0-flash") {
		return "GEMINI_MODEL=" + model + " unavailable for new API keys; use " + DefaultModel + " (https://ai.google.dev/gemini-api/docs/models)"
	}
	return ""
}
