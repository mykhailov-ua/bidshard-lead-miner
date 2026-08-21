package config_test

import (
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/crm/config"
)

func TestValidateForRun(t *testing.T) {
	cfg := config.Config{
		MongoURI:    "mongodb://localhost:27017",
		WebhookAddr: "127.0.0.1:8080",
	}
	if errs := cfg.ValidateForRun(); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	cfg.MongoURI = ""
	if errs := cfg.ValidateForRun(); len(errs) == 0 {
		t.Fatal("expected validation error")
	}
}

func TestMaskSecretInView(t *testing.T) {
	cfg := config.Config{WebhookSecret: "1234567890"}
	view := cfg.View()
	if view.WebhookSecret == "1234567890" {
		t.Fatalf("webhook_secret not masked: %q", view.WebhookSecret)
	}
}

func TestConfigViewWriteText(t *testing.T) {
	cfg := config.Config{
		WebhookSecret:       "secret-value",
		MongoURI:            "mongodb://localhost:27017",
		LeadNotesCollection: "lead_notes",
	}
	var buf strings.Builder
	if err := cfg.View().WriteText(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "secret-value") {
		t.Fatalf("webhook secret leaked in config show:\n%s", out)
	}
	if !strings.Contains(out, "lead_notes_collection=lead_notes") {
		t.Fatalf("missing notes collection in config show:\n%s", out)
	}
}
