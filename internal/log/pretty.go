package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bidshard/parser/internal/pretty"
)

// prettyHandler writes colorized, multi-line log records for interactive terminals.
type prettyHandler struct {
	w      io.Writer
	opts   slog.HandlerOptions
	attrs  []slog.Attr
	groups []string
	color  bool
	mu     sync.Mutex
}

func newPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *prettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &prettyHandler{w: w, opts: *opts, color: pretty.ColorEnabled(w)}
}

func (h *prettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	min := slog.LevelInfo
	if h.opts.Level != nil {
		min = h.opts.Level.Level()
	}
	return level >= min
}

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	if !h.Enabled(context.Background(), r.Level) {
		return nil
	}

	var b strings.Builder
	h.writeHeader(&b, r)
	h.writeAttrs(&b, h.attrs)
	r.Attrs(func(a slog.Attr) bool {
		h.writeAttr(&b, a)
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write([]byte(b.String()))
	return err
}

func (h *prettyHandler) writeHeader(b *strings.Builder, r slog.Record) {
	ts := r.Time.Local().Format("15:04:05")
	level := strings.ToUpper(r.Level.String())
	msg := r.Message

	if h.color {
		fmt.Fprintf(b, "%s%s%s  ", pretty.Dim, ts, pretty.Reset)
		fmt.Fprintf(b, "%s%s%s  ", levelStyle(r.Level), level, pretty.Reset)
		fmt.Fprintf(b, "%s%s%s\n", pretty.Bold, msg, pretty.Reset)
		return
	}
	fmt.Fprintf(b, "%s  %s  %s\n", ts, level, msg)
}

func levelStyle(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return pretty.Red + pretty.Bold
	case level >= slog.LevelWarn:
		return pretty.Yellow + pretty.Bold
	case level >= slog.LevelInfo:
		return pretty.Green + pretty.Bold
	default:
		return pretty.Cyan
	}
}

func (h *prettyHandler) writeAttrs(b *strings.Builder, attrs []slog.Attr) {
	for _, a := range attrs {
		h.writeAttr(b, a)
	}
}

func (h *prettyHandler) writeAttr(b *strings.Builder, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	if a.Value.Kind() == slog.KindGroup {
		for _, ga := range a.Value.Group() {
			if a.Key != "" {
				h2 := &prettyHandler{
					w:      h.w,
					opts:   h.opts,
					attrs:  h.attrs,
					groups: append(append([]string{}, h.groups...), a.Key),
					color:  h.color,
				}
				h2.writeAttr(b, ga)
				continue
			}
			h.writeAttr(b, ga)
		}
		return
	}

	key := a.Key
	if len(h.groups) > 0 {
		key = strings.Join(h.groups, ".") + "." + key
	}
	if key == "" {
		return
	}

	val := formatValue(a.Value)
	indent := "    "
	if h.color {
		fmt.Fprintf(b, "%s%s%s%s %s%s%s\n", indent, pretty.Cyan, key, pretty.Reset, pretty.Dim, val, pretty.Reset)
		return
	}
	fmt.Fprintf(b, "%s%s %s\n", indent, key, val)
}

func formatValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return pretty.QuoteIfNeeded(v.String())
	case slog.KindTime:
		return v.Time().UTC().Format(time.RFC3339)
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindGroup:
		var parts []string
		for _, a := range v.Group() {
			if a.Key == "" {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s=%s", a.Key, formatValue(a.Value)))
		}
		return "{" + strings.Join(parts, " ") + "}"
	default:
		return v.String()
	}
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &prettyHandler{
		w:      h.w,
		opts:   h.opts,
		attrs:  append(append([]slog.Attr{}, h.attrs...), attrs...),
		groups: h.groups,
		color:  h.color,
	}
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &prettyHandler{
		w:      h.w,
		opts:   h.opts,
		attrs:  h.attrs,
		groups: append(append([]string{}, h.groups...), name),
		color:  h.color,
	}
}
