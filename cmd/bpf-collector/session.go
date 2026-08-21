package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type targetEntry struct {
	PID      uint32 `json:"pid"`
	CgroupID uint64 `json:"cgroup_id,omitempty"`
	Role     uint8  `json:"role"`
	Name     string `json:"name"`
}

type sessionMeta struct {
	StartedAt   time.Time     `json:"started_at"`
	EndedAt     *time.Time    `json:"ended_at,omitempty"`
	SampleRate  uint32        `json:"sample_rate"`
	RolesWanted string        `json:"roles_wanted,omitempty"`
	Targets     []targetEntry `json:"targets"`
}

type session struct {
	Dir  string
	Meta sessionMeta
}

func openSession(dir string) (*session, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	targetsPath := filepath.Join(dir, "targets.json")
	data, err := os.ReadFile(targetsPath)
	if err != nil {
		return nil, fmt.Errorf("read targets.json: %w", err)
	}
	var meta sessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse targets.json: %w", err)
	}
	return &session{Dir: dir, Meta: meta}, nil
}

func (s *session) markEnded() error {
	now := time.Now().UTC()
	s.Meta.EndedAt = &now
	return s.writeMeta()
}

func (s *session) writeMeta() error {
	path := filepath.Join(s.Dir, "session.json")
	data, err := json.MarshalIndent(s.Meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *session) Targets() []targetEntry {
	return s.Meta.Targets
}
