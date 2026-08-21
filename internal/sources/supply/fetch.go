package supply

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/breaker"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/limit"
)

type Fetcher struct {
	Client   *http.Client
	Limiters *limit.HostLimiters
	Breaker  *breaker.SourceBreaker
	BaseURL  string
	Source   string
}

func NewFetcher(timeout time.Duration, hostRPS float64, baseURL string) *Fetcher {
	return &Fetcher{
		Client:   httpclient.Shared(timeout),
		Limiters: limit.NewHostLimiters(hostRPS, 4),
		Breaker:  breaker.NewSourceBreaker(),
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		Source:   "supply",
	}
}

func (f *Fetcher) Get(ctx context.Context, domain, path string) ([]byte, int, error) {
	if f.Breaker != nil && !f.Breaker.Allow(f.Source) {
		return nil, 0, fmt.Errorf("source circuit open")
	}

	host := domain
	if err := f.Limiters.Wait(ctx, host); err != nil {
		return nil, 0, err
	}

	url := f.url(domain, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := f.Client.Do(req)
	if err != nil {
		if f.Breaker != nil {
			f.Breaker.RecordTransportError(f.Source)
		}
		return nil, 0, err
	}

	if f.Breaker != nil {
		f.Breaker.RecordResponse(f.Source, resp)
	}

	body, err := httpclient.ReadResponseBody(resp, 2<<20)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != http.StatusOK {
		return body, resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
	}
	return body, resp.StatusCode, nil
}

func (f *Fetcher) url(domain, path string) string {
	if f.BaseURL != "" {
		return f.BaseURL + path
	}
	return "https://" + domain + path
}
