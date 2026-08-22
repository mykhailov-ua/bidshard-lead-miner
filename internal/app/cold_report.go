package app

import (
	"context"
	"fmt"

	"github.com/bidshard/parser/internal/config"
)

// RunColdReport triggers one cold-path junk report cycle (ops/debug).
func RunColdReport(ctx context.Context, cfg config.Config) error {
	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		return err
	}
	defer deps.closeMongo(ctx)
	if deps.coldPath == nil {
		return fmt.Errorf("cold path not configured: set GEMINI_API_KEY and MONGO_URI")
	}
	deps.coldPath.RunReportOnce(ctx)
	return nil
}
