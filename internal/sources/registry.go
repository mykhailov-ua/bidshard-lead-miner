package sources

import (
	"context"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/sources/lander"
	"github.com/bidshard/parser/internal/sources/supply"
)

func Build(cfg config.Config) []Source {
	switch cfg.Source {
	case "supply":
		return []Source{wrapSupply(supply.NewCrawler(cfg, nil))}
	case "forum":
		return []Source{wrapForum(forum.NewAdapter(cfg, nil))}
	case "lander":
		return []Source{wrapLander(lander.NewCrawler(cfg, nil, nil))}
	default:
		return DefaultStubs()
	}
}

type supplySource struct {
	inner *supply.Crawler
}

func wrapSupply(inner *supply.Crawler) Source {
	return &supplySource{inner: inner}
}

func (s *supplySource) Name() string {
	return s.inner.Name()
}

func (s *supplySource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type forumSource struct {
	inner *forum.Adapter
}

func wrapForum(inner *forum.Adapter) Source {
	return &forumSource{inner: inner}
}

func (s *forumSource) Name() string {
	return s.inner.Name()
}

func (s *forumSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}

type landerSource struct {
	inner *lander.Crawler
}

func wrapLander(inner *lander.Crawler) Source {
	return &landerSource{inner: inner}
}

func (s *landerSource) Name() string {
	return s.inner.Name()
}

func (s *landerSource) Collect(ctx context.Context, emit EmitFunc) error {
	return s.inner.Collect(ctx, func(ctx context.Context, item model.RawItem) error {
		return emit(ctx, item)
	})
}
