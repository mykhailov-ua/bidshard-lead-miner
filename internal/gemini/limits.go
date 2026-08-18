package gemini

import (
	"strings"
	"time"
)

// ModelLimits mirrors Gemini API quotas (per project). Defaults target free-tier Flash.
// Override via GEMINI_RPM / GEMINI_RPD / GEMINI_TPM env or LimitsForModel presets.
// Official docs: https://ai.google.dev/gemini-api/docs/rate-limits
type ModelLimits struct {
	RPM        int
	TPM        int
	RPD        int
	MaxRetries int
	RetryBase  time.Duration
	RetryMax   time.Duration
}

type LimitConfig struct {
	ModelLimits
	EmbedRPM int // embedContent shares project quota; separate bucket optional
}

func DefaultLimitConfig(model string) LimitConfig {
	limits := LimitsForModel(model)
	return LimitConfig{
		ModelLimits: limits,
		EmbedRPM:    max(1, limits.RPM/3),
	}
}

func LimitsForModel(model string) ModelLimits {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "flash-lite"), strings.Contains(m, "flash_lite"):
		return ModelLimits{RPM: 15, TPM: 250_000, RPD: 1000, MaxRetries: 4, RetryBase: 2 * time.Second, RetryMax: 60 * time.Second}
	case strings.Contains(m, "2.5-flash"), strings.Contains(m, "2.5_flash"):
		return ModelLimits{RPM: 10, TPM: 250_000, RPD: 250, MaxRetries: 4, RetryBase: 2 * time.Second, RetryMax: 60 * time.Second}
	case strings.Contains(m, "2.5-pro"), strings.Contains(m, "2.5_pro"):
		return ModelLimits{RPM: 5, TPM: 250_000, RPD: 100, MaxRetries: 3, RetryBase: 3 * time.Second, RetryMax: 90 * time.Second}
	case strings.Contains(m, "2.0-flash"), strings.Contains(m, "2.0_flash"):
		return ModelLimits{RPM: 15, TPM: 1_000_000, RPD: 1500, MaxRetries: 4, RetryBase: 2 * time.Second, RetryMax: 60 * time.Second}
	default:
		// Conservative fallback for unknown / preview models.
		return ModelLimits{RPM: 10, TPM: 250_000, RPD: 500, MaxRetries: 4, RetryBase: 2 * time.Second, RetryMax: 60 * time.Second}
	}
}

func (l ModelLimits) withOverrides(rpm, tpm, rpd, maxRetries int) ModelLimits {
	if rpm > 0 {
		l.RPM = rpm
	}
	if tpm > 0 {
		l.TPM = tpm
	}
	if rpd > 0 {
		l.RPD = rpd
	}
	if maxRetries > 0 {
		l.MaxRetries = maxRetries
	}
	if l.RetryBase <= 0 {
		l.RetryBase = 2 * time.Second
	}
	if l.RetryMax <= 0 {
		l.RetryMax = 60 * time.Second
	}
	return l
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// EstimateTokens is a cheap heuristic (≈4 chars/token). Used for TPM budgeting.
func EstimateTokens(parts ...string) int {
	var n int
	for _, p := range parts {
		n += len(p)
	}
	tokens := n / 4
	if tokens < 1 {
		return 1
	}
	// Reserve headroom for JSON schema + model output.
	return tokens + 256
}
