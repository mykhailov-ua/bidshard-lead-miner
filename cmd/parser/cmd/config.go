package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate configuration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Validate .env, seed files, and optional Mongo connectivity",
		Long: `Load configuration from the environment and report problems.

Does not print secret values — only whether required tokens are set.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigCheck(cmd.Context(), cmd.OutOrStdout())
		},
	})

	return cmd
}

func runConfigCheck(ctx context.Context, out io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
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
			fmt.Fprintln(out, "ok  MongoDB reachable")
		}
	} else if cfg.ExportJSONPath == "" {
		warnings = append(warnings, "no MONGO_URI or PARSER_EXPORT_JSON — leads will not be persisted")
	}

	if cfg.GeminiAPIKey == "" && (cfg.ParserICPClassify || cfg.ParserGeoClassify) {
		warnings = append(warnings, "GEMINI_API_KEY unset — ICP/geo classify will be skipped")
	}

	for _, w := range warnings {
		fmt.Fprintln(out, "warn", w)
	}
	for _, e := range errors {
		fmt.Fprintln(out, "error", e)
	}

	if len(errors) > 0 {
		return fmt.Errorf("%d error(s) — fix config and retry", len(errors))
	}

	fmt.Fprintf(out, "ok  config loaded (source=%s)\n", cfg.Source)
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
	defer client.Disconnect(context.Background())

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
