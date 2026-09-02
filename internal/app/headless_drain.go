package app

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/extract"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/sources/lander"
)

// RunHeadlessDrainCLI loads deps, drains the queue, and flushes the store.
func RunHeadlessDrainCLI(ctx context.Context, cfg config.Config) error {
	deps, err := buildDeps(ctx, cfg)
	if err != nil {
		return err
	}
	defer deps.flushStore(ctx)
	defer deps.closeMongo(ctx)
	return RunHeadlessDrain(ctx, cfg, deps)
}

// RunHeadlessDrain processes deferred headless queue items (nightly cron / compose profile).
func RunHeadlessDrain(ctx context.Context, cfg config.Config, deps *runtimeDeps) error {
	if deps == nil || deps.processor == nil {
		return nil
	}
	limit := cfg.LanderHeadlessDrainLimit
	if limit <= 0 {
		limit = 25
	}
	items, err := lander.LoadPendingHeadless(cfg.LanderHeadlessQueuePath, limit, 3)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		slog.Info("headless drain skipped", "reason", "queue empty")
		return nil
	}

	start := time.Now()
	pool := lander.NewPlaywrightPoolFetcher(cfg.LanderHeadlessMaxBrowsers, cfg.HTTPTimeout)

	workCtx, workCancel := context.WithTimeout(ctx, cfg.ScanTimeout)
	defer workCancel()

	taskCh := make(chan pipeline.Task, cfg.TaskBuffer)
	var wg sync.WaitGroup
	workerPool := pipeline.NewPool(cfg.WorkerCount, deps.processor, cfg.ProcessorTaskTimeout)
	workerPool.Run(workCtx, &wg, taskCh)

	roundID := newRoundID()
	state := &pipeline.RoundState{}
	drained := 0
	failed := 0

	for _, item := range items {
		select {
		case <-workCtx.Done():
			close(taskCh)
			wg.Wait()
			return workCtx.Err()
		default:
		}

		html, err := pool.Fetch(workCtx, item.URL)
		if err != nil {
			slog.Warn("headless drain fetch failed", "url", item.URL, "error", err)
			_ = lander.BumpHeadlessQueueAttempts(cfg.LanderHeadlessQueuePath, item.URL)
			failed++
			continue
		}
		text, _ := lander.TextForContactExtract(html)
		contacts := extract.Extract(text)
		contacts.Contacts = extract.FilterJunkContacts(contacts.Contacts)
		if contacts.Rejected || len(contacts.Contacts) == 0 {
			slog.Warn("headless drain no contacts", "url", item.URL)
			_ = lander.BumpHeadlessQueueAttempts(cfg.LanderHeadlessQueuePath, item.URL)
			failed++
			continue
		}

		formatted := extract.FormatAll(contacts.Contacts)
		rawItem := model.RawItem{
			Source:    lander.RawSourceLabel(item),
			Raw:       text,
			Contact:   formatted[0],
			CrawlHTML: model.LimitCrawlHTML(html),
		}
		task := pipeline.Task{
			RoundID: roundID,
			Item:    rawItem,
			Stats:   state,
		}
		if err := pipeline.TryEnqueue(workCtx, taskCh, task); err != nil {
			slog.Warn("headless drain enqueue failed", "url", item.URL, "error", err)
			_ = lander.BumpHeadlessQueueAttempts(cfg.LanderHeadlessQueuePath, item.URL)
			failed++
			continue
		}
		if err := lander.RemoveHeadlessQueueItem(cfg.LanderHeadlessQueuePath, item.URL); err != nil {
			slog.Warn("headless drain queue remove failed", "url", item.URL, "error", err)
		}
		drained++
		metrics.RecordHeadlessDrained(1)
	}

	close(taskCh)
	wg.Wait()
	_ = lander.TrimHeadlessQueue(cfg.LanderHeadlessQueuePath, cfg.LanderHeadlessQueueMax)

	slog.Info("headless drain finished",
		"processed", len(items),
		"drained", drained,
		"failed", failed,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}
