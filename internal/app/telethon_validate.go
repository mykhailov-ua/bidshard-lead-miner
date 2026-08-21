package app

import (
	"fmt"
	"os"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/telethon"
)

// ValidateTelethonForRun fails fast when telegram bg jobs are enabled but session is missing.
func ValidateTelethonForRun(cfg config.Config) error {
	if !cfg.BGTelegramEnabled {
		return nil
	}
	if cfg.TelegramAPIID == 0 || cfg.TelegramAPIHash == "" {
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
