package lander

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/diag"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/filter"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/proxybudget"
	"github.com/bidshard/parser/internal/validate"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	seedPath        string
	registryPath    string
	http            *HTTPFetcher
	headless        HeadlessFetcher
	headlessEnabled bool
	fetchOpts       PageFetchOptions
	usesProxy       bool
}

func NewCrawler(cfg config.Config, httpFetcher *HTTPFetcher, headless HeadlessFetcher) *Crawler {
	if httpFetcher == nil {
		var err error
		httpFetcher, err = NewHTTPFetcherFromConfig(cfg)
		if err != nil {
			httpFetcher = NewHTTPFetcher(cfg.HTTPTimeout, cfg.LanderBaseURL)
		}
	}
	if headless == nil {
		headless = DisabledHeadless{}
	}
	return &Crawler{
		seedPath:        cfg.LanderSeedPath,
		registryPath:    cfg.SourceRegistryPath,
		http:            httpFetcher,
		headless:        headless,
		headlessEnabled: cfg.LanderHeadless,
		fetchOpts:       PageFetchOptsFromConfig(cfg, "lander"),
		usesProxy:       len(cfg.ProxyURLsForSource("lander")) > 0,
	}
}

func (c *Crawler) Name() string {
	return "lander"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	if skip, reason := proxybudget.ShouldSkipProxySource("lander", c.usesProxy); skip {
		slog.Info("lander crawl skipped", "reason", reason)
		return nil
	}
	urls, err := LoadURLsCombined(c.seedPath, c.registryPath)
	if err != nil {
		slog.Error("lander seed load failed", "path", c.seedPath, "error", err)
		return err
	}

	start := time.Now()
	emitted := 0

	slog.Info("lander crawl started", "seed_path", c.seedPath, "pages", len(urls), "headless", c.headlessEnabled)

	for _, pageURL := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		host := hostFromURL(pageURL)
		if validate.IsBlacklistedDomain(host) {
			slog.Debug("lander skip blacklisted domain", "host", host, "url", pageURL)
			continue
		}

		html, fetchMeta, err := c.fetchHTML(ctx, pageURL)
		if err != nil {
			slog.Warn("lander fetch failed",
				"url", pageURL,
				"stage", fetchMeta.stage,
				"error", err,
			)
			continue
		}

		slog.Debug("lander fetch ok",
			"url", pageURL,
			"stage", fetchMeta.stage,
			"html_bytes", len(html),
			"rsc_fetched", fetchMeta.rscFetched,
			"rsc_bytes", fetchMeta.rscBytes,
			"has_next_data", strings.Contains(html, "__NEXT_DATA__"),
			"has_next_f", strings.Contains(html, "__next_f"),
			"html_preview", diag.PreviewHTML(html, diag.HTMLPreview),
		)

		text, method := TextForContactExtract(html)
		if text == "" {
			slog.Warn("lander extract empty",
				"url", pageURL,
				"method", method,
				"html_bytes", len(html),
				"html_preview", diag.PreviewHTML(html, 300),
			)
			continue
		}

		slog.Debug("lander extract ok",
			"url", pageURL,
			"method", method,
			"text_bytes", len(text),
			"text_preview", diag.Preview(text, diag.DefaultPreview),
		)

		contacts := extract.Extract(text)
		contacts.Contacts = extract.FilterJunkContacts(contacts.Contacts)
		if contacts.Rejected {
			slog.Warn("lander contacts rejected",
				"url", pageURL,
				"reason", contacts.Reason,
				"text_preview", diag.Preview(text, 300),
			)
			continue
		}
		if len(contacts.Contacts) == 0 {
			slog.Warn("lander no contacts",
				"url", pageURL,
				"method", method,
				"text_preview", diag.Preview(text, 300),
			)
			continue
		}
		if !filter.LanderRequiresEmailOrSkype(contacts.Contacts) {
			slog.Warn("lander no email or skype contact",
				"url", pageURL,
				"method", method,
				"text_preview", diag.Preview(text, 300),
			)
			continue
		}

		formatted := extract.FormatAll(contacts.Contacts)
		slog.Debug("lander contacts found",
			"url", pageURL,
			"contact_count", len(formatted),
			"contact_preview", diag.MaskContact(formatted[0]),
		)

		item := model.RawItem{
			Source:    "lander:" + hostFromURL(pageURL),
			Raw:       text,
			Contact:   formatted[0],
			CrawlHTML: model.LimitCrawlHTML(html),
		}
		if err := emit(ctx, item); err != nil {
			return err
		}
		emitted++
	}

	slog.Info("lander crawl finished",
		"pages", len(urls),
		"emitted", emitted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

type fetchMeta struct {
	stage      string
	rscFetched bool
	rscBytes   int
}

func (c *Crawler) fetchHTML(ctx context.Context, pageURL string) (string, fetchMeta, error) {
	pf := NewPageFetcher(c.http, c.headless, c.fetchOpts)
	html, meta, err := pf.FetchHTML(ctx, pageURL)
	return html, fetchMeta{
		stage:      meta.Stage,
		rscFetched: meta.RSCFetched,
		rscBytes:   meta.RSCBytes,
	}, err
}
