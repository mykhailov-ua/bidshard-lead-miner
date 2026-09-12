package jobboard

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
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/proxybudget"
	"github.com/bidshard/parser/internal/sources/forum"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Adapter struct {
	registryPath         string
	employerRegistryPath string
	fetcher              *forum.Fetcher
	workers              int
	usesProxy            bool
}

func NewAdapter(cfg config.Config, fetcher *forum.Fetcher) *Adapter {
	if fetcher == nil {
		fetcher = forum.NewFetcherForSource(cfg, "jobboard")
	}
	workers := cfg.HTTPWorkers
	if workers <= 0 {
		workers = 10
	}
	return &Adapter{
		registryPath:         cfg.JobboardRegistryPath,
		employerRegistryPath: cfg.EmployerRegistryPath,
		fetcher:              fetcher,
		workers:              workers,
		usesProxy:            len(cfg.ProxyURLsForSource("jobboard")) > 0,
	}
}

func (a *Adapter) registryPathEmployers() string {
	if strings.TrimSpace(a.employerRegistryPath) != "" {
		return a.employerRegistryPath
	}
	return DefaultEmployerRegistryPath
}

func (a *Adapter) Name() string {
	return "jobboard"
}

func (a *Adapter) Collect(ctx context.Context, emit EmitFunc) error {
	if skip, reason := proxybudget.ShouldSkipProxySource("jobboard", a.usesProxy); skip {
		slog.Info("jobboard crawl skipped", "reason", reason)
		return nil
	}
	reg, err := LoadRegistry(a.registryPath)
	if err != nil {
		return err
	}
	if len(reg.URLs) == 0 {
		slog.Debug("jobboard registry empty", "path", a.registryPath)
		return nil
	}

	start := time.Now()
	var (
		mu      sync.Mutex
		emitted int
		skipped int
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(a.workers)

	for _, entry := range reg.URLs {
		entry := entry
		if !ShouldCrawl(entry.URL, entry.Title, entry.Snippet) {
			skipped++
			continue
		}
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			html, err := a.fetcher.Get(gCtx, entry.URL)
			if err != nil {
				slog.Warn("jobboard fetch failed", "url", entry.URL, "error", err)
				return nil
			}

			page, ok := ParseHTML(entry.URL, html)
			if !ok {
				return nil
			}
			combined := strings.TrimSpace(entry.Title + " " + entry.Snippet + " " + page.Title + " " + page.Company + " " + page.Body)
			if !ShouldCrawl(entry.URL, page.Title, combined) {
				return nil
			}

			contacts := extract.Extract(combined)
			contacts.Contacts = extract.FilterJunkContacts(contacts.Contacts)
			primary := pickContact(page, contacts.Contacts)
			if primary == "" {
				return nil
			}

			item := model.RawItem{
				Source:   sourceName(entry.URL, page),
				Raw:      combined,
				Contact:  primary,
				Title:    firstNonEmpty(page.Title, entry.Title),
				Username: page.Company,
			}
			if _, err := UpsertEmployer(a.registryPathEmployers(), EmployerFromPage(page, entry.URL, entry.Source, entry.Query)); err != nil {
				slog.Warn("employer registry upsert failed", "company", page.Company, "error", err)
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
		slog.Warn("jobboard crawl completed with error", "error", err)
	}

	slog.Info("jobboard crawl finished",
		"urls", len(reg.URLs),
		"skipped_seeds", skipped,
		"emitted", emitted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func pickContact(page Page, contacts []extract.Contact) string {
	for _, c := range contacts {
		if c.Type == "email" || c.Type == "telegram" {
			return extract.FormatAll([]extract.Contact{c})[0]
		}
	}
	if page.Slug != "" {
		return "jobboard:company/" + page.Slug
	}
	if page.Company != "" {
		slug := strings.ToLower(strings.ReplaceAll(page.Company, " ", "-"))
		return "jobboard:company/" + slug
	}
	return ""
}

func sourceName(rawURL string, page Page) string {
	host := HostFromURL(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil || host == "" {
		return "jobboard:unknown"
	}
	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "jobboard:" + host
	}
	if page.Kind == KindProfile {
		parts := strings.Split(path, "/")
		if len(parts) >= 2 && parts[0] == "q" {
			return "jobboard:djinni/profile/" + parts[1]
		}
	}
	if page.Slug != "" {
		return "jobboard:" + host + "/" + page.Slug
	}
	return "jobboard:" + host + "/" + path
}
