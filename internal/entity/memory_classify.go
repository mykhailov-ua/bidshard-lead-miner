package entity

import "context"

func (s *MemoryStore) GetEntity(_ context.Context, entityID string) (EntityDoc, bool, error) {
	doc, ok := s.Get(entityID)
	return doc, ok, nil
}

func (s *MemoryStore) PatchEntityClassification(_ context.Context, entityID string, patch EntityClassificationPatch) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[entityID]
	if !ok || doc == nil {
		return nil
	}
	doc.UnifiedPain = patch.UnifiedPain
	doc.ActorConfidence = patch.ActorConfidence
	doc.BuyerIntent = patch.BuyerIntent
	doc.NeedsReview = patch.NeedsReview
	doc.EntityClassifiedAt = patch.EntityClassifiedAt
	if patch.HeatTierDowngrade != "" {
		doc.HeatTier = patch.HeatTierDowngrade
	}
	return nil
}

func (s *MemoryStore) MarkClassifyForce(_ context.Context, entityID string) error {
	if s == nil || entityID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[entityID]
	if !ok || doc == nil {
		return nil
	}
	doc.ClassifyForce = true
	return nil
}

func (s *MemoryStore) ListClassifyForce(_ context.Context, limit int) ([]string, error) {
	if s == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 32
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []string
	for id, doc := range s.docs {
		if doc != nil && doc.ClassifyForce {
			ids = append(ids, id)
			if len(ids) >= limit {
				break
			}
		}
	}
	return ids, nil
}

func (s *MemoryStore) ClearClassifyForce(_ context.Context, entityID string) error {
	if s == nil || entityID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.docs[entityID]
	if !ok || doc == nil {
		return nil
	}
	doc.ClassifyForce = false
	return nil
}
