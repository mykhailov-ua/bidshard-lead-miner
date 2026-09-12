package warmpath

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SpillRecord is one warm-path batch queued for Gemini Batch API submission.
type SpillRecord struct {
	Key         string  `json:"key"`
	GeoClassify bool    `json:"geo_classify"`
	Events      []Event `json:"events"`
}

// BatchSpill appends pending warm-path batches to a local JSONL file.
type BatchSpill struct {
	path string
	mu   sync.Mutex
}

func NewBatchSpill(path string) *BatchSpill {
	return &BatchSpill{path: path}
}

func (s *BatchSpill) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *BatchSpill) Append(rec SpillRecord) error {
	if s == nil {
		return fmt.Errorf("batch spill nil")
	}
	if rec.Key == "" || len(rec.Events) == 0 {
		return fmt.Errorf("batch spill: empty record")
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Drain reads and clears all pending spill records.
func (s *BatchSpill) Drain() ([]SpillRecord, error) {
	if s == nil {
		return nil, fmt.Errorf("batch spill nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []SpillRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec SpillRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("batch spill decode: %w", err)
		}
		if rec.Key != "" && len(rec.Events) > 0 {
			out = append(out, rec)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return out, nil
}
