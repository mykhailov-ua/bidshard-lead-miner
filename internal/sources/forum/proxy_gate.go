package forum

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/config"
)

// CrawlProxyReady reports whether forum HTTP fetch may run given proxy config.
// Skips when forum is listed in PARSER_PROXY_SOURCES but PARSER_PROXY_LIST is empty.
func CrawlProxyReady(cfg config.Config) bool {
	if len(cfg.ProxyURLs) > 0 {
		return true
	}
	for _, s := range cfg.ProxySources {
		if strings.EqualFold(s, "forum") {
			return false
		}
	}
	return true
}

// RunBGCrawl runs fn when forum crawl egress is ready; otherwise logs once and returns nil.
func RunBGCrawl(ctx context.Context, cfg config.Config, fn func(context.Context) error) error {
	if !CrawlProxyReady(cfg) {
		slog.Info("forum crawl skipped; proxy required but PARSER_PROXY_LIST empty")
		return nil
	}
	return fn(ctx)
}
