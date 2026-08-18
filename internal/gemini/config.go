package gemini

import (
	"net/http"

	"github.com/bidshard/parser/internal/config"
)

func LimitConfigFrom(cfg config.Config) LimitConfig {
	lc := DefaultLimitConfig(cfg.GeminiModel)
	lc.ModelLimits = lc.ModelLimits.withOverrides(cfg.GeminiRPM, cfg.GeminiTPM, cfg.GeminiRPD, cfg.GeminiMaxRetries)
	if cfg.GeminiEmbedRPM > 0 {
		lc.EmbedRPM = cfg.GeminiEmbedRPM
	}
	return lc
}

func ClientOptionsFrom(cfg config.Config) []Option {
	opts := []Option{WithLimitConfig(LimitConfigFrom(cfg))}
	if cfg.GeminiRequestTimeout > 0 {
		opts = append(opts, WithHTTPClient(&http.Client{Timeout: cfg.GeminiRequestTimeout}))
	}
	return opts
}
