package serp

import (
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/jobboard"
)

// EmployerReverseConfigFrom builds employer reverse settings from app config.
func EmployerReverseConfigFrom(cfg config.Config) EmployerReverseConfig {
	tgChannels := cfg.TelegramChannelsPath
	if tgChannels == "" {
		tgChannels = defaultTGChannelsPath
	}
	return EmployerReverseConfig{
		EmployerRegistryPath:  cfg.EmployerRegistryPath,
		JobboardRegistryPath:  cfg.JobboardRegistryPath,
		TGChannelsPath:        tgChannels,
		EmployerTGQueriesPath: jobboard.DefaultEmployerTGQueriesPath,
		MaxPerRun:             cfg.EmployerReverseMaxPerRun,
		RescanDays:            cfg.EmployerReverseRescanDays,
	}
}
