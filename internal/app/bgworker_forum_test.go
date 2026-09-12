package app

import (
	"testing"
	"time"

	"github.com/bidshard/parser/internal/bgworker"
	"github.com/bidshard/parser/internal/config"
)

func forumCrawlJobIndex(jobs []bgworker.Job) int {
	for i, job := range jobs {
		if job.Name == "forum_crawl" {
			return i
		}
	}
	return -1
}

func TestForumCrawlBGJobConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		BGForumCrawlEnabled:  true,
		BGForumCrawlInterval: 3 * time.Hour,
	}
	if !cfg.BGForumCrawlEnabled {
		t.Fatal("expected forum crawl enabled by default wiring flag")
	}
	if cfg.BGForumCrawlInterval != 3*time.Hour {
		t.Fatalf("interval=%v", cfg.BGForumCrawlInterval)
	}
}
