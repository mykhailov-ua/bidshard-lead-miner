package gemini

import "context"

func classifyJSON[T any](
	c *Client,
	ctx context.Context,
	priority Priority,
	systemPrompt, userPrompt string,
	schema map[string]any,
) (T, error) {
	var zero T
	raw, err := c.generateJSON(ctx, priority, systemPrompt, userPrompt, schema)
	if err != nil {
		return zero, err
	}
	var parsed T
	if err := decodeModelJSON(raw, &parsed); err != nil {
		return zero, err
	}
	return parsed, nil
}
