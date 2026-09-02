package gemini

import (
	"fmt"
	"strings"
)

// DefaultModel is the stable Flash model for new Gemini API keys (Google AI Studio / deprecations).
// See https://ai.google.dev/gemini-api/docs/models and gemini-2.5-flash shutdown Oct 2026.
const DefaultModel = "gemini-3.6-flash"

// DeprecatedModelWarning returns a config-check warning when model is blocked for new API keys.
func DeprecatedModelWarning(model string) string {
	if err := DeprecatedModelConfigError(model); err != nil {
		return err.Error()
	}
	return ""
}

// DeprecatedModelConfigError is a hard config failure for blocked Gemini model IDs.
func DeprecatedModelConfigError(model string) error {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return nil
	}
	if strings.Contains(m, "2.5-flash") || strings.Contains(m, "2.0-flash") {
		return fmt.Errorf("GEMINI_MODEL=%s unavailable for new API keys; use %s (https://ai.google.dev/gemini-api/docs/models)", model, DefaultModel)
	}
	return nil
}
