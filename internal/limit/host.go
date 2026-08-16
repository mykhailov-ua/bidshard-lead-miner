package limit

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type HostLimiters struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

func NewHostLimiters(rps float64, burst int) *HostLimiters {
	if rps <= 0 {
		rps = 2
	}
	if burst <= 0 {
		burst = 4
	}
	return &HostLimiters{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
	}
}

func (h *HostLimiters) Wait(ctx context.Context, host string) error {
	lim := h.forHost(host)
	return lim.Wait(ctx)
}

func (h *HostLimiters) forHost(host string) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()

	lim, ok := h.limiters[host]
	if !ok {
		lim = rate.NewLimiter(h.rate, h.burst)
		h.limiters[host] = lim
	}
	return lim
}
