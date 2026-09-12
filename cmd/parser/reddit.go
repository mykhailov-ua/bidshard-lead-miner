package main

import (
	"fmt"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/reddit"
	"github.com/spf13/cobra"
)

func newRedditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reddit",
		Short: "Reddit offline tools (not hot poll)",
	}
	cmd.AddCommand(newRedditOfflineArchiveCmd())
	return cmd
}

func newRedditOfflineArchiveCmd() *cobra.Command {
	var since, until, out string
	var noFilter bool

	cmd := &cobra.Command{
		Use:   "offline-archive",
		Short: "M7 offline PullPush/Arctic Shift JSONL export",
		Long: `Batch archive Reddit posts/comments for outreach (1 req/2s, 429 backoff).

Does not register in PARSER_SOURCE hot poll. Subreddits from REDDIT_SUBREDDITS.
Output JSONL fields: author, body, created_utc, permalink.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := globalOpts.apply(&cfg); err != nil {
				return err
			}

			opts := reddit.ArchiveOptionsFromConfig(cfg)
			if since != "" {
				t, err := time.Parse("2006-01-02", since)
				if err != nil {
					return fmt.Errorf("invalid --since: %w", err)
				}
				opts.Since = t.UTC()
			}
			if until != "" {
				t, err := time.Parse("2006-01-02", until)
				if err != nil {
					return fmt.Errorf("invalid --until: %w", err)
				}
				opts.Until = t.UTC().Add(24*time.Hour - time.Second)
			}
			if out != "" {
				opts.Out = out
			}
			opts.Filter = !noFilter

			res, err := reddit.RunArchive(cmd.Context(), opts)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"reddit offline archive: written=%d filtered=%d fetched=%d out=%s\n",
				res.Written, res.Filtered, res.Fetched, opts.Out,
			)
			return nil
		},
	}

	cmd.Flags().StringVar(&since, "since", "2025-01-01", "start date YYYY-MM-DD")
	cmd.Flags().StringVar(&until, "until", "", "end date YYYY-MM-DD (default: now)")
	cmd.Flags().StringVar(&out, "out", "data/export/reddit_archive.ndjson", "output JSONL path")
	cmd.Flags().BoolVar(&noFilter, "no-filter", false, "write all rows (skip buyer/tracker gate)")
	return cmd
}
