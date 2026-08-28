package lander

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	nextDataRe       = regexp.MustCompile(`(?is)<script[^>]*id="__NEXT_DATA__"[^>]*>(.+?)</script>`)
	scriptBodyRe     = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)
	nextFlightPushRe = regexp.MustCompile(`(?:self\.)?__next_f\.push\(\[1,"((?:\\.|[^"])*)"`)
	rscStringRe      = regexp.MustCompile(`"((?:\\.|[^"\\]){3,})"`)
)

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

// HasNextFlightPayload reports App Router RSC flight payloads in raw HTML (pre-hydration).
func HasNextFlightPayload(html string) bool {
	return len(extractFlightScriptBodies(html)) > 0
}

// ExtractNextFlightText reads App Router RSC payloads from self.__next_f.push(...) script
// chunks embedded in SSR HTML - no browser hydration required.
// Best practice: scan <script> bodies (not the whole document) and extract string literals
// from React Flight wire format rather than fully deserializing the component tree.
func ExtractNextFlightText(html string) string {
	scripts := extractFlightScriptBodies(html)
	if len(scripts) == 0 {
		return ""
	}

	var parts []string
	for _, script := range scripts {
		for _, m := range nextFlightPushRe.FindAllStringSubmatch(script, -1) {
			if len(m) < 2 {
				continue
			}
			decoded, err := strconv.Unquote("\"" + m[1] + "\"")
			if err != nil {
				decoded = m[1]
			}
			if chunk := flattenFlightChunk(decoded); chunk != "" {
				parts = append(parts, chunk)
			}
		}
	}
	return strings.Join(parts, " ")
}

func extractFlightScriptBodies(html string) []string {
	var scripts []string
	for _, m := range scriptBodyRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		body := m[1]
		if strings.Contains(body, "__next_f") {
			scripts = append(scripts, body)
		}
	}
	return scripts
}

func flattenFlightChunk(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, ":"); idx >= 0 {
		payload := strings.TrimSpace(raw[idx+1:])
		if json.Valid([]byte(payload)) {
			return flattenJSON(payload)
		}
		raw = payload
	}
	return collectRSCStrings(raw)
}

func collectRSCStrings(raw string) string {
	var parts []string
	for _, m := range rscStringRe.FindAllStringSubmatch(raw, -1) {
		if len(m) < 2 {
			continue
		}
		s, err := strconv.Unquote("\"" + m[1] + "\"")
		if err != nil {
			s = m[1]
		}
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "$") || strings.HasPrefix(s, "@") {
			continue
		}
		if looksLikeRSCInternal(s) || skipExtractString(s) {
			continue
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

func looksLikeRSCInternal(s string) bool {
	if strings.Contains(s, " ") {
		return false
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return false
	}
	if strings.Contains(s, "/") && !strings.Contains(s, " ") {
		return true
	}
	if len(s) <= 3 {
		return true
	}
	return false
}

func DiagnoseExtract(content string) (text string, method string) {
	if data, err := ExtractNextDataJSON(content); err == nil {
		return flattenJSON(string(data)), "next_data"
	}
	if flight := ExtractNextFlightText(content); flight != "" {
		return flight, "next_flight_script"
	}
	if wire := ExtractFlightWireText(content); wire != "" {
		return wire, "rsc_wire"
	}
	meta := extractMetaFallback(content)
	if meta != "" {
		if static := ExtractStaticLandingText(content); static != "" {
			return strings.TrimSpace(meta + " " + static), "meta+static"
		}
		return meta, "meta"
	}
	if static := ExtractStaticLandingText(content); static != "" {
		return static, "static_html"
	}
	return "", "empty"
}

func ExtractPageText(html string) (string, error) {
	text, _ := DiagnoseExtract(html)
	return text, nil
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
		s := strings.TrimSpace(t)
		// skipExtractString drops CDN/asset literals that inflate keyword prescan.
		if s != "" && !skipExtractString(s) {
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
