package forum

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/breaker"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/limit"
)

type Fetcher struct {
	client   *http.Client
	limiters *limit.HostLimiters
	breaker  *breaker.SourceBreaker
	baseURL  string
}

func NewFetcher(timeout time.Duration, baseURL string) *Fetcher {
	return &Fetcher{
		client:   httpclient.Shared(timeout),
		limiters: limit.NewHostLimiters(0.5, 1),
		breaker:  breaker.NewSourceBreaker(),
		baseURL:  strings.TrimSuffix(baseURL, "/"),
	}
}

func NewFetcherWithConfig(cfg config.Config) *Fetcher {
	return &Fetcher{
		client:   httpclient.CrawlClient(cfg.HTTPTimeout, cfg.ProxyURLsForSource("forum"), "forum"),
		limiters: limit.NewHostLimiters(0.5, 1),
		breaker:  breaker.NewSourceBreaker(),
		baseURL:  strings.TrimSuffix(cfg.ForumBaseURL, "/"),
	}
}

func (f *Fetcher) Get(ctx context.Context, rawURL string) (string, error) {
	if isFixtureURL(rawURL) {
		return loadFixtureHTML(rawURL)
	}

	if f.breaker != nil && !f.breaker.Allow("forum") {
		return "", fmt.Errorf("source circuit open")
	}

	host := hostFromURL(rawURL)
	if err := f.limiters.Wait(ctx, host); err != nil {
		return "", err
	}

	fetchURL := rawURL
	if f.baseURL != "" {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", err
		}
		fetchURL = f.baseURL + parsed.Path
		if parsed.RawQuery != "" {
			fetchURL += "?" + parsed.RawQuery
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}

	// RecordResponse before body read; DoBytes would not allow this ordering without a hook.
	if f.breaker != nil {
		f.breaker.RecordResponse("forum", resp)
	}
	body, err := httpclient.ReadResponseBody(resp, 2<<20)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	return string(body), nil
}

func hostFromURL(raw string) string {
	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	if idx := strings.Index(raw, "/"); idx >= 0 {
		return raw[:idx]
	}
	return raw
}
