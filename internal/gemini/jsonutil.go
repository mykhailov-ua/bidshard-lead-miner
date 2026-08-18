package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// extractModelJSON trims Gemini output and strips optional markdown fences.
func extractModelJSON(raw []byte) []byte {
	s := strings.TrimSpace(string(raw))
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	s = strings.TrimSpace(s)
	return []byte(s)
}

func decodeModelJSON(raw []byte, v any) error {
	clean := extractModelJSON(raw)
	dec := json.NewDecoder(bytes.NewReader(clean))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode model json: %w (head=%q)", err, truncate(string(clean), 120))
	}
	return nil
}
