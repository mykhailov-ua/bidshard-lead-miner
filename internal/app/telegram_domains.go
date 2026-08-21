package app

import (
	"context"
	"log/slog"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/tgweb"
)

// RunTelegramDomainsReset clears crawled_at for specific domains so they can be re-crawled.
func RunTelegramDomainsReset(ctx context.Context, cfg config.Config, domains []string) error {
	_ = ctx
	updated, err := tgweb.ClearCrawledAt(cfg.TelegramDomainsPath, domains)
	if err != nil {
		return err
	}
	slog.Info("telegram domains reset",
		"path", cfg.TelegramDomainsPath,
		"domains", domains,
		"updated", updated,
	)
	return nil
}

// RunTelegramDomainsPrune removes invalid hosts from the Telegram domain registry.
func RunTelegramDomainsPrune(ctx context.Context, cfg config.Config) error {
	_ = ctx
	kept, removed, err := tgweb.PruneInvalidDomains(cfg.TelegramDomainsPath)
	if err != nil {
		return err
	}
	slog.Info("telegram domains pruned",
		"path", cfg.TelegramDomainsPath,
		"kept", kept,
		"removed", removed,
	)
	return nil
}
