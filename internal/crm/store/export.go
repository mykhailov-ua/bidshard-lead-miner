package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bidshard/parser/internal/sink"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const defaultExportMaxRows = 500

type ExportFilter struct {
	Status string
	Since  time.Time
	Limit  int64
}

type ExportResult struct {
	Path string
	Rows int
}

func ParseSinceDuration(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("since duration empty")
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d, nil
	}
	if strings.HasSuffix(raw, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(raw, "d"))
		if err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid since duration %q", raw)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("invalid since duration %q (use 24h, 7d)", raw)
}

func (s *LeadStore) BuildNDJSON(ctx context.Context, filter ExportFilter) (ExportResult, error) {
	if s == nil || s.leads == nil {
		return ExportResult{}, fmt.Errorf("lead store not initialized")
	}

	limit := filter.Limit
	if limit <= 0 || limit > s.exportMaxRows {
		limit = s.exportMaxRows
	}

	query := bson.M{}
	if status := strings.TrimSpace(filter.Status); status != "" {
		normalized, err := NormalizeStatus(status)
		if err != nil {
			return ExportResult{}, err
		}
		query["status"] = normalized
	}
	if !filter.Since.IsZero() {
		query["ts"] = bson.M{"$gte": filter.Since.UTC()}
	}

	queryCtx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	cur, err := s.leads.Find(queryCtx, query,
		options.Find().
			SetSort(bson.D{{Key: "score", Value: -1}, {Key: "hash_id", Value: 1}}).
			SetLimit(limit),
	)
	if err != nil {
		return ExportResult{}, err
	}
	defer func() { _ = cur.Close(queryCtx) }()

	file, err := os.CreateTemp("", "crm-export-*.ndjson")
	if err != nil {
		return ExportResult{}, fmt.Errorf("create temp export file: %w", err)
	}
	path := file.Name()

	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	rows := 0
	for cur.Next(queryCtx) {
		var doc sink.LeadDoc
		if err := cur.Decode(&doc); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return ExportResult{}, err
		}
		if err := enc.Encode(doc); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return ExportResult{}, err
		}
		rows++
	}
	if err := cur.Err(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return ExportResult{}, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return ExportResult{}, err
	}
	return ExportResult{Path: path, Rows: rows}, nil
}
