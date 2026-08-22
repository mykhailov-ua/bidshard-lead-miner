package app

import (
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/entity"
)

func entityHeatFromConfig(cfg config.Config) entity.HeatConfig {
	return entity.HeatConfigFromThresholds(
		cfg.ParserEntityHeatEnabled,
		cfg.EntityHeatBlazing,
		cfg.EntityHeatHot,
		cfg.EntityHeatWarm,
		cfg.EntityHeatDecay7D,
		cfg.EntityHeatDecay30D,
		cfg.EntityHeatDecay90D,
	)
}
