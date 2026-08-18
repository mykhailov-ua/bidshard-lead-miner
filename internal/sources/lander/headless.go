package lander

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// HeadlessFetcher retrieves dynamically rendered HTML for Next.js App Router / RSC pages.
type HeadlessFetcher interface {
	Fetch(ctx context.Context, url string) (string, error)
}

type DisabledHeadless struct{}

func (DisabledHeadless) Fetch(ctx context.Context, url string) (string, error) {
	_ = ctx
	_ = url
	return "", fmt.Errorf("headless disabled")
}

// PlaywrightPoolFetcher implements a bounded headless fetcher pool.
// Note: Maximum 2 concurrent browser contexts allowed (memory cap <= 512MB per browser).
type PlaywrightPoolFetcher struct {
	mu           sync.Mutex
	maxBrowsers  int
	active       int
	fetchTimeout time.Duration
	mockRunner   func(ctx context.Context, url string) (string, error)
}

func NewPlaywrightPoolFetcher(maxBrowsers int, timeout time.Duration) *PlaywrightPoolFetcher {
	if maxBrowsers <= 0 {
		maxBrowsers = 2
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &PlaywrightPoolFetcher{
		maxBrowsers:  maxBrowsers,
		fetchTimeout: timeout,
	}
}

func (p *PlaywrightPoolFetcher) SetMockRunner(fn func(ctx context.Context, url string) (string, error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mockRunner = fn
}

func (p *PlaywrightPoolFetcher) Fetch(ctx context.Context, pageURL string) (string, error) {
	p.mu.Lock()
	if p.active >= p.maxBrowsers {
		p.mu.Unlock()
		return "", fmt.Errorf("headless browser pool saturated (%d/%d)", p.active, p.maxBrowsers)
	}
	p.active++
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(ctx, p.fetchTimeout)
	defer cancel()

	p.mu.Lock()
	runner := p.mockRunner
	p.mu.Unlock()

	if runner != nil {
		return runner(ctx, pageURL)
	}

	// Default fallback when external browser process is not connected
	return "", fmt.Errorf("headless browser process unavailable")
}
