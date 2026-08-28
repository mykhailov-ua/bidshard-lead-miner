package scoring

import (
	"encoding/json"
	"regexp"
	"strings"
)

const stackStructuredBoost = 25

var (
	nextDataScriptRe = regexp.MustCompile(`(?is)<script[^>]*id="__NEXT_DATA__"[^>]*>(.+?)</script>`)
	nextFlightPushRe = regexp.MustCompile(`(?:self\.)?__next_f\.push\(\[1,"((?:\\.|[^"])*)"`)
)

// CollectStack scans crawl HTML including Next.js __NEXT_DATA__ and RSC flight chunks.
// structured is true when a tracker fingerprint was found only via structured payloads.
func CollectStack(crawlHTML string) (stack []string, structured bool) {
	if crawlHTML == "" {
		return nil, false
	}
	stack = DetectCompetitorStack(crawlHTML)
	baseLen := len(stack)

	var payload strings.Builder
	if m := nextDataScriptRe.FindStringSubmatch(crawlHTML); len(m) >= 2 {
		payload.WriteString(flattenStackJSON(m[1]))
		payload.WriteByte(' ')
	}
	for _, m := range nextFlightPushRe.FindAllStringSubmatch(crawlHTML, -1) {
		if len(m) >= 2 {
			payload.WriteString(m[1])
			payload.WriteByte(' ')
		}
	}
	if payload.Len() == 0 {
		return stack, false
	}
	extra := DetectCompetitorStack(payload.String())
	stack = mergeStack(stack, extra)
	structured = len(extra) > 0 && baseLen == 0
	if len(extra) > 0 && baseLen > 0 {
		structured = true
	}
	return stack, structured && len(stack) > 0
}

func mergeStack(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	add := func(v string) {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, v := range base {
		add(v)
	}
	for _, v := range extra {
		add(v)
	}
	return out
}

func flattenStackJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !json.Valid([]byte(raw)) {
		return raw
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return raw
	}
	var parts []string
	collectJSONStrings(v, &parts)
	return strings.Join(parts, " ")
}

func collectJSONStrings(v any, parts *[]string) {
	switch t := v.(type) {
	case map[string]any:
		for _, val := range t {
			collectJSONStrings(val, parts)
		}
	case []any:
		for _, val := range t {
			collectJSONStrings(val, parts)
		}
	case string:
		if s := strings.TrimSpace(t); s != "" {
			*parts = append(*parts, s)
		}
	}
}
