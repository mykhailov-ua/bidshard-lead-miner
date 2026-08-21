package app

import (
	"context"

	"github.com/bidshard/parser/internal/config"
)

// RunTelegramWeb crawls websites linked from Telegram channels for LPR contacts.
func RunTelegramWeb(ctx context.Context, cfg config.Config) error {
	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		return err
	}
	defer deps.flushStore(ctx)
	defer deps.closeMongo(ctx)
	return runTelegramWebOnce(ctx, cfg, deps)
}
