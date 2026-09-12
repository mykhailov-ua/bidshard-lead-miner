package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrOutputTruncated is returned when Gemini stops with finishReason MAX_TOKENS (truncated JSON).
var ErrOutputTruncated = errors.New("gemini: output truncated (MAX_TOKENS)")

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type Client struct {
	apiKey           string
	model            string
	baseURL          string
	provider         string
	ollamaEmbedModel string
	httpClient       *http.Client
	limits           ModelLimits
	limiter          *QuotaLimiter
	maxOutputTokens  int
}

type Option func(*Client)

func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.httpClient = c }
}

func WithBaseURL(url string) Option {
	return func(cl *Client) { cl.baseURL = strings.TrimRight(url, "/") }
}

func WithLimitConfig(cfg LimitConfig) Option {
	return func(cl *Client) {
		cl.limits = cfg.withOverrides(0, 0, 0, 0)
		cl.limiter = NewQuotaLimiter(cfg)
	}
}

// WithMaxOutputTokens sets generateContent maxOutputTokens (0 = API default).
func WithMaxOutputTokens(n int) Option {
	return func(cl *Client) { cl.maxOutputTokens = n }
}

func NewClient(apiKey, model string, opts ...Option) (*Client, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key required")
	}
	if model == "" {
		model = DefaultModel
	}
	lc := DefaultLimitConfig(model)
	cl := &Client{
		apiKey:     apiKey,
		model:      model,
		baseURL:    defaultBaseURL,
		provider:   LLMProviderGemini,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		limits:     lc.ModelLimits,
		limiter:    NewQuotaLimiter(lc),
	}
	for _, opt := range opts {
		opt(cl)
	}
	return cl, nil
}

func (c *Client) Model() string {
	if c == nil {
		return ""
	}
	return c.model
}

func (c *Client) Limits() ModelLimits {
	if c == nil {
		return ModelLimits{}
	}
	return c.limits
}

func (c *Client) QuotaStats() QuotaStats {
	if c == nil || c.limiter == nil {
		return QuotaStats{}
	}
	return c.limiter.Stats()
}

type generateRequest struct {
	Contents          []content        `json:"contents"`
	SystemInstruction *content         `json:"systemInstruction,omitempty"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMIMEType string         `json:"responseMimeType"`
	ResponseSchema   map[string]any `json:"responseSchema"`
	MaxOutputTokens  int            `json:"maxOutputTokens,omitempty"`
}

type generateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	Error *apiError `json:"error"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

func (c *Client) generateJSON(ctx context.Context, priority Priority, systemPrompt, userPrompt string, schema map[string]any) ([]byte, error) {
	if c != nil && c.provider == LLMProviderOllama {
		return c.generateOllamaJSON(ctx, priority, systemPrompt, userPrompt, schema)
	}
	genCfg := generationConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}
	if c.maxOutputTokens > 0 {
		genCfg.MaxOutputTokens = c.maxOutputTokens
	}
	body := generateRequest{
		Contents:         []content{{Parts: []part{{Text: userPrompt}}}},
		GenerationConfig: genCfg,
	}
	if systemPrompt != "" {
		body.SystemInstruction = &content{Parts: []part{{Text: systemPrompt}}}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	estTokens := EstimateTokens(systemPrompt, userPrompt)
	respBody, err := c.postWithQuota(ctx, callGenerate, priority, url, raw, estTokens)
	if err != nil {
		return nil, err
	}

	var parsed generateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != nil {
		return nil, &APIError{Code: parsed.Error.Code, Status: parsed.Error.Status, Message: parsed.Error.Message}
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini: empty response")
	}
	cand := parsed.Candidates[0]
	if reason := strings.ToUpper(strings.TrimSpace(cand.FinishReason)); reason == "MAX_TOKENS" {
		return nil, ErrOutputTruncated
	}
	text := strings.TrimSpace(cand.Content.Parts[0].Text)
	if text == "" {
		return nil, fmt.Errorf("gemini: empty text")
	}
	return []byte(text), nil
}
