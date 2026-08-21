package forum

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Adapter struct {
	seedPath string
	fetcher  *Fetcher
	workers  int
}

func NewAdapter(cfg config.Config, fetcher *Fetcher) *Adapter {
	if fetcher == nil {
		fetcher = NewFetcherWithConfig(cfg)
	}
	workers := cfg.HTTPWorkers
	if workers <= 0 {
		workers = 10
	}
	return &Adapter{
		seedPath: cfg.ForumSeedPath,
		fetcher:  fetcher,
		workers:  workers,
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
	var (
		mu      sync.Mutex
		emitted int
		visited = make(map[string]struct{})
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(a.workers)

	var processURL func(rawURL string, depth int)
	processURL = func(rawURL string, depth int) {
		if depth > 2 {
			return
		}

		mu.Lock()
		if _, seen := visited[rawURL]; seen {
			mu.Unlock()
			return
		}
		visited[rawURL] = struct{}{}
		mu.Unlock()

		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			html, err := a.fetcher.Get(gCtx, rawURL)
			if err != nil {
				slog.Warn("forum fetch failed", "url", rawURL, "error", err)
				return nil
			}

			discoveredThreads := ParseThreadURLsFromCategory(html)
			for _, discURL := range discoveredThreads {
				processURL(discURL, depth+1)
			}

			for _, post := range ParsePostsFromHTML(html) {
				if !HasPainSignal(post.Body) {
					continue
				}
				contacts := extract.Extract(post.Body)
				if contacts.Rejected {
					continue
				}
				primary := ""
				if len(contacts.Contacts) > 0 {
					primary = extract.FormatAll(contacts.Contacts)[0]
				} else if post.Author != "" {
					primary = "forum:user/" + post.Author
				} else {
					continue
				}

				item := model.RawItem{
					Source:      sourceName(rawURL),
					Raw:         post.Body,
					Contact:     primary,
					Title:       post.Author,
					Username:    post.Author,
					ForumUserID: post.UserID,
					PostedAt:    post.PostedAt,
				}

				mu.Lock()
				if err := emit(gCtx, item); err != nil {
					mu.Unlock()
					return err
				}
				emitted++
				mu.Unlock()
			}
			return nil
		})
	}

	for _, seedURL := range urls {
		processURL(seedURL, 0)
	}

	if err := g.Wait(); err != nil && err != context.Canceled {
		slog.Warn("forum crawl completed with error", "error", err)
	}

	slog.Info("forum crawl finished",
		"threads", len(visited),
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
