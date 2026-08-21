package gemini

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/metrics"
	"golang.org/x/time/rate"
)

type quotaEntry struct {
	at     time.Time
	tokens int
}

// QuotaLimiter enforces per-priority RPM (token bucket), shared TPM (rolling 60s), and RPD (rolling 24h).
type QuotaLimiter struct {
	mu sync.Mutex

	buckets map[Priority]*rate.Limiter
	embed   *rate.Limiter

	tpmMax    int
	tpmWindow []quotaEntry
	tpmTotal  int
	rpdMax    int
	rpdWindow []time.Time
	stats     QuotaStats
}

type QuotaStats struct {
	GenerateWaits int64
	EmbedWaits    int64
	RPMThrottles  int64
	TPMThrottles  int64
	RPDThrottles  int64
	DailyUsed     int
}

func NewQuotaLimiter(cfg LimitConfig) *QuotaLimiter {
	genRPM := cfg.RPM
	if genRPM <= 0 {
		genRPM = 10
	}
	embedRPM := cfg.EmbedRPM
	if embedRPM <= 0 {
		embedRPM = max(1, genRPM/3)
	}
	tpm := cfg.TPM
	if tpm <= 0 {
		tpm = 250_000
	}
	rpd := cfg.RPD
	if rpd <= 0 {
		rpd = 1500
	}

	split := cfg.QuotaSplit.Normalize()
	buckets := map[Priority]*rate.Limiter{
		PriorityCritical: newPriorityLimiter(genRPM, split, PriorityCritical),
		PriorityHigh:     newPriorityLimiter(genRPM, split, PriorityHigh),
		PriorityNormal:   newPriorityLimiter(genRPM, split, PriorityNormal),
		PriorityLow:      newPriorityLimiter(genRPM, split, PriorityLow),
	}

	return &QuotaLimiter{
		buckets:   buckets,
		embed:     rate.NewLimiter(rate.Limit(float64(embedRPM)/60.0), 1),
		tpmMax:    tpm,
		rpdMax:    rpd,
		tpmWindow: make([]quotaEntry, 0, 32),
		rpdWindow: make([]time.Time, 0, 64),
	}
}

func newPriorityLimiter(totalRPM int, split QuotaSplit, p Priority) *rate.Limiter {
	rpm := split.RPM(totalRPM, p)
	return rate.NewLimiter(rate.Limit(float64(rpm)/60.0), 1)
}

func (q *QuotaLimiter) Stats() QuotaStats {
	if q == nil {
		return QuotaStats{}
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	s := q.stats
	s.DailyUsed = len(q.rpdWindow)
	return s
}

func (q *QuotaLimiter) WaitGenerate(ctx context.Context, priority Priority, estTokens int) error {
	if q == nil {
		return nil
	}
	lim := q.buckets[priority]
	if lim == nil {
		lim = q.buckets[PriorityNormal]
	}
	return q.wait(ctx, lim, estTokens, true)
}

func (q *QuotaLimiter) WaitEmbed(ctx context.Context, estTokens int) error {
	if q == nil {
		return nil
	}
	return q.wait(ctx, q.embed, estTokens, false)
}

func (q *QuotaLimiter) wait(ctx context.Context, lim *rate.Limiter, estTokens int, generate bool) error {
	// Fail fast on RPD/TPM before blocking on RPM token bucket; record usage only after wait succeeds.
	if err := q.checkRPD(); err != nil {
		q.mu.Lock()
		q.stats.RPDThrottles++
		q.mu.Unlock()
		return err
	}
	if err := q.checkTPM(estTokens); err != nil {
		q.mu.Lock()
		q.stats.TPMThrottles++
		q.mu.Unlock()
		return err
	}

	waitStart := time.Now()
	if err := lim.Wait(ctx); err != nil {
		return err
	}
	if waited := time.Since(waitStart); waited > 0 {
		metrics.RecordGeminiWait(waited)
	}

	q.mu.Lock()
	if generate {
		q.stats.GenerateWaits++
	} else {
		q.stats.EmbedWaits++
	}
	q.recordTPM(estTokens)
	q.recordRPD()
	q.mu.Unlock()
	return nil
}

func (q *QuotaLimiter) checkRPD() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pruneRPD(time.Now())
	if len(q.rpdWindow) >= q.rpdMax {
		return fmt.Errorf("gemini daily quota exceeded (%d/%d RPD)", len(q.rpdWindow), q.rpdMax)
	}
	return nil
}

func (q *QuotaLimiter) recordRPD() {
	now := time.Now()
	q.pruneRPD(now)
	q.rpdWindow = append(q.rpdWindow, now)
}

func (q *QuotaLimiter) pruneRPD(now time.Time) {
	cutoff := now.Add(-24 * time.Hour)
	i := 0
	for _, t := range q.rpdWindow {
		if t.After(cutoff) {
			q.rpdWindow[i] = t
			i++
		}
	}
	q.rpdWindow = q.rpdWindow[:i]
}

func (q *QuotaLimiter) checkTPM(estTokens int) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.pruneTPM(time.Now())
	if q.tpmTotal+estTokens > q.tpmMax {
		return fmt.Errorf("gemini tpm budget exceeded (%d+%d > %d)", q.tpmTotal, estTokens, q.tpmMax)
	}
	return nil
}

func (q *QuotaLimiter) recordTPM(estTokens int) {
	now := time.Now()
	q.pruneTPM(now)
	q.tpmWindow = append(q.tpmWindow, quotaEntry{at: now, tokens: estTokens})
	q.tpmTotal += estTokens
}

func (q *QuotaLimiter) pruneTPM(now time.Time) {
	cutoff := now.Add(-time.Minute)
	i := 0
	total := 0
	for _, e := range q.tpmWindow {
		if e.at.After(cutoff) {
			q.tpmWindow[i] = e
			total += e.tokens
			i++
		}
	}
	q.tpmWindow = q.tpmWindow[:i]
	q.tpmTotal = total
}
