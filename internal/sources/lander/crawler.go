package lander

import (
	"context"
	"log/slog"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/model"
)

type EmitFunc func(ctx context.Context, item model.RawItem) error

type Crawler struct {
	seedPath        string
	http            *HTTPFetcher
	headless        HeadlessFetcher
	headlessEnabled bool
}

func NewCrawler(cfg config.Config, httpFetcher *HTTPFetcher, headless HeadlessFetcher) *Crawler {
	if httpFetcher == nil {
		httpFetcher = NewHTTPFetcher(cfg.HTTPTimeout, cfg.LanderBaseURL)
	}
	if headless == nil {
		headless = DisabledHeadless{}
	}
	return &Crawler{
		seedPath:        cfg.LanderSeedPath,
		http:            httpFetcher,
		headless:        headless,
		headlessEnabled: cfg.LanderHeadless,
	}
}

func (c *Crawler) Name() string {
	return "lander"
}

func (c *Crawler) Collect(ctx context.Context, emit EmitFunc) error {
	urls, err := LoadURLs(c.seedPath)
	if err != nil {
		return err
	}

	start := time.Now()
	emitted := 0

	for _, pageURL := range urls {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		html, err := c.fetchHTML(ctx, pageURL)
		if err != nil {
			slog.Debug("lander fetch failed", "url", pageURL, "error", err)
			continue
		}

		text, err := ExtractPageText(html)
		if err != nil || text == "" {
			continue
		}

		contacts := extract.Extract(text)
		if contacts.Rejected || len(contacts.Contacts) == 0 {
			continue
		}

		item := model.RawItem{
			Source:    "lander:" + hostFromURL(pageURL),
			Raw:       text,
			Contact:   extract.FormatAll(contacts.Contacts)[0],
			CrawlHTML: html,
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

func (c *Crawler) fetchHTML(ctx context.Context, pageURL string) (string, error) {
	html, err := c.http.Get(ctx, pageURL)
	if err == nil {
		return html, nil
	}
	if !c.headlessEnabled {
		return "", err
	}
	return c.headless.Fetch(ctx, pageURL)
}
