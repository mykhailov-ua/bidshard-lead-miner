package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/crm/config"
	crmmetrics "github.com/bidshard/parser/internal/crm/metrics"
	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/crm/webhook"
	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/mongo"
)

type Runtime struct {
	mongo           *mongo.Client
	leadStore       *store.LeadStore
	webhook         *webhook.Server
	shutdownTimeout time.Duration
}

func NewRuntime(ctx context.Context, cfg config.Config) (*Runtime, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := sink.ConnectMongoClient(connectCtx, cfg.MongoURI)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	leadStore := store.New(client, store.Options{
		DBName:                 cfg.MongoDB,
		LeadsCollection:        cfg.MongoCollection,
		SourceStatsCollection:  cfg.SourceStatsCollection,
		KeywordStatsCollection: cfg.KeywordStatsCollection,
		CrmBoostCollection:     cfg.CrmBoostCollection,
		LeadNotesCollection:    cfg.LeadNotesCollection,
		LeadCrmMetaCollection:  cfg.LeadCrmMetaCollection,
		QueryTimeout:           cfg.QueryTimeout,
		WriteTimeout:           cfg.WriteTimeout,
		StatsTimeout:           cfg.StatsTimeout,
		ExportMaxRows:          int64(cfg.ExportMaxRows),
		SearchMaxRows:          int64(cfg.SearchMaxRows),
	})

	httpHandler := webhook.NewMux(cfg.WebhookSecret, leadStore)
	httpServer := webhook.NewServer(cfg.WebhookAddr, httpHandler)

	return &Runtime{
		mongo:           client,
		leadStore:       leadStore,
		webhook:         httpServer,
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
}

func (rt *Runtime) Run(ctx context.Context, cfg config.Config) error {
	if rt == nil {
		return fmt.Errorf("crm runtime not initialized")
	}

	_ = crmmetrics.StartServer(ctx, cfg.MetricsAddr)
	_ = StartPprofServer(ctx, cfg.PprofAddr)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if rt.webhook == nil {
			return
		}
		if err := rt.webhook.ListenAndServe(); err != nil && ctx.Err() == nil {
			slog.Error("crm http server stopped", "error", err)
		}
	}()

	<-ctx.Done()

	// Parent ctx is already cancelled; use a fresh timeout for graceful http shutdown.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), rt.shutdownTimeoutDuration())
	defer cancel()

	if rt.webhook != nil {
		if err := rt.webhook.Shutdown(shutdownCtx); err != nil && shutdownCtx.Err() == nil {
			slog.Warn("crm http shutdown failed", "error", err)
		}
	}
	wg.Wait()

	if err := ctx.Err(); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

func (rt *Runtime) shutdownTimeoutDuration() time.Duration {
	if rt == nil || rt.shutdownTimeout <= 0 {
		return 30 * time.Second
	}
	return rt.shutdownTimeout
}

func (rt *Runtime) Close(ctx context.Context) error {
	if rt == nil || rt.mongo == nil {
		return nil
	}
	closeCtx := ctx
	if closeCtx == nil {
		closeCtx = context.Background()
	}
	discCtx, cancel := context.WithTimeout(closeCtx, 5*time.Second)
	defer cancel()
	if err := rt.mongo.Disconnect(discCtx); err != nil && discCtx.Err() == nil {
		return fmt.Errorf("mongo disconnect: %w", err)
	}
	return nil
}

func Run(ctx context.Context, cfg config.Config) error {
	rt, err := NewRuntime(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := rt.Close(closeCtx); err != nil {
			slog.Warn("crm runtime close failed", "error", err)
		}
	}()

	slog.Info("crm-bot started",
		"mongo_db", cfg.MongoDB,
		"collection", cfg.MongoCollection,
		"webhook_addr", cfg.WebhookAddr,
		"metrics_addr", cfg.MetricsAddr,
	)

	err = rt.Run(ctx, cfg)
	if err != nil && err != context.Canceled {
		return fmt.Errorf("crm-bot run: %w", err)
	}
	slog.Info("crm-bot stopped")
	return nil
}
