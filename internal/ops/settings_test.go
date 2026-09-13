package ops

import "testing"

func TestDefaultSettingsLLMEnabled(t *testing.T) {
	t.Parallel()
	if !defaultSettings().LLMScoringEnabled {
		t.Fatal("expected default llm scoring on")
	}
}

func TestCollectionOrDefault(t *testing.T) {
	t.Parallel()
	if collectionOrDefault("") != "crm_settings" {
		t.Fatal("default collection")
	}
	if collectionOrDefault("custom") != "custom" {
		t.Fatal("custom collection")
	}
}
