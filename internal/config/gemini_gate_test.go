package config

import "testing"

func TestGeminiMisconfigErrors(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ParserICPClassifyTgWeb: true,
		GeminiAPIKey:           "",
	}
	errs := GeminiMisconfigErrors(cfg, true)
	if len(errs) != 1 {
		t.Fatalf("errs=%v want 1 tgweb error", errs)
	}

	errs = GeminiMisconfigErrors(cfg, false)
	if len(errs) != 0 {
		t.Fatalf("errs=%v want empty when tgweb inactive", errs)
	}

	cfg = Config{
		ParserICPClassify: true,
		GeminiAPIKey:      "",
	}
	errs = GeminiMisconfigErrors(cfg, false)
	if len(errs) != 1 {
		t.Fatalf("errs=%v want icp error", errs)
	}

	cfg.GeminiAPIKey = "set"
	if got := GeminiMisconfigErrors(cfg, true); len(got) != 0 {
		t.Fatalf("errs=%v want empty when key set", got)
	}
}
