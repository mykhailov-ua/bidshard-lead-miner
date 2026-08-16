package breaker

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	defaultFailThreshold = 5
	defaultOpenTimeout   = 60 * time.Second
)

type SourceBreaker struct {
	mu           sync.Mutex
	failures     map[string]int
	openUntil    map[string]time.Time
	failThreshold int
	openTimeout   time.Duration
}

func NewSourceBreaker() *SourceBreaker {
	return &SourceBreaker{
		failures:      make(map[string]int),
		openUntil:     make(map[string]time.Time),
		failThreshold: defaultFailThreshold,
		openTimeout:   defaultOpenTimeout,
	}
}

func (b *SourceBreaker) Allow(source string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	until, ok := b.openUntil[source]
	if !ok {
		return true
	}
	if time.Now().After(until) {
		delete(b.openUntil, source)
		delete(b.failures, source)
		return true
	}
	return false
}

func (b *SourceBreaker) WaitDuration(source string) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	until, ok := b.openUntil[source]
	if !ok {
		return 0
	}
	remaining := time.Until(until)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (b *SourceBreaker) RecordSuccess(source string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.failures, source)
	delete(b.openUntil, source)
}

func (b *SourceBreaker) RecordFailure(source string, statusCode int, retryAfter time.Duration) {
	if statusCode != http.StatusTooManyRequests && statusCode < 500 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.failures[source]++
	if b.failures[source] < b.failThreshold {
		return
	}

	openFor := b.openTimeout
	if retryAfter > openFor {
		openFor = retryAfter
	}
	b.openUntil[source] = time.Now().Add(openFor)
	slog.Warn("source circuit open", "source", source, "retry_after", openFor)
}

func ParseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(header); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func (b *SourceBreaker) RecordTransportError(source string) {
	b.RecordFailure(source, http.StatusServiceUnavailable, 0)
}

func (b *SourceBreaker) RecordResponse(source string, resp *http.Response) {
	if resp == nil {
		return
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		b.RecordSuccess(source)
		return
	}
	retryAfter := ParseRetryAfter(resp.Header.Get("Retry-After"))
	b.RecordFailure(source, resp.StatusCode, retryAfter)
}
