package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/log"
)

func main() {
	scanOnce := flag.Bool("scan-once", false, "run a single scan round and exit")
	ingestStdin := flag.Bool("ingest-stdin", false, "ingest NDJSON tasks from stdin")
	telegramSidecar := flag.Bool("telegram-sidecar", false, "run Telethon sidecar and ingest stdout")
	telegramDryRun := flag.Bool("telegram-dry-run", false, "Telethon dry-run (no MTProto session)")
	source := flag.String("source", "", "source set: stub|supply|forum|lander (override PARSER_SOURCE)")
	landerHeadless := flag.Bool("lander-headless", false, "enable Playwright headless lander fetch (P2)")
	fixture := flag.String("fixture", "", "NDJSON fixture file (implies -scan-once -ingest-stdin)")
	output := flag.String("output", "", "override PARSER_OUTPUT (table|ndjson|quiet)")
	logFormat := flag.String("log-format", "", "override PARSER_LOG_FORMAT (json|text)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	if *scanOnce {
		cfg.ScanOnce = true
	}
	if *source != "" {
		cfg.Source = *source
	}
	if *ingestStdin {
		cfg.IngestStdin = true
		cfg.IngestReader = os.Stdin
		cfg.ScanOnce = true
	}
	if *telegramSidecar {
		cfg.TelegramSidecar = true
		cfg.ScanOnce = true
	}
	if *telegramDryRun {
		cfg.TelegramDryRun = true
	}
	if *landerHeadless {
		cfg.LanderHeadless = true
	}
	if *fixture != "" {
		f, err := os.Open(*fixture)
		if err != nil {
			slog.Error("open fixture failed", "error", err)
			os.Exit(1)
		}
		defer f.Close()
		cfg.IngestStdin = true
		cfg.IngestReader = f
		cfg.ScanOnce = true
	}
	if *output != "" {
		cfg.Output = *output
	}
	if *logFormat != "" {
		cfg.LogFormat = *logFormat
	}

	log.Init(cfg.LogFormat, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("run failed", "error", err)
		os.Exit(1)
	}
}
