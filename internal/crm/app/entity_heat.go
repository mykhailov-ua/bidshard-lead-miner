package app

import (
	parsercfg "github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/entity"
)

func entityHeatFromParserConfig(cfg parsercfg.Config) entity.HeatConfig {
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
