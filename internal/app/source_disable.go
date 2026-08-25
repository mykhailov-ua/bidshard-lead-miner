package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/sourcedisable"
)

// RunSourceDisableGovernor persists families with zero accepts once raw volume crosses minRaw.
func RunSourceDisableGovernor(ctx context.Context, cfg config.Config, stats *sink.SourceStatsStore) error {
	if stats == nil {
		return nil
	}
	docs, err := stats.ListAll(ctx)
	if err != nil {
		return err
	}
	minRaw := cfg.SourceDisableMinRaw
	if minRaw <= 0 {
		minRaw = 100
	}
	disabled := sourcedisable.Evaluate(docs, minRaw)
	if len(disabled) == 0 {
		return nil
	}
	audit := make([]string, 0, len(disabled))
	for _, src := range disabled {
		audit = append(audit, fmt.Sprintf("%s: raw>=%d accepted=0", src, minRaw))
	}
	path := cfg.DisabledSourcesPath
	if path == "" {
		path = sourcedisable.DefaultPath
	}
	if err := sourcedisable.Save(path, disabled, audit); err != nil {
		return err
	}
	slog.Info("source disable governor", "disabled", len(disabled), "path", path)
	return nil
}
