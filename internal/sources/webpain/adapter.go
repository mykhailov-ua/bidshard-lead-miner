package webpain

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/filter"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/proxybudget"
	"github.com/bidshard/parser/internal/sources/forum"
	"github.com/bidshard/parser/internal/sources/lander"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Adapter struct {
	registryPath string
	fetcher      *forum.Fetcher
	workers      int
	usesProxy    bool
}

func NewAdapter(cfg config.Config, fetcher *forum.Fetcher) *Adapter {
	if fetcher == nil {
		fetcher = forum.NewFetcherForSource(cfg, "webpain")
	}
	workers := cfg.HTTPWorkers
	if workers <= 0 {
		workers = 10
	}
	return &Adapter{
		registryPath: cfg.WebPainRegistryPath,
		fetcher:      fetcher,
		workers:      workers,
		usesProxy:    len(cfg.ProxyURLsForSource("webpain")) > 0,
	}
}

func (a *Adapter) Name() string {
	return "webpain"
}

func (a *Adapter) Collect(ctx context.Context, emit EmitFunc) error {
	if skip, reason := proxybudget.ShouldSkipProxySource("webpain", a.usesProxy); skip {
		slog.Info("webpain crawl skipped", "reason", reason)
		return nil
	}
	reg, err := LoadRegistry(a.registryPath)
	if err != nil {
		return err
	}
	if len(reg.URLs) == 0 {
		slog.Debug("webpain registry empty", "path", a.registryPath)
		return nil
	}

	start := time.Now()
	var (
		mu      sync.Mutex
		emitted int
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(a.workers)

	for _, entry := range reg.URLs {
		entry := entry
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			html, err := a.fetcher.Get(gCtx, entry.URL)
			if err != nil {
				slog.Warn("webpain fetch failed", "url", entry.URL, "error", err)
				return nil
			}

			pageText, _ := lander.TextForContactExtract(html)
			combined := strings.TrimSpace(entry.Snippet + " " + pageText)
			if combined == "" {
				return nil
			}
			if reject, _ := filter.RejectHTMLBoilerplate(combined); reject {
				return nil
			}

			siteDomain := HostFromURL(entry.URL)
			pageContacts := extract.Extract(combined)
			pageContacts.Contacts = extract.FilterJunkContacts(pageContacts.Contacts)
			if pageContacts.Rejected {
				return nil
			}

			primary, ok := PickPageLPR(pageContacts, siteDomain)
			if !ok {
				return nil
			}

			item := model.RawItem{
				Source:    sourceName(entry.URL),
				Raw:       combined,
				Contact:   formatPrimaryContact(primary),
				Title:     entry.Title,
				CrawlHTML: model.LimitCrawlHTML(html),
			}

			mu.Lock()
			defer mu.Unlock()
			if err := emit(gCtx, item); err != nil {
				return err
			}
			emitted++
			return nil
		})
	}

	if err := g.Wait(); err != nil && err != context.Canceled {
		slog.Warn("webpain crawl completed with error", "error", err)
	}

	slog.Info("webpain crawl finished",
		"urls", len(reg.URLs),
		"emitted", emitted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func sourceName(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "webpain:unknown"
	}
	host := strings.ToLower(strings.TrimPrefix(u.Host, "www."))
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "webpain:" + host
	}
	return "webpain:" + host + "/" + path
}
