package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/signal"
	"syscall"
	"time"

	"github.com/bidshard/parser/internal/crm/config"
	"github.com/bidshard/parser/internal/crm/tail"
	"github.com/bidshard/parser/internal/log"
	"github.com/bidshard/parser/internal/sink"
	"github.com/spf13/cobra"
)

func newTailCmd() *cobra.Command {
	var (
		jsonl    string
		minScore int
		asJSON   bool
	)
	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Stream newly accepted leads to the terminal",
		Long: `Print accepted leads as they are written.

Modes:
  crm-bot tail                     Mongo change stream (MONGO_URI)
  crm-bot tail --jsonl path        Follow local NDJSON export file

Laptop -> VPS (SSH tunnel):
  ssh -N -L 27017:127.0.0.1:27017 root@your-vps &
  MONGO_URI=mongodb://127.0.0.1:27017 crm-bot tail

Or use: lip tail  (remote JSONL over SSH)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			log.Init(cfg.LogFormat, "warn")

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			opts := tail.Options{MinScore: minScore, JSON: asJSON}
			emit := buildEmitter(cmd.OutOrStdout(), asJSON)

			if stringsTrim(jsonl) != "" {
				return tail.FollowJSONL(ctx, jsonl, opts, emit)
			}
			if stringsTrim(cfg.MongoURI) == "" {
				return fmt.Errorf("MONGO_URI empty; use --jsonl or set Mongo connection")
			}

			connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			client, err := sink.ConnectMongoClient(connectCtx, cfg.MongoURI)
			if err != nil {
				return err
			}
			defer func() {
				discCtx, discCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer discCancel()
				_ = client.Disconnect(discCtx)
			}()

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "watching new leads (Ctrl+C to stop)...")
			return tail.WatchMongo(ctx, client, cfg.MongoDB, cfg.MongoCollection, opts, emit)
		},
	}
	cmd.Flags().StringVar(&jsonl, "jsonl", "", "tail NDJSON file instead of Mongo")
	cmd.Flags().IntVar(&minScore, "min-score", 0, "skip leads below this score")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print full JSON per lead")
	return cmd
}

func buildEmitter(out io.Writer, asJSON bool) tail.Emitter {
	return func(doc sink.LeadDoc) {
		if asJSON {
			enc := json.NewEncoder(out)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(doc)
			return
		}
		_, _ = fmt.Fprintln(out, tail.FormatLine(doc))
	}
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
