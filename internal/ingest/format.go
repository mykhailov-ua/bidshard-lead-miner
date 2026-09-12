package ingest

import (
	"context"
	"io"
	"strings"

	"github.com/bidshard/parser/internal/pipeline"
)

// Format selects Telethon sidecar wire encoding.
type Format string

const (
	FormatNDJSON  Format = "ndjson"
	FormatMsgpack Format = "msgpack"
)

// ParseFormat normalizes TELETHON_IPC_FORMAT; defaults to ndjson.
func ParseFormat(raw string) Format {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "msgpack", "messagepack":
		return FormatMsgpack
	default:
		return FormatNDJSON
	}
}

// ScanFormat dispatches to NDJSON lines or length-prefixed MessagePack frames.
func ScanFormat(ctx context.Context, r io.Reader, format Format, taskCh chan<- pipeline.Task, stats *pipeline.RoundState, roundID string) {
	switch format {
	case FormatMsgpack:
		ScanMsgpack(ctx, r, taskCh, stats, roundID)
	default:
		Scan(ctx, r, taskCh, stats, roundID)
	}
}
