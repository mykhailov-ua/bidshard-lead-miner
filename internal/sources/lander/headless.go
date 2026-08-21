package lander

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	python := strings.TrimSpace(os.Getenv("PARSER_TELETHON_PYTHON"))
	if python == "" {
		python = "python3"
	}
	// One subprocess per fetch (Playwright). BPF may show thread_fork noise; not an FD leak if processes exit.
	cmd := exec.CommandContext(ctx, python, "-m", "sources.headless.fetch", pageURL)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+headlessRepoRoot())
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("headless fetch: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("headless fetch: %w", err)
	}
	html := strings.TrimSpace(string(out))
	if html == "" {
		return "", fmt.Errorf("headless fetch returned empty HTML")
	}
	return html, nil
}

func headlessRepoRoot() string {
	if v := strings.TrimSpace(os.Getenv("PYTHONPATH")); v != "" {
		return strings.Split(v, string(os.PathListSeparator))[0]
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
