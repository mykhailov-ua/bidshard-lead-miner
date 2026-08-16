package forum

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

type Fetcher struct {
	client   *http.Client
	limiters *limit.HostLimiters
	breaker  *breaker.SourceBreaker
	baseURL  string
}

func NewFetcher(timeout time.Duration, baseURL string) *Fetcher {
	return &Fetcher{
		client:   httpclient.Shared(timeout),
		limiters: limit.NewHostLimiters(1, 1),
		breaker:  breaker.NewSourceBreaker(),
		baseURL:  strings.TrimSuffix(baseURL, "/"),
	}
}

func (f *Fetcher) Get(ctx context.Context, rawURL string) (string, error) {
	if f.breaker != nil && !f.breaker.Allow("forum") {
		return "", fmt.Errorf("source circuit open")
	}

	host := hostFromURL(rawURL)
	if err := f.limiters.Wait(ctx, host); err != nil {
		return "", err
	}

	url := rawURL
	if f.baseURL != "" {
		if idx := strings.Index(rawURL, "/threads/"); idx >= 0 {
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
		f.breaker.RecordResponse("forum", resp)
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
