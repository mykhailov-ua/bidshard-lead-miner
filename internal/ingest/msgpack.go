package ingest

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"

	"github.com/bidshard/parser/internal/model"
	"github.com/bidshard/parser/internal/pipeline"
	"github.com/vmihailenco/msgpack/v5"
)

const maxMsgpackFrame = 4 << 20 // 4MB per Telethon message frame

// ScanMsgpack reads length-prefixed MessagePack frames from r (Telethon UDS IPC).
func ScanMsgpack(ctx context.Context, r io.Reader, taskCh chan<- pipeline.Task, stats *pipeline.RoundState, roundID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		frame, err := readMsgpackFrame(r)
		if err != nil {
			if err != io.EOF {
				slog.Debug("ingest msgpack stream end", "error", err)
			}
			return
		}
		item, err := parseMsgpackFrame(frame)
		if err != nil {
			slog.Warn("ingest skip bad msgpack", "error", err)
			continue
		}
		if err := validateTelegramItem(item); err != nil {
			slog.Warn("ingest skip invalid telegram item", "error", err, "source", item.Source)
			continue
		}
		emit(ctx, taskCh, stats, roundID, item)
	}
}

func readMsgpackFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint32(hdr[:]))
	if n <= 0 || n > maxMsgpackFrame {
		return nil, fmt.Errorf("invalid msgpack frame size %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func parseMsgpackFrame(frame []byte) (model.RawItem, error) {
	var item telegramItem
	if err := msgpack.Unmarshal(frame, &item); err != nil {
		return model.RawItem{}, err
	}
	return item.toRawItem(), nil
}
