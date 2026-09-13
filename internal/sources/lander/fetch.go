package lander

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/breaker"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/diag"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/limit"
)

// LastProxyIndex returns the proxy pool index from the last HTTP request, or -1.
func (f *HTTPFetcher) LastProxyIndex() int {
	if f == nil || f.client == nil {
		return -1
	}
	return httpclient.LastProxyIndex(f.client)
}

type HTTPFetcher struct {
	client  *http.Client
	breaker *breaker.SourceBreaker
	limiter *limit.HostLimiters
	baseURL string
}

func NewHTTPFetcher(timeout time.Duration, baseURL string) *HTTPFetcher {
	return newHTTPFetcher(httpclient.Shared(timeout), baseURL)
}

// NewHTTPFetcherFromConfig builds an HTTP fetcher for the lander source family.
func NewHTTPFetcherFromConfig(cfg config.Config) (*HTTPFetcher, error) {
	return NewHTTPFetcherForSource(cfg, "lander")
}

// NewHTTPFetcherForSource builds an HTTP fetcher with per-source proxy routing.
func NewHTTPFetcherForSource(cfg config.Config, sourceID string) (*HTTPFetcher, error) {
	proxyURLs := cfg.ProxyURLsForSource(sourceID)
	if len(proxyURLs) > 0 {
		client, err := httpclient.NewClientWithProxies(cfg.HTTPTimeout, proxyURLs, sourceID)
		if err != nil {
			return nil, err
		}
		slog.Info("lander http fetcher using proxy rotation", "source", sourceID, "proxies", len(proxyURLs))
		return newHTTPFetcher(client, cfg.LanderBaseURL), nil
	}
	return NewHTTPFetcher(cfg.HTTPTimeout, cfg.LanderBaseURL), nil
}

func newHTTPFetcher(client *http.Client, baseURL string) *HTTPFetcher {
	return &HTTPFetcher{
		client:  client,
		breaker: breaker.NewSourceBreaker(),
		limiter: limit.NewHostLimiters(2, 4), // max 2 concurrent requests per host, 4 RPS
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

func (f *HTTPFetcher) Get(ctx context.Context, rawURL string) (string, error) {
	body, _, err := f.GetStatus(ctx, rawURL)
	return body, err
}

// GetStatus returns response body and status code. Transport errors return status 0.
func (f *HTTPFetcher) GetStatus(ctx context.Context, rawURL string) (body string, status int, err error) {
	body, status, _, err = f.getStatusMeta(ctx, rawURL, false, true)
	return body, status, err
}

// GetStatusMeta returns body, status, and whether the response looks like a Cloudflare block.
func (f *HTTPFetcher) GetStatusMeta(ctx context.Context, rawURL string, logNonOK bool) (body string, status int, cfBlocked bool, err error) {
	return f.getStatusMeta(ctx, rawURL, false, logNonOK)
}

// GetRSC fetches the App Router flight payload via RSC HTTP headers (no browser hydration).
func (f *HTTPFetcher) GetRSC(ctx context.Context, rawURL string) (string, error) {
	body, _, _, err := f.getStatusMeta(ctx, rawURL, true, true)
	return body, err
}

func (f *HTTPFetcher) getStatusMeta(ctx context.Context, rawURL string, rsc bool, logNonOK bool) (string, int, bool, error) {
	if f.breaker != nil && !f.breaker.Allow("lander") {
		return "", 0, false, fmt.Errorf("source circuit open")
	}

	host := hostFromURL(rawURL)
	if err := f.limiter.Wait(ctx, host); err != nil {
		return "", 0, false, err
	}

	url := resolveFetchURL(f.baseURL, rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, false, err
	}
	if rsc {
		// Next.js App Router flight request; response body is RSC wire, not HTML.
		req.Header.Set("RSC", "1")
		req.Header.Set("Next-Url", pagePath(rawURL))
		req.Header.Set("Next-Router-Prefetch", "1")
		req.Header.Set("Accept", "text/x-component")
	}

	resp, err := f.client.Do(req)
	if err != nil {
		slog.Debug("lander http transport error",
			"url", url,
			"rsc", rsc,
			"error", err,
		)
		return "", 0, false, err
	}

	if f.breaker != nil {
		f.breaker.RecordResponse("lander", resp)
	}

	rawBody, err := httpclient.ReadResponseBody(resp, 2<<20)
	if err != nil {
		slog.Debug("lander http read error",
			"url", url,
			"rsc", rsc,
			"status", resp.StatusCode,
			"error", err,
		)
		return "", resp.StatusCode, false, err
	}
	cfBlocked := httpclient.LooksCloudflareBlocked(resp.StatusCode, resp.Header, rawBody)
	bodyStr := string(rawBody)
	if resp.StatusCode != http.StatusOK {
		if logNonOK {
			slog.Warn("lander http non-ok",
				"url", url,
				"rsc", rsc,
				"status", resp.StatusCode,
				"body_bytes", len(rawBody),
				"body_preview", diag.Preview(bodyStr, 300),
			)
		}
		return bodyStr, resp.StatusCode, cfBlocked, fmt.Errorf("http %d", resp.StatusCode)
	}

	slog.Debug("lander http ok",
		"url", url,
		"rsc", rsc,
		"status", resp.StatusCode,
		"body_bytes", len(rawBody),
		"content_type", resp.Header.Get("Content-Type"),
	)
	return bodyStr, resp.StatusCode, cfBlocked, nil
}

func hostFromURL(raw string) string {
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	if idx := strings.Index(raw, "/"); idx >= 0 {
		return raw[:idx]
	}
	return raw
}
