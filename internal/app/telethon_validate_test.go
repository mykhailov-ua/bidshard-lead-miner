package app

import (
	"testing"

	"github.com/bidshard/parser/internal/config"
)

func TestValidateTelethonForRunSkipsWhenDisabled(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		BGTelegramEnabled: false,
		TelegramAPIID:     1,
		TelegramAPIHash:   "x",
	}
	if err := ValidateTelethonForRun(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateTelethonForRunMissingSession(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		BGTelegramEnabled:  true,
		TelegramAPIID:      1,
		TelegramAPIHash:    "x",
		TelegramConfigPath: "config/sources.telegram.yaml",
	}
	if err := ValidateTelethonForRun(cfg); err == nil {
		t.Fatal("expected error for missing session")
	}
}
