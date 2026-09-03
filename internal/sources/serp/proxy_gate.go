package serp

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/config"
)

// HarvestProxyReady reports whether SERP harvest may run given proxy config.
// Skips when serp is listed in PARSER_PROXY_SOURCES but PARSER_PROXY_LIST is empty.
// Default proxy scope (forum,tgweb,lander,webpain) keeps serp on direct egress.
func HarvestProxyReady(cfg config.Config) bool {
	if len(cfg.ProxyURLs) > 0 {
		return true
	}
	for _, s := range cfg.ProxySources {
		if strings.EqualFold(s, "serp") {
			return false
		}
	}
	return true
}

// RunBGHarvest runs fn when harvest is allowed; otherwise logs once and returns nil.
func RunBGHarvest(ctx context.Context, cfg config.Config, job string, fn func(context.Context) error) error {
	if !HarvestProxyReady(cfg) {
		slog.Info("serp harvest skipped; proxy required but PARSER_PROXY_LIST empty", "job", job)
		return nil
	}
	return fn(ctx)
}
