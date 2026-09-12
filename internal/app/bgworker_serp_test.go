package app

import (
	"testing"
	"time"

	"github.com/bidshard/parser/internal/bgworker"
	"github.com/bidshard/parser/internal/config"
)

func TestSerpHarvestBGJobOrder(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		BGForumDiscoverInterval: time.Hour,
		BGSerpTelegramInterval:  time.Hour,
	}
	jobs := serpHarvestBackgroundJobs(cfg)
	if len(jobs) != len(serpHarvestBGJobOrder) {
		t.Fatalf("job count=%d want %d", len(jobs), len(serpHarvestBGJobOrder))
	}
	for i, want := range serpHarvestBGJobOrder {
		if jobs[i].Name != want {
			t.Fatalf("job[%d]=%q want %q", i, jobs[i].Name, want)
		}
	}

	jobboardIdx := indexBGJob(jobs, "serp_jobboard_urls")
	employerIdx := indexBGJob(jobs, "serp_employer_reverse")
	telegramIdx := indexBGJob(jobs, "serp_telegram_catalog")
	if jobboardIdx < 0 || employerIdx < 0 || telegramIdx < 0 {
		t.Fatalf("missing serp bg jobs")
	}
	if jobboardIdx >= employerIdx || employerIdx >= telegramIdx {
		t.Fatalf("jobboard must register before employer reverse and telegram catalog")
	}
}

func indexBGJob(jobs []bgworker.Job, name string) int {
	for i, job := range jobs {
		if job.Name == name {
			return i
		}
	}
	return -1
}
