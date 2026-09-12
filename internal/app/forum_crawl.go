package app

import (
	"context"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sources/forum"
)

func runForumCrawlOnce(ctx context.Context, cfg config.Config, deps *runtimeDeps) error {
	return forum.RunBGCrawl(ctx, cfg, func(ctx context.Context) error {
		adapter := forum.NewAdapter(cfg, nil)
		return runCollectOnce(ctx, cfg, deps, adapter.Name(), func(ctx context.Context, emit func(ctx context.Context, item model.RawItem) error) error {
			return adapter.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
				return emit(ctx, item)
			})
		})
	})
}
