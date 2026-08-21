package gemini

import (
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/httpclient"
)

func LimitConfigFrom(cfg config.Config) LimitConfig {
	// GEMINI_RPM/TPM/RPD/MaxRetries at 0 keep DefaultLimitConfig(model) tier defaults.
	lc := DefaultLimitConfig(cfg.GeminiModel)
	lc.ModelLimits = lc.withOverrides(cfg.GeminiRPM, cfg.GeminiTPM, cfg.GeminiRPD, cfg.GeminiMaxRetries)
	if cfg.GeminiEmbedRPM > 0 {
		lc.EmbedRPM = cfg.GeminiEmbedRPM
	}
	lc.QuotaSplit = QuotaSplit{
		Critical: cfg.GeminiQuotaCriticalPct,
		High:     cfg.GeminiQuotaHighPct,
		Normal:   cfg.GeminiQuotaNormalPct,
		Low:      cfg.GeminiQuotaLowPct,
	}.Normalize()
	return lc
}

func ClientOptionsFrom(cfg config.Config) []Option {
	opts := []Option{WithLimitConfig(LimitConfigFrom(cfg))}
	if cfg.GeminiRequestTimeout > 0 {
		opts = append(opts, WithHTTPClient(httpclient.ClientWithSharedTransport(cfg.GeminiRequestTimeout)))
	}
	return opts
}
