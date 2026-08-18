package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bidshard/parser/internal/app"
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/log"
	"github.com/spf13/cobra"
)

// Version is set at link time: -ldflags "-X github.com/bidshard/parser/cmd/parser/cmd.Version=..."
var Version = "dev"

type cliOpts struct {
	source          string
	output          string
	logFormat       string
	landerHeadless  bool
	scanOnce        bool
	telegramSidecar bool
	telegramDryRun  bool
	ingestStdin     bool
	fixture         string
}

func (o *cliOpts) bindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&o.source, "source", "", "source set: stub|forum|supply|lander|reddit|discord|warrior|ct|github|reviews|all (overrides PARSER_SOURCE)")
	cmd.PersistentFlags().StringVar(&o.output, "output", "", "output format: table|ndjson|quiet (overrides PARSER_OUTPUT)")
	cmd.PersistentFlags().StringVar(&o.logFormat, "log-format", "", "log format: json|text (overrides PARSER_LOG_FORMAT)")
	cmd.PersistentFlags().BoolVar(&o.landerHeadless, "lander-headless", false, "enable Playwright headless lander fetch")
}

func (o *cliOpts) bindLegacyFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.scanOnce, "scan-once", false, "deprecated: use 'parser scan'")
	cmd.Flags().BoolVar(&o.ingestStdin, "ingest-stdin", false, "deprecated: use 'parser ingest'")
	cmd.Flags().BoolVar(&o.telegramSidecar, "telegram-sidecar", false, "deprecated: use 'parser telegram'")
	cmd.Flags().BoolVar(&o.telegramDryRun, "telegram-dry-run", false, "Telethon dry-run (no MTProto session)")
	cmd.Flags().StringVar(&o.fixture, "fixture", "", "NDJSON fixture file (deprecated: use 'parser ingest --fixture')")

	_ = cmd.Flags().MarkDeprecated("scan-once", "use 'parser scan'")
	_ = cmd.Flags().MarkDeprecated("ingest-stdin", "use 'parser ingest'")
	_ = cmd.Flags().MarkDeprecated("telegram-sidecar", "use 'parser telegram'")
	_ = cmd.Flags().MarkDeprecated("fixture", "use 'parser ingest --fixture'")
}

func (o *cliOpts) apply(cfg *config.Config) error {
	if o.source != "" {
		cfg.Source = o.source
	}
	if o.output != "" {
		cfg.Output = o.output
	}
	if o.logFormat != "" {
		cfg.LogFormat = o.logFormat
	}
	if o.landerHeadless {
		cfg.LanderHeadless = true
	}
	if o.scanOnce {
		cfg.ScanOnce = true
	}
	if o.ingestStdin {
		cfg.IngestStdin = true
		cfg.IngestReader = os.Stdin
		cfg.ScanOnce = true
	}
	if o.telegramSidecar {
		cfg.TelegramSidecar = true
		cfg.ScanOnce = true
	}
	if o.telegramDryRun {
		cfg.TelegramDryRun = true
	}
	if o.fixture != "" {
		f, err := os.Open(o.fixture)
		if err != nil {
			return fmt.Errorf("open fixture: %w", err)
		}
		cfg.IngestStdin = true
		cfg.IngestReader = f
		cfg.ScanOnce = true
	}
	return nil
}

func (o *cliOpts) run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := o.apply(&cfg); err != nil {
		return err
	}

	log.Init(cfg.LogFormat, cfg.LogLevel)

	if closer, ok := cfg.IngestReader.(io.Closer); ok && o.fixture != "" {
		defer closer.Close()
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

var globalOpts cliOpts

var rootCmd = &cobra.Command{
	Use:   "parser",
	Short: "BidShard lead miner — crawl, score, and store affiliate leads",
	Long: `BidShard lead miner crawls public gray-market signals, scores intent,
and stores qualified leads for the BidShard tracker.

Quick start:
  parser help              Show all commands
  parser scan              One scan round (no API keys needed with stub source)
  parser run               Continuous polling loop
  parser config check      Validate .env and seed files

Configuration comes from environment variables (see .env.example).
Flags override env for source, output, and log format.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q — run 'parser help' for usage", args[0])
		}
		return globalOpts.run(cmd.Context())
	},
}

// Execute runs the CLI.
func Execute() {
	rootCmd.SetContext(context.Background())
	if len(os.Args) > 1 {
		rootCmd.SetArgs(normalizeLegacyArgs(os.Args[1:]))
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// normalizeLegacyArgs maps single-dash long flags (e.g. -scan-once, -output=quiet) to pflag form.
func normalizeLegacyArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, normalizeLegacyArg(a))
	}
	return out
}

func normalizeLegacyArg(a string) string {
	if !strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") {
		return a
	}
	body := strings.TrimPrefix(a, "-")
	if body == "" {
		return a
	}
	// Keep single-letter shorthands such as -h.
	if !strings.Contains(body, "=") && len(body) == 1 {
		return a
	}
	return "--" + body
}

func init() {
	globalOpts.bindFlags(rootCmd)
	globalOpts.bindLegacyFlags(rootCmd)

	rootCmd.AddCommand(
		newRunCmd(),
		newScanCmd(),
		newTelegramCmd(),
		newIngestCmd(),
		newVersionCmd(),
		newSourcesCmd(),
		newConfigCmd(),
	)
}
