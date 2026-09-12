package app

import (
	"fmt"
	"os"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/telethon"
)

// ValidateTelethonForRun fails fast when a run will start Telethon but session is missing.
func ValidateTelethonForRun(cfg config.Config) error {
	if !telethonRequiredForRun(cfg) {
		return nil
	}
	sessionPath := telethon.SessionPath(cfg.TelegramConfigPath)
	if _, err := os.Stat(sessionPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("telethon session missing: %s (run: parser telegram login --qr)", sessionPath)
		}
		return fmt.Errorf("telethon session %s: %w", sessionPath, err)
	}
	return nil
}

func telethonRequiredForRun(cfg config.Config) bool {
	if cfg.TelegramRealtime || cfg.TelegramSidecar {
		return cfg.TelegramAPIID != 0 && cfg.TelegramAPIHash != ""
	}
	if !cfg.BGTelegramEnabled || cfg.TelegramAPIID == 0 || cfg.TelegramAPIHash == "" {
		return false
	}
	// BG telegram jobs are not started on scan-once or when bg worker is off.
	if cfg.ScanOnce || !cfg.BGWorkerEnabled {
		return false
	}
	return true
}
