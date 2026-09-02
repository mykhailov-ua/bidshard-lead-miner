package serp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/metrics"
)

const serpMaxAttempts = 6

func isRetryableSERPStatus(code int) bool {
	switch code {
	case http.StatusAccepted, http.StatusTooManyRequests,
		http.StatusForbidden, http.StatusServiceUnavailable, http.StatusBadGateway:
		return true
	default:
		return false
	}
}

func (c *Crawler) searchDork(ctx context.Context, dork string) ([]SERPResult, error) {
	var lastErr error
	for attempt := 0; attempt < serpMaxAttempts; attempt++ {
		body, status, err := c.fetchDorkHTML(ctx, dork, attempt)
		if err != nil {
			lastErr = err
			if attempt < serpMaxAttempts-1 {
				if waitErr := sleepSERPRetry(ctx, attempt); waitErr != nil {
					return nil, waitErr
				}
				continue
			}
			metrics.RecordSERPHarvestFailed()
			return nil, err
		}
		if status == http.StatusOK {
			return parseSERPResults(body), nil
		}
		lastErr = fmt.Errorf("http %d", status)
		if isRetryableSERPStatus(status) && attempt < serpMaxAttempts-1 {
			if waitErr := sleepSERPRetry(ctx, attempt); waitErr != nil {
				return nil, waitErr
			}
			continue
		}
		metrics.RecordSERPHarvestFailed()
		return nil, lastErr
	}
	metrics.RecordSERPHarvestFailed()
	return nil, fmt.Errorf("serp retry exhausted")
}

func sleepSERPRetry(ctx context.Context, attempt int) error {
	sec := 1 << (attempt + 1)
	if sec > 30 {
		sec = 30
	}
	d := time.Duration(sec) * time.Second
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Crawler) duckDuckGoHTMLURL() string {
	base := strings.TrimSpace(c.baseURL)
	if base == "" {
		base = "https://html.duckduckgo.com/html/"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	if idx := strings.Index(base, "?"); idx >= 0 {
		base = base[:idx]
	}
	return base
}

func (c *Crawler) fetchDorkHTML(ctx context.Context, dork string, attempt int) (string, int, error) {
	if attempt%2 == 1 {
		return c.fetchDorkGET(ctx, dork)
	}
	return c.fetchDorkPOST(ctx, dork)
}

func (c *Crawler) fetchDorkPOST(ctx context.Context, dork string) (string, int, error) {
	reqURL := c.duckDuckGoHTMLURL()
	body := "q=" + url.QueryEscape(dork)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	applySERPHeaders(req)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = int64(len(body))

	resp, err := c.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(raw), resp.StatusCode, nil
}

func (c *Crawler) fetchDorkGET(ctx context.Context, dork string) (string, int, error) {
	params := url.Values{}
	params.Set("q", dork)
	reqURL := c.duckDuckGoHTMLURL() + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", 0, err
	}
	applySERPHeaders(req)

	body, status, err := httpclient.DoBytes(c.client, req, 2<<20)
	if err != nil {
		return "", 0, err
	}
	return string(body), status, nil
}

func applySERPHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}
