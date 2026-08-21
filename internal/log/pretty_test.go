package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestPrettyHandlerWritesMultilineRecord(t *testing.T) {
	var buf bytes.Buffer
	h := newPrettyHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	h.color = false

	rec := slog.NewRecord(
		time.Now(),
		slog.LevelInfo,
		"lead accepted",
		0,
	)
	rec.AddAttrs(
		slog.String("round_id", "fb20d9"),
		slog.String("source", "tgweb:@wooden_blog"),
		slog.Int("score", 132),
		slog.String("matched", "[voluum alternative(+50)]"),
	)

	if err := h.Handle(t.Context(), rec); err != nil {
		t.Fatalf("handle: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"lead accepted",
		"round_id fb20d9",
		"source tgweb:@wooden_blog",
		"score 132",
		"matched",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestResolveFormatAuto(t *testing.T) {
	if got := resolveFormat("auto"); got != "pretty" && got != "json" {
		t.Fatalf("resolveFormat(auto) = %q, want pretty or json", got)
	}
	if got := resolveFormat("pretty"); got != "pretty" {
		t.Fatalf("resolveFormat(pretty) = %q", got)
	}
	if got := resolveFormat("json-pretty"); got != "json-pretty" {
		t.Fatalf("resolveFormat(json-pretty) = %q", got)
	}
}

func TestFormatValueQuotesSpaces(t *testing.T) {
	got := formatValue(slog.StringValue("hello world"))
	if got != `"hello world"` {
		t.Fatalf("formatValue = %q", got)
	}
}
