package app

import (
	"testing"
	"time"

	"github.com/bidshard/parser/internal/config"
)

func TestCollectDrainTimeout(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		CollectDrainTimeout: 90 * time.Second,
		ShutdownTimeout:     30 * time.Second,
	}
	if got := collectDrainTimeout(cfg); got != 90*time.Second {
		t.Fatalf("got %v want 90s", got)
	}

	cfg = config.Config{ShutdownTimeout: 45 * time.Second}
	if got := collectDrainTimeout(cfg); got != 45*time.Second {
		t.Fatalf("got %v want 45s", got)
	}

	if got := collectDrainTimeout(config.Config{}); got != 120*time.Second {
		t.Fatalf("got %v want 120s default", got)
	}
}
