package log

import (
	"log/slog"
	"os"
	"strings"
)

func Init(format, level string) {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	switch strings.ToLower(format) {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
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
