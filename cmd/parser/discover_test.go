package main

import (
	"context"
	"errors"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/serp"
)

type discoverSerpHarvestRecorder struct {
	calls []string
}

func (r *discoverSerpHarvestRecorder) HarvestJobboardURLs(context.Context, string) error {
	r.calls = append(r.calls, "jobboard")
	return nil
}

func (r *discoverSerpHarvestRecorder) HarvestForumThreads(context.Context, string) error {
	r.calls = append(r.calls, "forum")
	return nil
}

func (r *discoverSerpHarvestRecorder) HarvestEmployerReverse(context.Context, serp.EmployerReverseConfig) error {
	r.calls = append(r.calls, "employer_reverse")
	return nil
}

func (r *discoverSerpHarvestRecorder) HarvestTGCatalogSources(context.Context) error {
	r.calls = append(r.calls, "tg_catalog_meta")
	return nil
}

func (r *discoverSerpHarvestRecorder) CrawlTGCatalogPages(context.Context, int) error {
	r.calls = append(r.calls, "tg_catalog_crawl")
	return nil
}

func (r *discoverSerpHarvestRecorder) HarvestTelegramCatalog(context.Context) error {
	r.calls = append(r.calls, "telegram")
	return nil
}

func TestDiscoverSerpHarvestOrder(t *testing.T) {
	t.Parallel()

	rec := &discoverSerpHarvestRecorder{}
	cfg := config.Config{
		JobboardRegistryPath: "/tmp/jobboard.json",
		ForumRegistryPath:    "/tmp/forum.json",
	}

	if err := runDiscoverSerpHarvest(context.Background(), cfg, rec); err != nil {
		t.Fatalf("runDiscoverSerpHarvest: %v", err)
	}

	want := []string{"jobboard", "forum", "employer_reverse", "tg_catalog_meta", "tg_catalog_crawl", "telegram"}
	if len(rec.calls) != len(want) {
		t.Fatalf("calls=%v want %v", rec.calls, want)
	}
	for i, name := range want {
		if rec.calls[i] != name {
			t.Fatalf("call[%d]=%q want %q (full=%v)", i, rec.calls[i], name, rec.calls)
		}
	}

	jobboardIdx := indexOf(rec.calls, "jobboard")
	telegramIdx := indexOf(rec.calls, "telegram")
	if jobboardIdx < 0 || telegramIdx < 0 || jobboardIdx >= telegramIdx {
		t.Fatalf("jobboard must run before telegram: calls=%v", rec.calls)
	}
}

func indexOf(items []string, target string) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}
	return -1
}

func TestDiscoverSerpHarvestStopsOnError(t *testing.T) {
	t.Parallel()

	rec := &discoverSerpHarvestRecorder{}
	cfg := config.Config{JobboardRegistryPath: "/tmp/jobboard.json"}

	err := runDiscoverSerpHarvest(context.Background(), cfg, &failingDiscoverHarvester{
		rec:   rec,
		failAt: "forum",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(rec.calls) != 2 || rec.calls[0] != "jobboard" || rec.calls[1] != "forum" {
		t.Fatalf("calls=%v want [jobboard forum] before stop", rec.calls)
	}
}

type failingDiscoverHarvester struct {
	rec    *discoverSerpHarvestRecorder
	failAt string
}

func (f *failingDiscoverHarvester) HarvestJobboardURLs(ctx context.Context, registryPath string) error {
	return f.rec.HarvestJobboardURLs(ctx, registryPath)
}

func (f *failingDiscoverHarvester) HarvestForumThreads(ctx context.Context, registryPath string) error {
	_ = f.rec.HarvestForumThreads(ctx, registryPath)
	return errors.New("forum harvest failed")
}

func (f *failingDiscoverHarvester) HarvestEmployerReverse(context.Context, serp.EmployerReverseConfig) error {
	return nil
}

func (f *failingDiscoverHarvester) HarvestTGCatalogSources(context.Context) error {
	return nil
}

func (f *failingDiscoverHarvester) CrawlTGCatalogPages(context.Context, int) error {
	return nil
}

func (f *failingDiscoverHarvester) HarvestTelegramCatalog(context.Context) error {
	return nil
}
