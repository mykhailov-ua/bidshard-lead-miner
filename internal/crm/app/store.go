package app

import (
	"context"
	"fmt"
	"time"

	"github.com/bidshard/parser/internal/crm/config"
	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/mongo"
)

func OpenLeadStore(ctx context.Context, cfg config.Config) (*store.LeadStore, *mongo.Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := sink.ConnectMongoClient(connectCtx, cfg.MongoURI)
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	leadStore := store.New(client, store.Options{
		DBName:                 cfg.MongoDB,
		LeadsCollection:        cfg.MongoCollection,
		EntityCollection:       cfg.EntityCollection,
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
	return leadStore, client, nil
}

func CloseMongo(client *mongo.Client) {
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = client.Disconnect(ctx)
}
