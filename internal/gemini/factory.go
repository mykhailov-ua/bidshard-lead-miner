package gemini

import (
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/config"
)

const (
	LLMProviderGemini = "gemini"
	LLMProviderOllama = "ollama"
)

// NewFromConfig builds a Gemini API or local Ollama (Gemma) client from config.
func NewFromConfig(cfg config.Config) (*Client, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.LLMProvider))
	if provider == "" {
		provider = LLMProviderGemini
	}
	switch provider {
	case LLMProviderOllama:
		return NewOllamaClient(cfg)
	case LLMProviderGemini:
		if strings.TrimSpace(cfg.GeminiAPIKey) == "" {
			return nil, fmt.Errorf("gemini api key required (set GEMINI_API_KEY or LLM_PROVIDER=ollama)")
		}
		return NewClient(cfg.GeminiAPIKey, cfg.GeminiModel, ClientOptionsFrom(cfg)...)
	default:
		return nil, fmt.Errorf("unknown LLM_PROVIDER=%s (use gemini or ollama)", provider)
	}
}

// LLMConfigured reports whether deferred / classify features can run.
func LLMConfigured(cfg config.Config) bool {
	if strings.EqualFold(strings.TrimSpace(cfg.LLMProvider), LLMProviderOllama) {
		return strings.TrimSpace(cfg.OllamaBaseURL) != "" && strings.TrimSpace(cfg.OllamaModel) != ""
	}
	return strings.TrimSpace(cfg.GeminiAPIKey) != ""
}
