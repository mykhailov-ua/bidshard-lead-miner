package warmpath

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bidshard/parser/internal/gemini"
)

// EngageSpillItem pairs a lead event with its core batch result awaiting engage.
type EngageSpillItem struct {
	Event Event                  `json:"event"`
	Core  gemini.LeadBatchResult `json:"core"`
}

// EngageSpillRecord is one engage batch request (High-only chunk).
type EngageSpillRecord struct {
	Key   string            `json:"key"`
	Items []EngageSpillItem `json:"items"`
}

// EngageSpill queues High leads between core and engage batch jobs.
type EngageSpill struct {
	path string
	mu   sync.Mutex
}

func NewEngageSpill(path string) *EngageSpill {
	return &EngageSpill{path: path}
}

func (s *EngageSpill) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *EngageSpill) Append(rec EngageSpillRecord) error {
	if s == nil {
		return fmt.Errorf("engage spill nil")
	}
	if rec.Key == "" || len(rec.Items) == 0 {
		return fmt.Errorf("engage spill: empty record")
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

func (s *EngageSpill) Drain() ([]EngageSpillRecord, error) {
	if s == nil {
		return nil, fmt.Errorf("engage spill nil")
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

	var out []EngageSpillRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec EngageSpillRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("engage spill decode: %w", err)
		}
		if rec.Key != "" && len(rec.Items) > 0 {
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
