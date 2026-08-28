package config

import "fmt"

// GeminiFlag describes an env flag that requires GEMINI_API_KEY when enabled.
type GeminiFlag struct {
	EnvName string
	Enabled bool
}

// EnabledGeminiFlags returns Gemini-dependent toggles that are on in cfg.
func EnabledGeminiFlags(cfg Config) []GeminiFlag {
	flags := []GeminiFlag{
		{"PARSER_ICP_CLASSIFY", cfg.ParserICPClassify},
		{"PARSER_ICP_CLASSIFY_TGWEB", cfg.ParserICPClassifyTgWeb},
		{"PARSER_INTENT_CLASSIFY", cfg.ParserIntentClassify},
		{"PARSER_GEO_CLASSIFY", cfg.ParserGeoClassify},
		{"PARSER_GEMINI_ENGAGE", cfg.ParserGeminiEngage},
		{"PARSER_EMBED_PRESCAN", cfg.ParserEmbedPrescan},
		{"PARSER_EMBED_CLUSTER", cfg.ParserEmbedCluster},
		{"PARSER_GEMINI_ENRICH_SYNTH", cfg.ParserGeminiEnrichSynth},
	}
	var out []GeminiFlag
	for _, f := range flags {
		if f.Enabled {
			out = append(out, f)
		}
	}
	return out
}

// GeminiMisconfigErrors lists hard config errors when Gemini flags are on without a key.
func GeminiMisconfigErrors(cfg Config, tgwebSourceActive bool) []string {
	if cfg.GeminiAPIKey != "" {
		return nil
	}
	var errs []string
	for _, f := range EnabledGeminiFlags(cfg) {
		// PARSER_ICP_CLASSIFY_TGWEB defaults true but only matters when tgweb is in PARSER_SOURCE.
		if f.EnvName == "PARSER_ICP_CLASSIFY_TGWEB" && !tgwebSourceActive {
			continue
		}
		errs = append(errs, fmt.Sprintf("GEMINI_API_KEY required when %s=true", f.EnvName))
	}
	return errs
}
