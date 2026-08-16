package lander

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var nextDataRe = regexp.MustCompile(`(?is)<script[^>]*id="__NEXT_DATA__"[^>]*>(.+?)</script>`)

func ExtractNextDataJSON(html string) ([]byte, error) {
	match := nextDataRe.FindStringSubmatch(html)
	if len(match) < 2 {
		return nil, fmt.Errorf("next data not found")
	}
	payload := strings.TrimSpace(match[1])
	if !json.Valid([]byte(payload)) {
		return nil, fmt.Errorf("invalid next data json")
	}
	return []byte(payload), nil
}

func ExtractPageText(html string) (string, error) {
	if data, err := ExtractNextDataJSON(html); err == nil {
		return flattenJSON(string(data)), nil
	}
	return extractMetaFallback(html), nil
}

func flattenJSON(raw string) string {
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	var parts []string
	collectJSONValues(v, &parts)
	return strings.Join(parts, " ")
}

func collectJSONValues(v any, parts *[]string) {
	switch t := v.(type) {
	case map[string]any:
		for _, val := range t {
			collectJSONValues(val, parts)
		}
	case []any:
		for _, val := range t {
			collectJSONValues(val, parts)
		}
	case string:
		if s := strings.TrimSpace(t); s != "" {
			*parts = append(*parts, s)
		}
	}
}

func extractMetaFallback(html string) string {
	var parts []string
	titleRe := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	descRe := regexp.MustCompile(`(?is)<meta[^>]+name="description"[^>]+content="([^"]+)"`)
	if m := titleRe.FindStringSubmatch(html); len(m) > 1 {
		parts = append(parts, stripTags(m[1]))
	}
	if m := descRe.FindStringSubmatch(html); len(m) > 1 {
		parts = append(parts, stripTags(m[1]))
	}
	return strings.Join(parts, " ")
}

func stripTags(s string) string {
	s = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}
