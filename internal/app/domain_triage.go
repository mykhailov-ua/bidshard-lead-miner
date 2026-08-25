package app

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bidshard/parser/internal/gemini"
	"github.com/bidshard/parser/internal/metrics"
	"github.com/bidshard/parser/internal/sourceregistry"
)

// DomainTriageConfig controls Gemini/heuristic domain filter before HTTP crawl.
type DomainTriageConfig struct {
	RegistryPath string
	CachePath    string
	BatchSize    int
}

type domainTriageClient interface {
	TriageDomains(ctx context.Context, items []gemini.DomainTriageInput) ([]gemini.DomainTriageResult, error)
}

// RunDomainTriage classifies registry domains and writes triage_status before crawl jobs run.
func RunDomainTriage(ctx context.Context, cfg DomainTriageConfig, client domainTriageClient) error {
	if client == nil {
		return nil
	}
	if cfg.RegistryPath == "" {
		cfg.RegistryPath = sourceregistry.DefaultPath
	}
	if cfg.CachePath == "" {
		cfg.CachePath = "data/runtime/domain_triage_cache.json"
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 20
	}

	f, err := sourceregistry.Load(cfg.RegistryPath)
	if err != nil {
		return err
	}
	cache := sourceregistry.ReadTriageCache(cfg.CachePath)
	dropped := 0

	var geminiQueue []gemini.DomainTriageInput
	for i := range f.Sources {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry := &f.Sources[i]
		domain := sourceregistry.NormalizeDomain(entry.Domain)
		if domain == "" {
			continue
		}
		meta := sourceregistry.DomainMeta{
			Domain:       domain,
			Channel:      entry.Channel,
			Source:       entry.Source,
			DiscoveredBy: entry.DiscoveredBy,
		}
		if action, _, ok := sourceregistry.HeuristicTriage(meta); ok {
			prev := strings.ToLower(strings.TrimSpace(entry.TriageStatus))
			sourceregistry.SetEntryTriageStatus(entry, action)
			cache.Decisions[domain] = action
			if action == "drop" && prev != "drop" {
				dropped++
			}
			continue
		}
		if cached, ok := cache.Decisions[domain]; ok && cached != "" && cached != "defer" {
			prev := strings.ToLower(strings.TrimSpace(entry.TriageStatus))
			sourceregistry.SetEntryTriageStatus(entry, cached)
			if cached == "drop" && prev != "drop" {
				dropped++
			}
			continue
		}
		status := strings.ToLower(strings.TrimSpace(entry.TriageStatus))
		if status == "keep" || status == "drop" {
			continue
		}
		geminiQueue = append(geminiQueue, gemini.DomainTriageInput{
			ID:           domain,
			Domain:       domain,
			Channel:      entry.Channel,
			Source:       entry.Source,
			DiscoveredBy: entry.DiscoveredBy,
		})
	}

	if err := sourceregistry.Save(cfg.RegistryPath, f); err != nil {
		return err
	}

	for start := 0; start < len(geminiQueue); start += cfg.BatchSize {
		end := start + cfg.BatchSize
		if end > len(geminiQueue) {
			end = len(geminiQueue)
		}
		results, err := client.TriageDomains(ctx, geminiQueue[start:end])
		if err != nil {
			slog.Warn("domain triage gemini failed", "error", err)
			continue
		}
		for _, res := range results {
			if res.ID == "" {
				continue
			}
			cache.Decisions[sourceregistry.NormalizeDomain(res.ID)] = res.Action
		}
	}

	added, err := sourceregistry.ApplyTriageDecisions(cfg.RegistryPath, cache)
	if err != nil {
		return err
	}
	dropped += added
	if err := sourceregistry.WriteTriageCache(cfg.CachePath, cache); err != nil {
		return err
	}
	if dropped > 0 {
		metrics.RecordSourcesTriagedDropped(dropped)
		slog.Info("domain triage complete", "dropped", dropped)
	}
	return nil
}
