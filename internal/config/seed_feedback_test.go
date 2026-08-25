package config

import "testing"

func TestParserSeedFeedbackEnv(t *testing.T) {
	t.Setenv("PARSER_SEED_FEEDBACK", "true")
	t.Setenv("PARSER_SEED_FEEDBACK_MIN_HEAT", "warm")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ParserSeedFeedback {
		t.Fatal("expected seed feedback enabled")
	}
	if cfg.ParserSeedFeedbackMinHeat != "warm" {
		t.Fatalf("min_heat=%q", cfg.ParserSeedFeedbackMinHeat)
	}
}
