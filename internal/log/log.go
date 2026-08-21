package log

import (
	"log/slog"
	"os"

	"strings"

	"github.com/bidshard/parser/internal/pretty"
)

func Init(format, level string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	switch resolveFormat(format) {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	case "json-pretty":
		handler = newPrettyJSONHandler(os.Stderr, opts)
	case "pretty":
		handler = newPrettyHandler(os.Stderr, opts)
	default:
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func resolveFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "auto":
		if pretty.IsTerminal(os.Stderr) {
			return "pretty"
		}
		return "json"
	case "text", "json-pretty", "pretty", "json":
		return strings.ToLower(format)
	default:
		return strings.ToLower(format)
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
