package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
)

type embedRequest struct {
	Model   string     `json:"model"`
	Content embedContent `json:"content"`
}

type embedContent struct {
	Parts []part `json:"parts"`
}

type embedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
	Error *apiError `json:"error"`
}

const defaultEmbedModel = "text-embedding-004"

func (c *Client) EmbedText(ctx context.Context, text string) ([]float32, error) {
	body := embedRequest{
		Model:   "models/" + defaultEmbedModel,
		Content: embedContent{Parts: []part{{Text: truncate(text, 1500)}}},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", c.baseURL, defaultEmbedModel, c.apiKey)
	respBody, err := c.postWithQuota(ctx, callEmbed, url, raw, EstimateTokens(text))
	if err != nil {
		return nil, err
	}
	var parsed embedResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini embed: %s", parsed.Error.Message)
	}
	if len(parsed.Embedding.Values) == 0 {
		return nil, fmt.Errorf("gemini embed: empty vector")
	}
	return parsed.Embedding.Values, nil
}

func CosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		af := float64(a[i])
		bf := float64(b[i])
		dot += af * bf
		na += af * af
		nb += bf * bf
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
