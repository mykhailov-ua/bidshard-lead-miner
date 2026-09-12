package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/httpclient"
)

// NewOllamaClient wires local Ollama (Gemma, etc.) using the chat API + JSON schema format.
func NewOllamaClient(cfg config.Config) (*Client, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg.OllamaBaseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:11434"
	}
	model := strings.TrimSpace(cfg.OllamaModel)
	if model == "" {
		model = "gemma3:12b"
	}
	lc := LimitConfig{
		ModelLimits: ModelLimits{RPM: 600, TPM: 10_000_000, RPD: 100_000, MaxRetries: 2, RetryBase: time.Second, RetryMax: 10 * time.Second},
		EmbedRPM:    120,
		QuotaSplit:  DefaultQuotaSplit(),
	}
	timeout := cfg.GeminiRequestTimeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	cl := &Client{
		apiKey:           "ollama",
		model:            model,
		baseURL:          base,
		provider:         LLMProviderOllama,
		ollamaEmbedModel: strings.TrimSpace(cfg.OllamaEmbedModel),
		httpClient:       httpclient.ClientWithSharedTransport(timeout),
		limits:           lc.ModelLimits,
		limiter:          NewQuotaLimiter(lc),
		maxOutputTokens:  cfg.GeminiMaxOutputTokens,
	}
	if cl.maxOutputTokens <= 0 {
		cl.maxOutputTokens = 4096
	}
	if cl.ollamaEmbedModel == "" {
		cl.ollamaEmbedModel = "nomic-embed-text"
	}
	return cl, nil
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Format   map[string]any  `json:"format,omitempty"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	DoneReason string `json:"done_reason"`
}

func (c *Client) generateOllamaJSON(ctx context.Context, priority Priority, systemPrompt, userPrompt string, schema map[string]any) ([]byte, error) {
	reqBody := ollamaChatRequest{
		Model: c.model,
		Messages: []ollamaMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Format: schema,
		Stream: false,
	}
	if c.maxOutputTokens > 0 {
		reqBody.Options = map[string]any{"num_predict": c.maxOutputTokens}
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	url := c.baseURL + "/api/chat"
	estTokens := EstimateTokens(systemPrompt, userPrompt)
	respBody, err := c.postWithQuota(ctx, callGenerate, priority, url, raw, estTokens)
	if err != nil {
		return nil, err
	}
	var parsed ollamaChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	reason := strings.ToLower(strings.TrimSpace(parsed.DoneReason))
	if reason == "length" || reason == "max_tokens" {
		return nil, ErrOutputTruncated
	}
	text := strings.TrimSpace(parsed.Message.Content)
	if text == "" {
		return nil, fmt.Errorf("ollama: empty response")
	}
	return []byte(text), nil
}

func (c *Client) embedOllamaText(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(map[string]any{
		"model":  c.ollamaEmbedModel,
		"prompt": text,
	})
	if err != nil {
		return nil, err
	}
	url := c.baseURL + "/api/embeddings"
	respBody, err := c.postWithQuota(ctx, callEmbed, PriorityLow, url, body, EstimateTokens(text))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Embedding) == 0 {
		return nil, fmt.Errorf("ollama embed: empty vector")
	}
	out := make([]float32, len(parsed.Embedding))
	for i, v := range parsed.Embedding {
		out[i] = float32(v)
	}
	return out, nil
}

// postOllama bypasses Gemini URL shape; used only when provider=ollama.
