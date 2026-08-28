package httpclient

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/bidshard/parser/internal/metrics"
)

// ErrProxyCooldown is returned when every proxy is cooling down and the request context ends.
var ErrProxyCooldown = errors.New("all proxies in cooldown")

// ProxyPoolConfig tunes per-proxy rate limits and block cooldowns.
type ProxyPoolConfig struct {
	PerProxyRPS      float64
	PerProxyBurst    int
	CooldownDuration time.Duration
}

// DefaultProxyPoolConfig is the process-wide default when env leaves limits at zero.
func DefaultProxyPoolConfig() ProxyPoolConfig {
	return ProxyPoolConfig{
		PerProxyRPS:      0.5,
		PerProxyBurst:    1,
		CooldownDuration: 10 * time.Minute,
	}
}

func (c ProxyPoolConfig) normalized() ProxyPoolConfig {
	out := DefaultProxyPoolConfig()
	if c.PerProxyRPS > 0 {
		out.PerProxyRPS = c.PerProxyRPS
	}
	if c.PerProxyBurst > 0 {
		out.PerProxyBurst = c.PerProxyBurst
	}
	if c.CooldownDuration > 0 {
		out.CooldownDuration = c.CooldownDuration
	}
	return out
}

var (
	poolCfgMu     sync.RWMutex
	globalPoolCfg = DefaultProxyPoolConfig()
)

// SetProxyPoolConfig sets defaults for NewClientWithProxies / CrawlClient (call once at startup).
func SetProxyPoolConfig(c ProxyPoolConfig) {
	poolCfgMu.Lock()
	globalPoolCfg = c.normalized()
	poolCfgMu.Unlock()
}

func activePoolConfig(override ...ProxyPoolConfig) ProxyPoolConfig {
	if len(override) > 0 {
		return override[0].normalized()
	}
	poolCfgMu.RLock()
	defer poolCfgMu.RUnlock()
	return globalPoolCfg
}

type proxyEndpoint struct {
	url      *url.URL
	key      string
	limiter  *rate.Limiter
	cooldown time.Time
}

type proxyPool struct {
	endpoints []*proxyEndpoint
	counter   uint64
	cfg       ProxyPoolConfig
	sourceID  string
	mu        sync.Mutex
}

func newProxyPool(parsed []*url.URL, cfg ProxyPoolConfig, sourceID string) *proxyPool {
	cfg = cfg.normalized()
	eps := make([]*proxyEndpoint, 0, len(parsed))
	for _, u := range parsed {
		eps = append(eps, &proxyEndpoint{
			url:     u,
			key:     u.String(),
			limiter: rate.NewLimiter(rate.Limit(cfg.PerProxyRPS), cfg.PerProxyBurst),
		})
	}
	return &proxyPool{
		endpoints: eps,
		cfg:       cfg,
		sourceID:  strings.TrimSpace(sourceID),
	}
}

func (p *proxyPool) acquire(ctx context.Context) (*url.URL, error) {
	if len(p.endpoints) == 0 {
		return nil, errors.New("no proxies configured")
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ep, waitUntil := p.pickEndpoint()
		if ep != nil {
			if err := ep.limiter.Wait(ctx); err != nil {
				return nil, err
			}
			return ep.url, nil
		}
		wait := time.Until(waitUntil)
		if wait <= 0 {
			p.pruneCooldowns(time.Now())
			continue
		}
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		slog.Info("proxy pool waiting for cooldown",
			"source", p.sourceID,
			"proxies", len(p.endpoints),
			"wait", wait.Round(time.Second),
			"next_available", waitUntil.UTC().Format(time.RFC3339),
		)
		metrics.RecordProxyCooldownWait(p.sourceID)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, ErrProxyCooldown
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (p *proxyPool) pickEndpoint() (*proxyEndpoint, time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	var available []*proxyEndpoint
	var earliest time.Time
	for _, ep := range p.endpoints {
		if ep.cooldown.IsZero() || now.After(ep.cooldown) {
			available = append(available, ep)
			continue
		}
		if earliest.IsZero() || ep.cooldown.Before(earliest) {
			earliest = ep.cooldown
		}
	}
	if len(available) == 0 {
		return nil, earliest
	}
	idx := atomic.AddUint64(&p.counter, 1) % uint64(len(available))
	return available[idx], time.Time{}
}

func (p *proxyPool) pruneCooldowns(now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, ep := range p.endpoints {
		if !ep.cooldown.IsZero() && now.After(ep.cooldown) {
			ep.cooldown = time.Time{}
		}
	}
}

func (p *proxyPool) markCooldown(proxy *url.URL, reason string) {
	if proxy == nil {
		return
	}
	key := proxy.String()
	until := time.Now().Add(p.cfg.CooldownDuration)
	p.mu.Lock()
	for _, ep := range p.endpoints {
		if ep.key == key {
			ep.cooldown = until
			break
		}
	}
	p.mu.Unlock()
	metrics.RecordProxyCFBlock(p.sourceID, reason)
	slog.Warn("proxy in cooldown",
		"source", p.sourceID,
		"proxy", proxy.Redacted(),
		"cooldown", p.cfg.CooldownDuration,
		"reason", reason,
	)
}

func isProxyBlock(resp *http.Response, bodyPrefix []byte) bool {
	if resp == nil {
		return false
	}
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return true
	case http.StatusForbidden, http.StatusServiceUnavailable:
		if resp.Header.Get("CF-Ray") != "" {
			return true
		}
		if strings.Contains(strings.ToLower(resp.Header.Get("Server")), "cloudflare") {
			return true
		}
		lower := strings.ToLower(string(bodyPrefix))
		for _, marker := range []string{
			"cf-browser-verification",
			"just a moment",
			"attention required",
			"cloudflare",
			"access denied",
		} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}
	return false
}

func peekBodyPrefix(resp *http.Response, limit int64) []byte {
	if resp == nil || resp.Body == nil {
		return nil
	}
	limited := io.LimitReader(resp.Body, limit)
	prefix, _ := io.ReadAll(limited)
	rest, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(prefix), bytes.NewReader(rest)))
	return prefix
}
