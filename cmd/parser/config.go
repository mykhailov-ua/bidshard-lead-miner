package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/seedcheck"
	"github.com/bidshard/parser/internal/sink"
	"github.com/bidshard/parser/internal/sources"
	"github.com/bidshard/parser/internal/telethon"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate configuration",
	}

	var showFormat string
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Print effective configuration (secrets masked)",
		Long: `Show resolved config after .env and flags.

Examples:
  parser config show
  parser config show --format=json | jq .export_json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow(cmd.Context(), cmd.OutOrStdout(), showFormat)
		},
	}
	showCmd.Flags().StringVar(&showFormat, "format", "text", "output format: text|json")
	cmd.AddCommand(showCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Validate .env, seed files, and optional Mongo connectivity",
		Long: `Load configuration from the environment and report problems.

Does not print secret values - only whether required tokens are set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigCheck(cmd.Context(), cmd.OutOrStdout())
		},
	})

	return cmd
}

func runConfigShow(ctx context.Context, out io.Writer, format string) error {
	_ = ctx
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	if err := globalOpts.apply(&cfg); err != nil {
		return err
	}

	view := configView(cfg)
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		enc := json.NewEncoder(out)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		return enc.Encode(view)
	default:
		keys := []string{
			"source", "output", "log_format", "log_level",
			"export_json", "export_format",
			"mongo_uri", "mongo_db", "mongo_collection",
			"poll_sec", "workers",
			"gemini_api_key", "telegram_api_id", "telegram_api_hash",
			"keywords",
		}
		for _, k := range keys {
			if v, ok := view[k]; ok {
				_, _ = fmt.Fprintf(out, "%s=%v\n", k, v)
			}
		}
		return nil
	}
}

func configView(cfg config.Config) map[string]any {
	return map[string]any{
		"source":            cfg.Source,
		"output":            cfg.Output,
		"log_format":        cfg.LogFormat,
		"log_level":         cfg.LogLevel,
		"export_json":       cfg.ExportJSONPath,
		"export_format":     cfg.ExportJSONFormat,
		"mongo_uri":         maskSecret(cfg.MongoURI),
		"mongo_db":          cfg.MongoDB,
		"mongo_collection":  cfg.MongoCollection,
		"poll_sec":          int(cfg.PollInterval.Seconds()),
		"workers":           cfg.WorkerCount,
		"gemini_api_key":    secretStatus(cfg.GeminiAPIKey),
		"telegram_api_id":   secretStatus(cfg.TelegramAPIID != 0),
		"telegram_api_hash": secretStatus(cfg.TelegramAPIHash != ""),
		"keywords":          cfg.KeywordsJSONPath,
	}
}

func secretStatus(set any) string {
	switch v := set.(type) {
	case bool:
		if v {
			return "set"
		}
	case string:
		if strings.TrimSpace(v) != "" {
			return "set"
		}
	}
	return ""
}

func maskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "set"
}

func runConfigCheck(ctx context.Context, out io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	if err := globalOpts.apply(&cfg); err != nil {
		return err
	}

	var warnings, errors []string

	checkFile := func(label, path string) {
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("missing %s: %s", label, path))
			} else {
				errors = append(errors, fmt.Sprintf("%s %s: %v", label, path, err))
			}
		}
	}

	checkFile("keywords", cfg.KeywordsJSONPath)
	checkFile("keywords gray", cfg.KeywordsGrayPath)
	for _, path := range config.KeywordOverlayPaths(cfg.KeywordsLocale, cfg.KeywordsLocalePath) {
		checkFile("keywords locale overlay", path)
	}
	checkFile("disposable domains", cfg.DisposableDomainsPath)

	active := sources.ParseSourceNames(cfg.Source)
	for _, name := range active {
		for _, info := range sources.Catalog() {
			if info.Name != name {
				continue
			}
			for _, envKey := range info.Requires {
				if envDefaultEmpty(envKey) && os.Getenv(envKey) == "" {
					warnings = append(warnings, fmt.Sprintf("source %q may need %s", name, envKey))
				}
			}
			for _, path := range seedPathForSource(name, cfg) {
				checkFile(fmt.Sprintf("seed for %s", name), path)
			}
		}
	}

	if cfg.MongoURI != "" {
		if err := pingMongo(ctx, cfg.MongoURI); err != nil {
			warnings = append(warnings, fmt.Sprintf("MongoDB ping failed: %v", err))
		} else {
			_, _ = fmt.Fprintln(out, "ok  MongoDB reachable")
		}
	} else if cfg.ExportJSONPath == "" {
		warnings = append(warnings, "no MONGO_URI or PARSER_EXPORT_JSON - leads will not be persisted")
	}

	if cfg.GeminiAPIKey == "" {
		errors = append(errors, config.GeminiMisconfigErrors(cfg, containsSource(active, "tgweb"))...)
	}
	if cfg.CRMWebhookEnabled && strings.TrimSpace(cfg.CRMWebhookURL) == "" {
		errors = append(errors, "PARSER_CRM_WEBHOOK enabled but PARSER_CRM_WEBHOOK_URL empty")
	}
	if strings.TrimSpace(cfg.CRMWebhookURL) != "" {
		if err := sink.ValidateWebhookURL(cfg.CRMWebhookURL); err != nil {
			errors = append(errors, err.Error())
		} else {
			_, _ = fmt.Fprintln(out, "ok  crm webhook URL set")
		}
	}
	for _, w := range config.GeoComplianceWarnings(cfg) {
		warnings = append(warnings, w)
	}
	for _, e := range config.GeoComplianceErrors(cfg, seedcheck.Profile() == seedcheck.ProfileProd) {
		errors = append(errors, e)
	}
	if config.SyncGeoGateConfigured(cfg) {
		_, _ = fmt.Fprintln(out, "ok  sync geo gate configured (inline before Mongo write)")
	}
	if seedcheck.Profile() == seedcheck.ProfileProd {
		if len(cfg.ProxyURLs) == 0 && (containsSource(active, "forum") || containsSource(active, "tgweb")) {
			warnings = append(warnings, "prod profile: PARSER_PROXY_LIST empty - forum/tgweb may fail on datacenter VPS")
		}
		for src, path := range map[string]string{
			"forum":  cfg.ForumSeedPath,
			"supply": cfg.SupplySeedPath,
			"lander": cfg.LanderSeedPath,
		} {
			if !containsSource(active, src) {
				continue
			}
			if fixture, marker := seedcheck.FileLooksLikeFixture(path); fixture {
				errors = append(errors, fmt.Sprintf("prod profile: %s seed has fixture marker %q in %s", src, marker, path))
			}
		}
		if containsSource(active, "supply") && !strings.Contains(cfg.SupplySeedPath, "domains.prod") {
			warnings = append(warnings, fmt.Sprintf("prod profile: prefer SUPPLY_SEED_PATH=data/seeds/domains.prod.csv (got %s)", cfg.SupplySeedPath))
		}
		if containsSource(active, "lander") && !strings.Contains(cfg.LanderSeedPath, "lander_urls.prod") {
			warnings = append(warnings, fmt.Sprintf("prod profile: prefer LANDER_SEED_PATH=data/seeds/lander_urls.prod.csv (got %s)", cfg.LanderSeedPath))
		}
		if containsSource(active, "forum") && !strings.Contains(cfg.ForumSeedPath, "forum_threads.live") {
			warnings = append(warnings, fmt.Sprintf("prod profile: prefer FORUM_SEED_PATH=data/seeds/forum_threads.live.csv (got %s)", cfg.ForumSeedPath))
		}
		for src, path := range map[string]string{
			"forum":  cfg.ForumSeedPath,
			"supply": cfg.SupplySeedPath,
			"lander": cfg.LanderSeedPath,
		} {
			if !containsSource(active, src) {
				continue
			}
			n, err := seedcheck.CountDataRows(path)
			if err != nil {
				errors = append(errors, fmt.Sprintf("prod profile: %s seed unreadable: %s", src, path))
				continue
			}
			min := seedcheck.ProdSeedMinRows[src]
			if n < min {
				errors = append(errors, fmt.Sprintf("prod profile: %s seed has %d rows (need >=%d in %s)", src, n, min, path))
			}
		}
	}

	if cfg.TelegramAPIID > 0 && cfg.TelegramAPIHash != "" {
		if cfg.TelegramProxyURL != "" {
			_, _ = fmt.Fprintln(out, "ok  telethon MTProto proxy configured (TELEGRAM_PROXY_URL)")
		}
		sessionPath := telethon.SessionPath(cfg.TelegramConfigPath)
		if _, err := os.Stat(sessionPath); err != nil {
			msg := fmt.Sprintf("telethon session missing: %s (run: parser telegram login --qr)", sessionPath)
			if os.IsNotExist(err) {
				if seedcheck.Profile() == seedcheck.ProfileProd && cfg.BGTelegramEnabled {
					errors = append(errors, msg)
				} else {
					warnings = append(warnings, msg)
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("telethon session %s: %v", sessionPath, err))
			}
		} else {
			_, _ = fmt.Fprintf(out, "ok  telethon session: %s\n", sessionPath)
		}
		if cfg.BGTelegramEnabled {
			_, _ = fmt.Fprintln(out, "ok  telegram bg worker enabled (PARSER_BG_TELEGRAM=true)")
		}
	}

	if containsSource(active, "forum") {
		if n := countForumFixtureSeeds(cfg.ForumSeedPath); n > 0 && seedcheck.Profile() != seedcheck.ProfileProd {
			_, _ = fmt.Fprintf(out, "ok  forum dev fixture seeds: %d (forum-fixture.test)\n", n)
		}
	}

	for _, w := range warnings {
		_, _ = fmt.Fprintln(out, "warn", w)
	}
	for _, e := range errors {
		_, _ = fmt.Fprintln(out, "error", e)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%d error(s) - fix config and retry", len(errors))
	}

	_, _ = fmt.Fprintf(out, "ok  config loaded (source=%s)\n", cfg.Source)
	return nil
}

func seedPathForSource(name string, cfg config.Config) []string {
	switch name {
	case "forum":
		return []string{cfg.ForumSeedPath}
	case "supply":
		return []string{cfg.SupplySeedPath}
	case "lander":
		return []string{cfg.LanderSeedPath}
	case "warrior":
		return []string{cfg.WarriorSeedPath}
	default:
		return nil
	}
}

func pingMongo(ctx context.Context, uri string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	return client.Ping(ctx, nil)
}

func envDefaultEmpty(key string) bool {
	// Seed paths have defaults on disk; warn only when the path file is missing (handled above).
	switch key {
	case "FORUM_SEED_PATH", "SUPPLY_SEED_PATH", "LANDER_SEED_PATH", "WARRIOR_SEED_PATH":
		return false
	default:
		return true
	}
}

func containsSource(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func countForumFixtureSeeds(path string) int {
	if path == "" {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "url,") {
			continue
		}
		if strings.Contains(line, "forum-fixture.test") {
			count++
		}
	}
	return count
}
