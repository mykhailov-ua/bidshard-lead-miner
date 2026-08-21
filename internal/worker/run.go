package worker

import (
	"context"
	"sync"
	"time"
)

func Run(ctx context.Context, wg *sync.WaitGroup, fn func(context.Context)) {
	if fn == nil {
		return
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		fn(ctx)
	}()
}

func DurationOr(v, def time.Duration) time.Duration {
	if v <= 0 {
		return def
	}
	return v
}

func IntOr(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func FloatOr(v, def float64) float64 {
	if v <= 0 {
		return def
	}
	return v
}
