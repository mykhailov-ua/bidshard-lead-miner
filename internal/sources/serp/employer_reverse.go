package serp

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sources/jobboard"
)

// HarvestEmployerReverse runs TG/open-web reverse lookups for employers in the registry.
func (c *Crawler) HarvestEmployerReverse(ctx context.Context, cfg EmployerReverseConfig) error {
	if cfg.EmployerRegistryPath == "" {
		cfg.EmployerRegistryPath = jobboard.DefaultEmployerRegistryPath
	}
	if cfg.TGChannelsPath == "" {
		cfg.TGChannelsPath = defaultTGChannelsPath
	}
	if cfg.EmployerTGQueriesPath == "" {
		cfg.EmployerTGQueriesPath = jobboard.DefaultEmployerTGQueriesPath
	}
	if cfg.JobboardRegistryPath == "" {
		cfg.JobboardRegistryPath = defaultJobboardRegistryPath
	}
	if cfg.MaxPerRun <= 0 {
		cfg.MaxPerRun = 15
	}
	if cfg.RescanDays <= 0 {
		cfg.RescanDays = 7
	}

	addedEmployers, err := jobboard.SyncEmployersFromJobRegistry(cfg.JobboardRegistryPath, cfg.EmployerRegistryPath)
	if err != nil {
		slog.Warn("employer sync from job registry failed", "error", err)
	} else if addedEmployers > 0 {
		slog.Info("employer registry synced from jobs", "added", addedEmployers)
	}

	f, err := jobboard.LoadEmployers(cfg.EmployerRegistryPath)
	if err != nil {
		return err
	}
	targets := jobboard.EmployersDueForReverse(f, cfg.RescanDays, cfg.MaxPerRun)
	if len(targets) == 0 {
		slog.Info("employer reverse discover finished", "employers", 0, "tg_added", 0, "jobs_added", 0)
		return nil
	}

	var jobsAdded int
	var tgQueriesQueued int
	now := time.Now().UTC()
	for _, emp := range targets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		name := strings.TrimSpace(emp.Name)
		if name == "" {
			continue
		}
		for _, dork := range jobboard.ReverseDorks(name) {
			results, err := c.searchDork(ctx, dork)
			if err != nil {
				slog.Warn("employer reverse serp failed", "employer", name, "dork", dork, "error", err)
				continue
			}
			if err := appendTelegramChannelDiscoveries(cfg.TGChannelsPath, "employer:"+name, results); err != nil {
				slog.Warn("employer reverse tg registry write failed", "employer", name, "error", err)
			}
			items := ExtractJobboardDiscoveries(results)
			n, err := jobboard.AppendDiscoveries(cfg.JobboardRegistryPath, "employer:"+name, dork, items)
			if err != nil {
				slog.Warn("employer reverse job registry write failed", "employer", name, "error", err)
				continue
			}
			jobsAdded += n
		}
		key := strings.TrimSpace(emp.Normalized)
		if key == "" {
			key = entity.NormalizeCompany(name)
		}
		if err := jobboard.MarkEmployerReversed(cfg.EmployerRegistryPath, key, now); err != nil {
			slog.Warn("employer reverse stamp failed", "employer", name, "error", err)
		}
		n, err := jobboard.AppendEmployerTGQueries(cfg.EmployerTGQueriesPath, []string{name}, "employer_reverse")
		if err != nil {
			slog.Warn("employer tg query queue failed", "employer", name, "error", err)
		} else {
			tgQueriesQueued += n
		}
	}
	if jobsAdded > 0 {
		metrics.RecordSourcesDiscovered("jobboard", jobsAdded)
	}
	slog.Info("employer reverse discover finished",
		"employers", len(targets),
		"jobs_added", jobsAdded,
		"tg_queries_queued", tgQueriesQueued,
	)
	return nil
}

// EmployerReverseConfig controls employer -> TG/job reverse SERP harvest.
type EmployerReverseConfig struct {
	EmployerRegistryPath  string
	JobboardRegistryPath  string
	TGChannelsPath        string
	EmployerTGQueriesPath string
	MaxPerRun             int
	RescanDays            int
}
