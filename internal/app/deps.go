package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/dedup"
	"github.com/bidshard/parser/internal/httpclient"
	"github.com/bidshard/parser/internal/ingest"
	"github.com/bidshard/parser/internal/output"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/bidshard/parser/internal/scoring"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/telethon"
	"github.com/bidshard/parser/internal/validate"
)

type runtimeDeps struct {
	processor *pipeline.Processor
	bulkStore *sink.BulkStore
}

func buildDeps(ctx context.Context, cfg config.Config) (*runtimeDeps, error) {
	reg := scoring.NewRegistry(cfg.KeywordsJSONPath)
	if err := reg.LoadWithOverlay(ctx, cfg.KeywordsJSONPath, cfg.KeywordsGrayPath); err != nil {
		return nil, err
	}

	_ = httpclient.Shared(cfg.HTTPTimeout)

	mx := validate.MXValidator(validate.StubMX{OK: true})
	if cfg.MXCheck {
		mx = validate.Resolver{}
	}

	inner := sink.OpenStore(ctx, cfg.MongoURI, cfg.MongoDB, cfg.MongoCollection, cfg.WriteSlots, cfg.ExportJSONPath)
	bulk := sink.NewBulkStore(inner, 50, 2*time.Second)

	return &runtimeDeps{
		processor: &pipeline.Processor{
			Registry: reg,
			Seen:     dedup.NewSeenCache(50_000, 24*time.Hour),
			Store:    bulk,
			MX:       mx,
		},
		bulkStore: bulk,
	}, nil
}

func (d *runtimeDeps) flushStore(ctx context.Context) {
	if d == nil || d.bulkStore == nil {
		return
	}
	if err := d.bulkStore.Flush(ctx); err != nil {
		slog.Warn("bulk store flush failed", "error", err)
	}
}

func runIngestOnce(ctx context.Context, cfg config.Config, deps *runtimeDeps, reader io.Reader) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskCh := make(chan pipeline.Task, cfg.TaskBuffer)
	statsCh := make(chan pipeline.RoundStats, 8)

	var wg sync.WaitGroup
	onceDone := make(chan struct{}, 1)

	pool := pipeline.NewPool(cfg.WorkerCount, deps.processor)
	pool.Run(ctx, &wg, taskCh)

	reporter := output.NewReporter(cfg.Output, nil)
	reporter.SetOnReport(func() {
		select {
		case onceDone <- struct{}{}:
		default:
		}
	})
	reporter.Run(ctx, &wg, statsCh)

	roundID := newRoundID()
	state := &pipeline.RoundState{}
	start := time.Now()

	ingest.Scan(ctx, reader, taskCh, state, roundID)
	state.Wait()
	stats := state.Snapshot(roundID, time.Since(start))
	select {
	case statsCh <- stats:
	default:
	}

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

	select {
	case <-onceDone:
	case <-time.After(2 * time.Second):
	}

	cancel()
	Drain(cancel, &wg, taskCh, cfg.ShutdownTimeout)
	return nil
}

func newRoundID() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func runTelegramSidecarOnce(ctx context.Context, cfg config.Config, deps *runtimeDeps) error {
	pr, pw := io.Pipe()

	sidecarCtx, sidecarCancel := context.WithCancel(ctx)
	defer sidecarCancel()

	errCh := make(chan error, 1)
	go func() {
		err := telethon.Run(sidecarCtx, telethon.Options{
			ConfigPath: cfg.TelegramConfigPath,
			DryRun:     cfg.TelegramDryRun,
			Once:       true,
		}, pw)
		_ = pw.Close()
		errCh <- err
	}()

	ingestErr := runIngestOnce(ctx, cfg, deps, pr)
	sidecarErr := <-errCh

	if ingestErr != nil {
		return ingestErr
	}
	if sidecarErr != nil && sidecarErr != context.Canceled {
		return sidecarErr
	}
	return nil
}
