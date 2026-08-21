package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/sources/lander"
	"github.com/bidshard/parser/internal/sources/tgweb"
)

func runCollectOnce(
	ctx context.Context,
	cfg config.Config,
	deps *runtimeDeps,
	name string,
	collect func(ctx context.Context, emit func(ctx context.Context, item model.RawItem) error) error,
) error {
	// Bound worker pool lifetime; collect and emit share workCtx so crawl stops when scan timeout fires.
	workCtx, workCancel := context.WithTimeout(ctx, cfg.ScanTimeout)
	defer workCancel()

	taskCh := make(chan pipeline.Task, cfg.TaskBuffer)
	var wg sync.WaitGroup

	pool := pipeline.NewPool(cfg.WorkerCount, deps.processor, cfg.ProcessorTaskTimeout)
	pool.Run(workCtx, &wg, taskCh)

	roundID := newRoundID()
	state := &pipeline.RoundState{}

	err := collect(workCtx, func(emitCtx context.Context, item model.RawItem) error {
		task := pipeline.Task{
			RoundID: roundID,
			Item:    item,
			Stats:   state,
		}
		if err := pipeline.TryEnqueue(emitCtx, taskCh, task); err == nil {
			return nil
		}
		if emitCtx.Err() != nil {
			return emitCtx.Err()
		}
		slog.Warn("collect task channel full", "source", name, "contact", item.MaskedContact())
		return nil
	})

	close(taskCh)

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	drainTimeout := collectDrainTimeout(cfg)
	select {
	case <-waitCh:
	case <-time.After(drainTimeout):
		slog.Warn("collect once drain slow",
			"source", name,
			"timeout", drainTimeout,
			"hint", "waiting for pipeline workers without cancel",
		)
		// Intentionally keep workCtx alive: cancel would abort in-flight Gemini/MX/HTTP
		// mid-request and can leave idle conns until GC (shows as fd_delta in BPF soak).
		<-waitCh
	}

	if err != nil && err != context.Canceled {
		state.Wait()
		deps.flushStore(ctx)
		logCollectOnceFinished(name, state, deps)
		return err
	}

	state.Wait()
	deps.flushStore(ctx)
	logCollectOnceFinished(name, state, deps)
	return nil
}

func logCollectOnceFinished(name string, state *pipeline.RoundState, deps *runtimeDeps) {
	var leadsWritten int64
	if deps.bulkStore != nil {
		leadsWritten = deps.bulkStore.LeadsWritten()
	}
	slog.Info("collect once finished",
		"source", name,
		"raw_total", state.RawTotal.Load(),
		"accepted", state.Accepted.Load(),
		"hard_rejected", state.HardRejected.Load(),
		"dedup", state.Dedup.Load(),
		"rejected_geo", state.RejectedGeo.Load(),
		"leads_buffered", len(state.Leads()),
		"leads_written", leadsWritten,
		"dropped", state.Dropped.Load(),
	)
}

func collectDrainTimeout(cfg config.Config) time.Duration {
	if cfg.CollectDrainTimeout > 0 {
		return cfg.CollectDrainTimeout
	}
	if cfg.ShutdownTimeout > 0 {
		return cfg.ShutdownTimeout
	}
	return 120 * time.Second
}

func runTelegramWebOnce(ctx context.Context, cfg config.Config, deps *runtimeDeps) error {
	var headless lander.HeadlessFetcher = lander.DisabledHeadless{}
	if cfg.LanderHeadless {
		// Cap concurrent browser contexts; default pool returns unavailable until Playwright is wired.
		headless = lander.NewPlaywrightPoolFetcher(2, cfg.HTTPTimeout)
	}
	crawler, err := tgweb.NewCrawler(cfg, nil, headless)
	if err != nil {
		return err
	}
	return runCollectOnce(ctx, cfg, deps, crawler.Name(), func(ctx context.Context, emit func(ctx context.Context, item model.RawItem) error) error {
		return crawler.Collect(ctx, emit)
	})
}
