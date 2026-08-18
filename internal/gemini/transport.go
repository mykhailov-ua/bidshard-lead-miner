package gemini

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type callKind int

const (
	callGenerate callKind = iota
	callEmbed
)

func (c *Client) postWithQuota(ctx context.Context, kind callKind, url string, body []byte, estTokens int) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("gemini client nil")
	}
	maxAttempts := c.limits.MaxRetries + 1
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := c.waitQuota(ctx, kind, estTokens); err != nil {
			return nil, err
		}

		respBody, status, retryAfter, err := c.postOnce(ctx, url, body)
		if err == nil && status >= 200 && status < 300 {
			return respBody, nil
		}

		apiErr := parseAPIError(status, respBody, retryAfter)
		if apiErr != nil {
			lastErr = apiErr
		} else if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("gemini http %d", status)
		}

		if attempt == maxAttempts-1 || !retryableError(lastErr) {
			break
		}
		if err := c.sleep(ctx, c.backoffDuration(attempt, retryAfter)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func retryableError(err error) bool {
	if err == nil {
		return false
	}
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.Retryable()
	}
	msg := err.Error()
	return strings.Contains(msg, "429") || strings.Contains(msg, "RESOURCE_EXHAUSTED") || strings.Contains(msg, "503")
}

func (c *Client) waitQuota(ctx context.Context, kind callKind, estTokens int) error {
	if c.limiter == nil {
		return nil
	}
	switch kind {
	case callEmbed:
		return c.limiter.WaitEmbed(ctx, estTokens)
	default:
		return c.limiter.WaitGenerate(ctx, estTokens)
	}
}

func (c *Client) postOnce(ctx context.Context, url string, body []byte) ([]byte, int, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, resp.StatusCode, 0, err
	}
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBody, resp.StatusCode, retryAfter, parseAPIError(resp.StatusCode, respBody, retryAfter)
	}
	return respBody, resp.StatusCode, retryAfter, nil
}
