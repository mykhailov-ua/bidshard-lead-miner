package app

import (
	"context"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/output"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/sources"
)

func runScheduler(ctx context.Context, wg *sync.WaitGroup, pollInterval time.Duration, scanOnce bool, scanCh chan<- struct{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		trigger := func() {
			select {
			case scanCh <- struct{}{}:
			case <-ctx.Done():
			}
		}

		if scanOnce {
			trigger()
			return
		}

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				select {
				case scanCh <- struct{}{}:
				default:
				}
			}
		}
	}()
}

func Run(ctx context.Context, cfg config.Config) error {
	if err := ValidateTelethonForRun(cfg); err != nil {
		return err
	}

	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		return err
	}
	defer deps.flushStore(ctx)
	defer deps.closeMongo(ctx)

	// coldPath/warmPath goroutines exit on ctx; pending warm batches may be lost if shutdown is abrupt.

	if cfg.IngestStdin && cfg.IngestReader != nil {
		return runIngestOnce(ctx, cfg, deps, cfg.IngestReader)
	}

	if cfg.TelegramSidecar {
		return runTelegramSidecarOnce(ctx, cfg, deps)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskCh := make(chan pipeline.Task, cfg.TaskBuffer)
	scanCh := make(chan struct{}, 1)
	statsCh := make(chan pipeline.RoundStats, 8)

	var wg sync.WaitGroup
	onceDone := make(chan struct{}, 1)

	if deps.coldPath != nil {
		deps.coldPath.Run(ctx, &wg)
	}
	if deps.warmPath != nil {
		deps.warmPath.Run(ctx, &wg)
	}

	startBackgroundWorkers(ctx, cfg, deps, &wg)

	pool := pipeline.NewPool(cfg.WorkerCount, deps.processor, cfg.ProcessorTaskTimeout)
	pool.Run(ctx, &wg, taskCh)

	reporter := output.NewReporter(cfg.Output, nil)
	if cfg.ScanOnce {
		reporter.SetOnReport(func() {
			select {
			case onceDone <- struct{}{}:
			default:
			}
		})
	}
	reporter.Run(ctx, &wg, statsCh)

	coordinator := sources.NewCoordinator(cfg, sources.Build(cfg))
	coordinator.Run(ctx, &wg, scanCh, taskCh, statsCh)

	runScheduler(ctx, &wg, cfg.PollInterval, cfg.ScanOnce, scanCh)

	if cfg.ScanOnce {
		select {
		case <-onceDone:
			cancel()
			Drain(cancel, &wg, taskCh, cfg.ShutdownTimeout)
			return nil
		case <-ctx.Done():
			Drain(cancel, &wg, taskCh, cfg.ShutdownTimeout)
			return ctx.Err()
		}
	}

	<-ctx.Done()
	Drain(cancel, &wg, taskCh, cfg.ShutdownTimeout)
	return ctx.Err()
}
