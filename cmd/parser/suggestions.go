package main

import (
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/discover"
	"github.com/bidshard/parser/internal/suggestions"
	"github.com/spf13/cobra"
)

func newSuggestionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suggestions",
		Short: "List and apply pending keyword/discover diffs",
		Long: `Manage cold-path suggestion files under data/suggestions/.

Examples:
  parser suggestions list
  parser suggestions preview --file data/suggestions/keywords_pending_abc.json
  parser suggestions apply --file data/suggestions/keywords_pending_abc.json --dry-run
  parser suggestions apply --file data/suggestions/discover_icp_pending_abc.json --auto
  parser suggestions reject --file data/suggestions/keywords_pending_abc.json`,
	}
	cmd.AddCommand(
		newSuggestionsListCmd(),
		newSuggestionsPreviewCmd(),
		newSuggestionsApplyCmd(),
		newSuggestionsRejectCmd(),
	)
	return cmd
}

func newSuggestionsListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List pending suggestion files",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := cmd.Flags().GetString("dir")
			if strings.TrimSpace(dir) == "" {
				dir = defaultSuggestionsDir()
			}
			files, err := suggestions.ListPending(dir)
			if err != nil {
				return err
			}
			if len(files) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no pending suggestion files")
				return nil
			}
			for _, f := range files {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", f.Kind, f.Status, f.Path)
			}
			return nil
		},
	}
	c.Flags().String("dir", "", "suggestions directory (default from config)")
	return c
}

func newSuggestionsPreviewCmd() *cobra.Command {
	var file, keywordsPath, icpPath string
	c := &cobra.Command{
		Use:   "preview",
		Short: "Preview merge of one pending file",
		RunE: func(cmd *cobra.Command, args []string) error {
			file = strings.TrimSpace(file)
			if file == "" {
				return fmt.Errorf("--file required")
			}
			kind, _, err := suggestions.PeekPending(file)
			if err != nil {
				return err
			}
			switch kind {
			case suggestions.KindKeywords:
				if keywordsPath == "" {
					keywordsPath = defaultKeywordsPath()
				}
				out, err := suggestions.PreviewKeywords(file, keywordsPath)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), out)
				return nil
			case suggestions.KindDiscover:
				if icpPath == "" {
					icpPath = discover.ResolveICPPath("")
				}
				out, err := suggestions.PreviewDiscover(file, icpPath)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprint(cmd.OutOrStdout(), out)
				return nil
			default:
				return fmt.Errorf("unsupported pending kind %q", kind)
			}
		},
	}
	c.Flags().StringVar(&file, "file", "", "pending JSON path")
	c.Flags().StringVar(&keywordsPath, "keywords", "", "keywords.json path")
	c.Flags().StringVar(&icpPath, "icp", "", "discover.icp.json path")
	return c
}

func newSuggestionsApplyCmd() *cobra.Command {
	var file, keywordsPath, icpPath string
	var dryRun, autoApply bool
	c := &cobra.Command{
		Use:   "apply",
		Short: "Apply one pending suggestion file",
		RunE: func(cmd *cobra.Command, args []string) error {
			file = strings.TrimSpace(file)
			if file == "" {
				return fmt.Errorf("--file required")
			}
			kind, _, err := suggestions.PeekPending(file)
			if err != nil {
				return err
			}
			switch kind {
			case suggestions.KindKeywords:
				if keywordsPath == "" {
					keywordsPath = defaultKeywordsPath()
				}
				if autoApply {
					cfg, err := config.Load()
					if err != nil {
						return err
					}
					summary, err := suggestions.ApplyKeywordsAuto(file, keywordsPath, suggestions.KeywordAutoApplyOptions{
						MaxPerWeek: cfg.DiscoverAutoApplyMaxWeek,
					}, dryRun)
					if err != nil {
						return err
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ok %s\n", summary)
					return nil
				}
				summary, err := suggestions.ApplyKeywords(file, keywordsPath, dryRun)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ok %s\n", summary)
				return nil
			case suggestions.KindDiscover:
				if icpPath == "" {
					icpPath = discover.ResolveICPPath("")
				}
				if autoApply {
					cfg, err := config.Load()
					if err != nil {
						return err
					}
					summary, err := suggestions.ApplyDiscoverAuto(file, icpPath, suggestions.DiscoverAutoApplyOptions{
						MaxPerWeek: cfg.DiscoverAutoApplyMaxWeek,
					}, dryRun)
					if err != nil {
						return err
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ok %s\n", summary)
					return nil
				}
				summary, err := suggestions.ApplyDiscover(file, icpPath, dryRun)
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ok %s\n", summary)
				return nil
			default:
				return fmt.Errorf("unsupported pending kind %q", kind)
			}
		},
	}
	c.Flags().StringVar(&file, "file", "", "pending JSON path")
	c.Flags().StringVar(&keywordsPath, "keywords", "", "keywords.json path")
	c.Flags().StringVar(&icpPath, "icp", "", "discover.icp.json path")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "preview merge without writing")
	c.Flags().BoolVar(&autoApply, "auto", false, "apply with denylist and weekly cap (discover + keywords)")
	return c
}

func newSuggestionsRejectCmd() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "reject",
		Short: "Mark pending file rejected without merge",
		RunE: func(cmd *cobra.Command, args []string) error {
			file = strings.TrimSpace(file)
			if file == "" {
				return fmt.Errorf("--file required")
			}
			if err := suggestions.RejectPending(file); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "rejected %s\n", file)
			return nil
		},
	}
	c.Flags().StringVar(&file, "file", "", "pending JSON path")
	return c
}

func defaultKeywordsPath() string {
	cfg, err := config.Load()
	if err != nil || cfg.KeywordsJSONPath == "" {
		return "data/keywords.json"
	}
	return cfg.KeywordsJSONPath
}

func defaultSuggestionsDir() string {
	cfg, err := config.Load()
	if err != nil || cfg.GeminiKeywordDiffDir == "" {
		return "data/suggestions"
	}
	return cfg.GeminiKeywordDiffDir
}
