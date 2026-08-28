package config

import "testing"

func TestMXCheckDefaultTrue(t *testing.T) {
	t.Setenv("PARSER_MX_CHECK", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.MXCheck {
		t.Fatal("expected PARSER_MX_CHECK default true")
	}
}

func TestEnrichSMTPVerifyDefaultFalse(t *testing.T) {
	t.Setenv("PARSER_ENRICH_SMTP_VERIFY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EnrichSMTPVerify {
		t.Fatal("expected PARSER_ENRICH_SMTP_VERIFY default false")
	}
}

func TestMXCheckExplicitFalse(t *testing.T) {
	t.Setenv("PARSER_MX_CHECK", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MXCheck {
		t.Fatal("expected explicit PARSER_MX_CHECK=false")
	}
}
