package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/crm/config"
	"github.com/bidshard/parser/internal/crm/osint"
	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/log"
	"github.com/bidshard/parser/internal/sink"
	"github.com/spf13/cobra"
)

func newSyncOSINTCmd() *cobra.Command {
	var profilesPath, membersPath, collection string
	cmd := &cobra.Command{
		Use:   "sync-osint",
		Short: "Merge Telethon people JSON into Mongo telegram_people",
		Long: `Reads data/runtime/telegram_user_profiles.json and telegram_chat_members.json
(or paths from env) and upserts telegram_people for CRM export.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			log.Init(cfg.LogFormat, cfg.LogLevel)

			if profilesPath == "" {
				profilesPath = envDefault("TELEGRAM_USER_PROFILES_EXPORT", "data/runtime/telegram_user_profiles.json")
			}
			if membersPath == "" {
				membersPath = envDefault("TELEGRAM_PARTICIPANTS_EXPORT", "data/runtime/telegram_chat_members.json")
			}
			if collection == "" {
				collection = envDefault("CRM_TELEGRAM_PEOPLE_COLLECTION", "telegram_people")
			}

			rows, err := osint.MergePeopleFiles(profilesPath, membersPath)
			if err != nil {
				return err
			}
			if len(rows) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "sync-osint: no rows (profiles=%s members=%s)\n", profilesPath, membersPath)
				return nil
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			client, err := sink.ConnectMongoClient(ctx, cfg.MongoURI)
			if err != nil {
				return fmt.Errorf("mongo connect: %w", err)
			}
			defer func() { _ = client.Disconnect(context.Background()) }()

			leadStore := store.New(client, store.Options{
				DBName:          cfg.MongoDB,
				LeadsCollection: cfg.MongoCollection,
				QueryTimeout:    cfg.QueryTimeout,
				WriteTimeout:    cfg.WriteTimeout,
			})
			n, err := leadStore.SyncTelegramPeople(ctx, collection, rows)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "sync-osint: upserted=%d collection=%s rows_in=%d\n", n, collection, len(rows))
			return nil
		},
	}
	cmd.Flags().StringVar(&profilesPath, "profiles", "", "telegram_user_profiles.json path")
	cmd.Flags().StringVar(&membersPath, "members", "", "telegram_chat_members.json path")
	cmd.Flags().StringVar(&collection, "collection", "", "Mongo collection (default telegram_people)")
	return cmd
}

func envDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
