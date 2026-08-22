package entity

// HeatConfigFromThresholds builds runtime heat config from parser env thresholds.
func HeatConfigFromThresholds(enabled bool, blazing, hot, warm, decay7D, decay30D, decay90D float64) HeatConfig {
	return HeatConfig{
		Enabled:          enabled,
		BlazingThreshold: blazing,
		HotThreshold:     hot,
		WarmThreshold:    warm,
		Decay7D:          decay7D,
		Decay30D:         decay30D,
		Decay90D:         decay90D,
	}
}
