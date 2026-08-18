package gemini

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func retryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code == http.StatusServiceUnavailable || code == http.StatusGatewayTimeout
}

func parseRetryAfter(header string) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

func (c *Client) backoffDuration(attempt int, retryAfter time.Duration) time.Duration {
	base := c.limits.RetryBase
	if base <= 0 {
		base = 2 * time.Second
	}
	max := c.limits.RetryMax
	if max <= 0 {
		max = 60 * time.Second
	}
	d := base * time.Duration(1<<attempt)
	if retryAfter > d {
		d = retryAfter
	}
	if d > max {
		return max
	}
	return d
}

func (c *Client) sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type APIError struct {
	Code       int
	Status     string
	Message    string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e == nil {
		return "gemini api error"
	}
	if e.Status != "" {
		return fmt.Sprintf("gemini api %s: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("gemini http %d: %s", e.Code, e.Message)
}

func (e *APIError) Retryable() bool {
	if e == nil {
		return false
	}
	if retryableStatus(e.Code) {
		return true
	}
	return strings.Contains(strings.ToUpper(e.Status), "RESOURCE_EXHAUSTED")
}

func parseAPIError(statusCode int, body []byte, retryAfter time.Duration) *APIError {
	var wrapper struct {
		Error *apiError `json:"error"`
	}
	_ = jsonUnmarshal(body, &wrapper)
	if wrapper.Error != nil {
		return &APIError{
			Code:       wrapper.Error.Code,
			Status:     wrapper.Error.Status,
			Message:    wrapper.Error.Message,
			RetryAfter: retryAfter,
		}
	}
	return &APIError{Code: statusCode, Message: truncate(string(body), 300), RetryAfter: retryAfter}
}
