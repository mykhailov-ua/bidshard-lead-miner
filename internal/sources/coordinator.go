package sources

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
)

type Coordinator struct {
	cfg     config.Config
	sources []Source

	roundMu     sync.Mutex
	activeRound *roundHandle
}

type roundHandle struct {
	cancel context.CancelFunc
	id     string
}

func NewCoordinator(cfg config.Config, sources []Source) *Coordinator {
	return &Coordinator{cfg: cfg, sources: sources}
}

func (c *Coordinator) Run(
	ctx context.Context,
	wg *sync.WaitGroup,
	scanCh <-chan struct{},
	taskCh chan<- pipeline.Task,
	statsCh chan<- pipeline.RoundStats,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.loop(ctx, scanCh, taskCh, statsCh)
	}()
}

func (c *Coordinator) loop(
	ctx context.Context,
	scanCh <-chan struct{},
	taskCh chan<- pipeline.Task,
	statsCh chan<- pipeline.RoundStats,
) {
	var roundWG sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			c.cancelActiveRound()
			roundWG.Wait()
			return
		case <-scanCh:
			c.cancelActiveRound()
			// Wait for the prior round goroutine to exit before starting another; avoids overlapping
			// taskCh pressure and duplicate round_id work when PARSER_POLL_SEC fires early.
			roundWG.Wait()
			roundWG.Add(1)
			go func() {
				defer roundWG.Done()
				c.runRound(ctx, taskCh, statsCh)
			}()
		}
	}
}

func (c *Coordinator) cancelActiveRound() {
	c.roundMu.Lock()
	defer c.roundMu.Unlock()
	if c.activeRound != nil {
		c.activeRound.cancel()
		c.activeRound = nil
	}
}

func (c *Coordinator) beginRound(parent context.Context) (context.Context, *roundHandle) {
	c.roundMu.Lock()
	defer c.roundMu.Unlock()

	// loop.cancelActiveRound may already have cancelled the prior handle; cancel again is idempotent.
	if c.activeRound != nil {
		c.activeRound.cancel()
	}

	roundCtx, cancel := context.WithTimeout(parent, c.cfg.ScanTimeout)
	handle := &roundHandle{cancel: cancel, id: newRoundID()}
	c.activeRound = handle
	return roundCtx, handle
}

func (c *Coordinator) clearRoundHandle(handle *roundHandle) {
	c.roundMu.Lock()
	defer c.roundMu.Unlock()
	if c.activeRound == handle {
		c.activeRound = nil
	}
}

func (c *Coordinator) runRound(
	parent context.Context,
	taskCh chan<- pipeline.Task,
	statsCh chan<- pipeline.RoundStats,
) {
	roundCtx, handle := c.beginRound(parent)
	defer func() {
		handle.cancel()
		c.clearRoundHandle(handle)
	}()

	roundID := handle.id
	start := time.Now()
	state := &pipeline.RoundState{}

	g, gctx := errgroup.WithContext(roundCtx)
	g.SetLimit(c.cfg.SourceConcurrency)

	for _, src := range c.sources {
		src := src
		g.Go(func() error {
			slog.Info("source collect started", "round_id", roundID, "source", src.Name())
			// Collect shares the round deadline (ScanTimeout). Per-request HTTPTimeout is enforced
			// inside fetchers; capping whole Collect at HTTPTimeout truncates multi-URL crawls.
			err := src.Collect(gctx, func(ctx context.Context, item model.RawItem) error {
				return c.emit(ctx, roundID, taskCh, state, item)
			})
			if err != nil {
				if ctxErr := gctx.Err(); ctxErr != nil {
					state.SourcesFail.Add(1)
					slog.Warn("source collect timeout",
						"round_id", roundID,
						"source", src.Name(),
						"error", ctxErr,
					)
					return nil
				}
				state.SourcesFail.Add(1)
				slog.Warn("source collect failed", "round_id", roundID, "source", src.Name(), "error", err)
				return nil
			}
			state.SourcesOK.Add(1)
			slog.Info("source collect finished", "round_id", roundID, "source", src.Name())
			// Return nil so errgroup does not cancel sibling sources on one failure.
			return nil
		})
	}

	_ = g.Wait()

	// Block until processor workers call FinishTask for every enqueued item.
	state.Wait()

	stats := state.Snapshot(roundID, time.Since(start))
	pipeline.TrySendStats(statsCh, stats)

	slog.Info("scan round finished",
		"round_id", stats.RoundID,
		"duration_ms", stats.Duration.Milliseconds(),
		"sources_ok", stats.SourcesOK,
		"sources_fail", stats.SourcesFail,
		"raw", stats.RawTotal,
		"accepted", stats.Accepted,
		"rejected_geo", stats.RejectedGeo,
		"dropped", stats.Dropped,
		"high", stats.High,
		"medium", stats.Medium,
	)
}

func (c *Coordinator) emit(
	ctx context.Context,
	roundID string,
	taskCh chan<- pipeline.Task,
	state *pipeline.RoundState,
	item model.RawItem,
) error {
	task := pipeline.Task{
		RoundID: roundID,
		Item:    item,
		Stats:   state,
	}

	err := pipeline.TryEnqueue(ctx, taskCh, task)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	slog.Warn("task channel full",
		"round_id", roundID,
		"source", item.Source,
		"contact", item.MaskedContact(),
	)
	return nil
}

func (c *Coordinator) ActiveRoundCancel() context.CancelFunc {
	c.roundMu.Lock()
	defer c.roundMu.Unlock()
	if c.activeRound == nil {
		return nil
	}
	return c.activeRound.cancel
}

func newRoundID() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
