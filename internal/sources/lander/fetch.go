package lander

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/breaker"
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
	return &HTTPFetcher{
		client:  httpclient.Shared(timeout),
		breaker: breaker.NewSourceBreaker(),
		limiter: limit.NewHostLimiters(2, 4),
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

func (f *HTTPFetcher) Get(ctx context.Context, rawURL string) (string, error) {
	if f.breaker != nil && !f.breaker.Allow("lander") {
		return "", fmt.Errorf("source circuit open")
	}

	host := hostFromURL(rawURL)
	if err := f.limiter.Wait(ctx, host); err != nil {
		return "", err
	}

	url := rawURL
	if f.baseURL != "" {
		if idx := strings.Index(rawURL, "/offer/"); idx >= 0 {
			url = f.baseURL + rawURL[idx:]
		} else if idx := strings.Index(rawURL, "/preview"); idx >= 0 {
			url = f.baseURL + rawURL[idx:]
		} else {
			url = f.baseURL + "/"
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if f.breaker != nil {
		f.breaker.RecordResponse("lander", resp)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
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
