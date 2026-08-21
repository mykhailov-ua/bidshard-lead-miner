package entity

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory Recorder for unit tests.
type MemoryStore struct {
	mu                sync.Mutex
	docs              map[string]*EntityDoc
	CrossSourceWindow time.Duration
	HeatConfig        HeatConfig
}

func (s *MemoryStore) heatConfig() HeatConfig {
	if s == nil {
		return HeatConfig{Enabled: false}
	}
	if s.HeatConfig == (HeatConfig{}) {
		return DefaultHeatConfig()
	}
	return s.HeatConfig
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{docs: make(map[string]*EntityDoc)}
}

func (s *MemoryStore) RecordSighting(ctx context.Context, in SightingInput) (RecordResult, error) {
	if s == nil {
		return RecordResult{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return RecordSightingCore(ctx, in, s.CrossSourceWindow, DefaultMergeMaxAttempts, s.heatConfig(), lockedMemoryBackend{s})
}

type lockedMemoryBackend struct {
	s *MemoryStore
}

func (b lockedMemoryBackend) FindExisting(_ context.Context, aliases []string, candidateID string) (*EntityDoc, error) {
	return b.s.findLocked(aliases, candidateID), nil
}

func (b lockedMemoryBackend) InsertNew(_ context.Context, doc EntityDoc) (duplicate bool, err error) {
	if b.s.findLocked(doc.AliasKeys, doc.EntityID) != nil {
		return true, nil
	}
	if _, ok := b.s.docs[doc.EntityID]; ok {
		return true, nil
	}
	stored := doc
	b.s.docs[doc.EntityID] = &stored
	return false, nil
}

func (b lockedMemoryBackend) ReplaceMerged(_ context.Context, doc EntityDoc, generation int) (bool, error) {
	existing, ok := b.s.docs[doc.EntityID]
	if !ok || existing == nil {
		return false, nil
	}
	// MergeSighting mutates *existing in place; generation advances by one per merge.
	return existing.MergeGeneration == generation+1, nil
}

func (s *MemoryStore) Get(entityID string) (EntityDoc, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[entityID]
	if !ok || doc == nil {
		return EntityDoc{}, false
	}
	return *doc, true
}

func (s *MemoryStore) findLocked(aliases []string, candidateID string) *EntityDoc {
	if doc, ok := s.docs[candidateID]; ok && doc != nil {
		return doc
	}
	for _, doc := range s.docs {
		if doc == nil {
			continue
		}
		for _, alias := range aliases {
			if containsString(doc.AliasKeys, alias) {
				return doc
			}
		}
	}
	return nil
}
