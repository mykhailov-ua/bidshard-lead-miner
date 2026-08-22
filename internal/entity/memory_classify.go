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
