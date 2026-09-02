package lander

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bidshard/parser/internal/diag"
	"github.com/bidshard/parser/internal/extract"
)

// PageFetcher combines HTTP GET, optional RSC flight fetch, and headless fallback.
type PageFetcher struct {
	HTTP            *HTTPFetcher
	Headless        HeadlessFetcher
	HeadlessEnabled bool
	HeadlessDefer   bool
	QueuePath       string
	SourceFamily    string
}

// PageFetchOptions configures HTTP/RSC/headless/defer-queue behavior.
type PageFetchOptions struct {
	HeadlessEnabled bool
	HeadlessDefer   bool
	QueuePath       string
	SourceFamily    string
}

// PageFetchMeta describes how page HTML was obtained.
type PageFetchMeta struct {
	Stage      string
	RSCFetched bool
	RSCBytes   int
}

func NewPageFetcher(http *HTTPFetcher, headless HeadlessFetcher, opts PageFetchOptions) *PageFetcher {
	if headless == nil {
		headless = DisabledHeadless{}
	}
	return &PageFetcher{
		HTTP:            http,
		Headless:        headless,
		HeadlessEnabled: opts.HeadlessEnabled,
		HeadlessDefer:   opts.HeadlessDefer,
		QueuePath:       opts.QueuePath,
		SourceFamily:    opts.SourceFamily,
	}
}

// FetchHTML retrieves page HTML with RSC merge and optional headless fallback on HTTP failure
// or empty extractable text after RSC.
func (p *PageFetcher) FetchHTML(ctx context.Context, pageURL string) (string, PageFetchMeta, error) {
	meta := PageFetchMeta{Stage: "http_get"}

	html, err := p.HTTP.Get(ctx, pageURL)
	if err != nil {
		if headlessHTML, ok := p.tryHeadless(ctx, pageURL, &meta, "headless_fallback"); ok {
			return headlessHTML, meta, nil
		}
		if !strings.HasSuffix(meta.Stage, "_queued") {
			meta.Stage = "http_get_failed"
		}
		return "", meta, err
	}

	html = p.mergeRSC(ctx, pageURL, html, &meta)

	if headlessHTML, ok := p.headlessForEmptyText(ctx, pageURL, html, &meta); ok {
		return headlessHTML, meta, nil
	}

	return html, meta, nil
}

func (p *PageFetcher) mergeRSC(ctx context.Context, pageURL, html string, meta *PageFetchMeta) string {
	if !ShouldFetchRSC(html) {
		return html
	}
	meta.Stage = "http_get+rsc"
	rsc, rscErr := p.HTTP.GetRSC(ctx, pageURL)
	if rscErr != nil {
		slog.Warn("page rsc fetch failed",
			"url", pageURL,
			"error", rscErr,
			"html_bytes", len(html),
		)
		return html
	}
	if strings.TrimSpace(rsc) == "" {
		slog.Warn("page rsc empty",
			"url", pageURL,
			"html_bytes", len(html),
		)
		return html
	}
	meta.RSCFetched = true
	meta.RSCBytes = len(rsc)
	slog.Debug("page rsc fetch ok",
		"url", pageURL,
		"rsc_bytes", meta.RSCBytes,
		"rsc_preview", diag.Preview(rsc, diag.DefaultPreview),
	)
	// Inject flight wire as synthetic __next_f scripts so DiagnoseExtract can parse App Router payloads.
	return html + "\n<script>" + wrapRSCAsFlightScripts(rsc) + "</script>"
}

func (p *PageFetcher) headlessForEmptyText(ctx context.Context, pageURL, html string, meta *PageFetchMeta) (string, bool) {
	if !p.HeadlessEnabled && !p.HeadlessDefer {
		return "", false
	}
	// Skip headless when HTTP+RSC already yields extractable contacts.
	if pageHasExtractableContacts(html) {
		return "", false
	}
	return p.tryHeadless(ctx, pageURL, meta, "headless_empty")
}

func pageHasExtractableContacts(html string) bool {
	text, _ := TextForContactExtract(html)
	if strings.TrimSpace(text) == "" {
		return false
	}
	contacts := extract.Extract(text)
	contacts.Contacts = extract.FilterJunkContacts(contacts.Contacts)
	return !contacts.Rejected && len(contacts.Contacts) > 0
}

// FetchForCrawl performs HTTP GET with optional RSC merge. Non-OK responses return body + status
// so callers can detect SPA soft-404 shells without treating them as transport errors.
func (p *PageFetcher) FetchForCrawl(ctx context.Context, pageURL string, logHTTPNonOK bool) (html string, meta PageFetchMeta, status int, err error) {
	meta = PageFetchMeta{Stage: "http_get"}

	html, status, err = p.HTTP.getStatus(ctx, pageURL, false, logHTTPNonOK)
	if err != nil && status == 0 {
		if headlessHTML, ok := p.tryHeadless(ctx, pageURL, &meta, "headless_fallback"); ok {
			return headlessHTML, meta, http.StatusOK, nil
		}
		if !strings.HasSuffix(meta.Stage, "_queued") {
			meta.Stage = "http_get_failed"
		}
		return "", meta, 0, err
	}

	if status == http.StatusOK {
		html = p.mergeRSC(ctx, pageURL, html, &meta)
		if headlessHTML, ok := p.headlessForEmptyText(ctx, pageURL, html, &meta); ok {
			return headlessHTML, meta, http.StatusOK, nil
		}
		return html, meta, status, nil
	}

	// Non-OK with body: return status+html for SPA404 fingerprinting; err mirrors HTTP status.
	return html, meta, status, err
}

func (p *PageFetcher) tryHeadless(ctx context.Context, pageURL string, meta *PageFetchMeta, stage string) (string, bool) {
	if p.HeadlessDefer {
		if err := EnqueueHeadless(p.QueuePath, HeadlessQueueItem{
			URL:          pageURL,
			SourceFamily: p.SourceFamily,
			Reason:       stage,
		}); err != nil {
			slog.Warn("headless defer enqueue failed", "url", pageURL, "error", err)
		} else {
			slog.Debug("headless deferred to queue", "url", pageURL, "reason", stage)
			meta.Stage = stage + "_queued"
		}
		return "", false
	}
	if !p.HeadlessEnabled {
		return "", false
	}
	headlessHTML, err := p.Headless.Fetch(ctx, pageURL)
	if err != nil {
		slog.Warn("page headless fetch failed",
			"url", pageURL,
			"stage", stage,
			"error", err,
		)
		meta.Stage = stage + "_failed"
		return "", false
	}
	if strings.TrimSpace(headlessHTML) == "" {
		meta.Stage = stage + "_empty"
		return "", false
	}
	meta.Stage = stage
	slog.Debug("page headless fetch ok",
		"url", pageURL,
		"stage", stage,
		"html_bytes", len(headlessHTML),
	)
	return headlessHTML, true
}
