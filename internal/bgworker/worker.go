package bgworker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job is a periodic background task (discovery, catalog harvest, etc.).
type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
	// SkipIfRunning drops a tick when the prior Run is still active (avoids overlapping pools).
	SkipIfRunning bool
	// InitialDelay before the first Run; zero uses 10s so the main scan loop can start.
	InitialDelay time.Duration
}

// Run starts each job in its own goroutine until ctx is cancelled.
func Run(ctx context.Context, wg *sync.WaitGroup, jobs []Job) {
	for _, job := range jobs {
		if job.Interval <= 0 || job.Run == nil {
			continue
		}
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			runLoop(ctx, j)
		}(job)
	}
}

func runLoop(ctx context.Context, job Job) {
	slog.Info("background worker started", "job", job.Name, "interval", job.Interval)

	var runMu sync.Mutex
	inFlight := false

	// First run after a short delay so main scan loop can start.
	firstDelay := job.InitialDelay
	if firstDelay <= 0 {
		firstDelay = 10 * time.Second
	}
	timer := time.NewTimer(firstDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("background worker stopped", "job", job.Name)
			return
		case <-timer.C:
			if job.SkipIfRunning {
				runMu.Lock()
				if inFlight {
					runMu.Unlock()
					slog.Warn("background worker skip; prior run still active", "job", job.Name)
					timer.Reset(job.Interval)
					continue
				}
				inFlight = true
				runMu.Unlock()

				// Reset interval before Run returns: ticks are wall-clock spaced, not job-duration spaced.
				timer.Reset(job.Interval)
				go func() {
					defer func() {
						runMu.Lock()
						inFlight = false
						runMu.Unlock()
					}()
					runOnce(ctx, job)
				}()
				continue
			}

			runOnce(ctx, job)
			timer.Reset(job.Interval)
		}
	}
}

func runOnce(ctx context.Context, job Job) {
	start := time.Now()
	err := job.Run(ctx)
	// Suppress failure log on shutdown cancel; ctx.Err() is expected during drain.
	if err != nil && ctx.Err() == nil {
		slog.Warn("background worker run failed", "job", job.Name, "error", err, "duration_ms", time.Since(start).Milliseconds())
		return
	}
	slog.Info("background worker run finished", "job", job.Name, "duration_ms", time.Since(start).Milliseconds())
}
