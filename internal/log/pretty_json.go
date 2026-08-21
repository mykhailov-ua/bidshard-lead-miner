package log

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"time"
)

// prettyJSONHandler writes one indented JSON object per log record (human-readable terminal).
type prettyJSONHandler struct {
	w     io.Writer
	opts  slog.HandlerOptions
	attrs []slog.Attr
}

func newPrettyJSONHandler(w io.Writer, opts *slog.HandlerOptions) *prettyJSONHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &prettyJSONHandler{w: w, opts: *opts}
}

func (h *prettyJSONHandler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.opts.Level != nil {
		min = h.opts.Level.Level()
	}
	return level >= min
}

func (h *prettyJSONHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.Enabled(context.Background(), r.Level) {
		return nil
	}
	m := map[string]any{
		"time":  r.Time.UTC().Format(time.RFC3339Nano),
		"level": r.Level.String(),
		"msg":   r.Message,
	}
	for _, a := range h.attrs {
		m[a.Key] = a.Value.Any()
	}
	r.Attrs(func(a slog.Attr) bool {
		m[a.Key] = a.Value.Any()
		return true
	})
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if _, err := h.w.Write(b); err != nil {
		return err
	}
	_, err = h.w.Write([]byte("\n"))
	return err
}

func (h *prettyJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h2 := *h
	h2.attrs = append(h2.attrs, attrs...)
	return &h2
}

func (h *prettyJSONHandler) WithGroup(name string) slog.Handler {
	// Groups are flattened for terminal readability.
	_ = name
	return h
}
