package forum

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Adapter struct {
	seedPath string
	fetcher  *Fetcher
}

func NewAdapter(cfg config.Config, fetcher *Fetcher) *Adapter {
	if fetcher == nil {
		fetcher = NewFetcher(cfg.HTTPTimeout, cfg.ForumBaseURL)
	}
	return &Adapter{
		seedPath: cfg.ForumSeedPath,
		fetcher:  fetcher,
	}
}

func (a *Adapter) Name() string {
	return "forum"
}

func (a *Adapter) Collect(ctx context.Context, emit EmitFunc) error {
	urls, err := LoadThreadURLs(a.seedPath)
	if err != nil {
		return err
	}

	start := time.Now()
	emitted := 0

	for _, threadURL := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		html, err := a.fetcher.Get(ctx, threadURL)
		if err != nil {
			slog.Warn("forum fetch failed", "url", threadURL, "error", err)
			continue
		}

		for _, post := range ParsePostsFromHTML(html) {
			if !HasPainSignal(post.Body) {
				continue
			}
			contacts := extract.Extract(post.Body)
			if contacts.Rejected || len(contacts.Contacts) == 0 {
				continue
			}
			primary := extract.FormatAll(contacts.Contacts)[0]
			item := model.RawItem{
				Source:  sourceName(threadURL),
				Raw:     post.Body,
				Contact: primary,
				Title:   post.Author,
			}
			if err := emit(ctx, item); err != nil {
				return err
			}
			emitted++
		}
	}

	slog.Info("forum crawl finished",
		"threads", len(urls),
		"emitted", emitted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func sourceName(threadURL string) string {
	host := hostFromURL(threadURL)
	if idx := strings.Index(threadURL, "/threads/"); idx >= 0 {
		slug := strings.TrimPrefix(threadURL[idx+len("/threads/"):], "/")
		if slash := strings.Index(slug, "/"); slash >= 0 {
			slug = slug[:slash]
		}
		return "forum:" + host + "/" + slug
	}
	return "forum:" + host
}
