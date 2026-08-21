package app

import (
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/entity"
)

func entityHeatFromConfig(cfg config.Config) entity.HeatConfig {
	return entity.HeatConfig{
		Enabled:          cfg.ParserEntityHeatEnabled,
		BlazingThreshold: cfg.EntityHeatBlazing,
		HotThreshold:     cfg.EntityHeatHot,
		WarmThreshold:    cfg.EntityHeatWarm,
		Decay7D:          cfg.EntityHeatDecay7D,
		Decay30D:         cfg.EntityHeatDecay30D,
		Decay90D:         cfg.EntityHeatDecay90D,
	}
}
