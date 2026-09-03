package config

import (
	"testing"
	"time"
)

func TestWarmAnalysisPendingScanIntervalDefault(t *testing.T) {
	t.Setenv("WARM_ANALYSIS_PENDING_SCAN_INTERVAL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WarmAnalysisPendingScanInterval != 5*time.Minute {
		t.Fatalf("expected WARM_ANALYSIS_PENDING_SCAN_INTERVAL default 5m, got %s", cfg.WarmAnalysisPendingScanInterval)
	}
}

func TestWarmAnalysisPendingScanIntervalOverride(t *testing.T) {
	t.Setenv("WARM_ANALYSIS_PENDING_SCAN_INTERVAL", "15m")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WarmAnalysisPendingScanInterval != 15*time.Minute {
		t.Fatalf("expected WARM_ANALYSIS_PENDING_SCAN_INTERVAL override 15m, got %s", cfg.WarmAnalysisPendingScanInterval)
	}
}
