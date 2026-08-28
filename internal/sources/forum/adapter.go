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
	"github.com/bidshard/parser/internal/proxybudget"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Adapter struct {
	seedPath        string
	warriorSeedPath string
	registryPath    string
	fetcher         *Fetcher
	workers         int
	usesProxy       bool
}

func NewAdapter(cfg config.Config, fetcher *Fetcher) *Adapter {
	if fetcher == nil {
		fetcher = NewFetcherWithConfig(cfg)
	}
	workers := cfg.HTTPWorkers
	if workers <= 0 {
		workers = 10
	}
	warriorPath := cfg.WarriorSeedPath
	if warriorPath == "" {
		warriorPath = "data/seeds/warrior_threads.csv"
	}
	return &Adapter{
		seedPath:        cfg.ForumSeedPath,
		warriorSeedPath: warriorPath,
		registryPath:    cfg.ForumRegistryPath,
		fetcher:         fetcher,
		workers:         workers,
		usesProxy:       len(cfg.ProxyURLsForSource("forum")) > 0,
	}
}

func (a *Adapter) Name() string {
	return "forum"
}

func (a *Adapter) Collect(ctx context.Context, emit EmitFunc) error {
	if skip, reason := proxybudget.ShouldSkipProxySource("forum", a.usesProxy); skip {
		slog.Info("forum crawl skipped", "reason", reason)
		return nil
	}
	seeds, err := LoadThreadSeedsCombined(a.seedPath, a.registryPath, a.warriorSeedPath)
	if err != nil {
		return err
	}

	start := time.Now()
	var (
		mu      sync.Mutex
		emitted int
		skipped int
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
				contacts := extract.Extract(post.Body)
				if contacts.Rejected {
					continue
				}
				primary := ""
				if len(contacts.Contacts) > 0 {
					primary = extract.FormatAll(contacts.Contacts)[0]
				} else if post.Author != "" && !strings.EqualFold(post.Author, "anonymous") {
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

	for _, seed := range seeds {
		if fetch, verdict := TriageThread(seed); !fetch {
			skipped++
			slog.Debug("forum thread triage skip", "url", seed.URL, "verdict", verdict)
			continue
		}
		processURL(seed.URL, 0)
	}

	if err := g.Wait(); err != nil && err != context.Canceled {
		slog.Warn("forum crawl completed with error", "error", err)
	}

	slog.Info("forum crawl finished",
		"threads", len(visited),
		"skipped_seeds", skipped,
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
