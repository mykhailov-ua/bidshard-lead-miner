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

type HTTPFetcher struct {
	client  *http.Client
	breaker *breaker.SourceBreaker
	limiter *limit.HostLimiters
	baseURL string
}

func NewHTTPFetcher(timeout time.Duration, baseURL string) *HTTPFetcher {
	return newHTTPFetcher(httpclient.Shared(timeout), baseURL)
}

// NewHTTPFetcherFromConfig builds an HTTP fetcher; uses PARSER_PROXY_LIST when set, else Shared().
func NewHTTPFetcherFromConfig(cfg config.Config) (*HTTPFetcher, error) {
	if len(cfg.ProxyURLs) > 0 {
		client, err := httpclient.NewClientWithProxies(cfg.HTTPTimeout, cfg.ProxyURLs)
		if err != nil {
			return nil, err
		}
		slog.Info("lander http fetcher using proxy rotation", "proxies", len(cfg.ProxyURLs))
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
	return f.getStatus(ctx, rawURL, false, true)
}

// GetRSC fetches the App Router flight payload via RSC HTTP headers (no browser hydration).
func (f *HTTPFetcher) GetRSC(ctx context.Context, rawURL string) (string, error) {
	body, _, err := f.getStatus(ctx, rawURL, true, true)
	return body, err
}

func (f *HTTPFetcher) getStatus(ctx context.Context, rawURL string, rsc bool, logNonOK bool) (string, int, error) {
	if f.breaker != nil && !f.breaker.Allow("lander") {
		return "", 0, fmt.Errorf("source circuit open")
	}

	host := hostFromURL(rawURL)
	if err := f.limiter.Wait(ctx, host); err != nil {
		return "", 0, err
	}

	url := resolveFetchURL(f.baseURL, rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, err
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
		return "", 0, err
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
		return "", resp.StatusCode, err
	}
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
		return bodyStr, resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
	}

	slog.Debug("lander http ok",
		"url", url,
		"rsc", rsc,
		"status", resp.StatusCode,
		"body_bytes", len(rawBody),
		"content_type", resp.Header.Get("Content-Type"),
	)
	return bodyStr, resp.StatusCode, nil
}

func hostFromURL(raw string) string {
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	if idx := strings.Index(raw, "/"); idx >= 0 {
		return raw[:idx]
	}
	return raw
}
