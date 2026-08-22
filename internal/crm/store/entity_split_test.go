package store

import (
	"testing"

	"github.com/bidshard/parser/internal/entity"
)

func TestLeadStoreEntityHeatConfigUsesOptions(t *testing.T) {
	custom := entity.HeatConfigFromThresholds(true, 99, 80, 60, 1, 1, 1)
	s := &LeadStore{entityHeat: custom}
	got := s.entityHeatConfig()
	if got.WarmThreshold != 60 {
		t.Fatalf("warm=%v want 60", got.WarmThreshold)
	}
}

func TestLeadStoreEntityHeatConfigDefault(t *testing.T) {
	s := &LeadStore{}
	got := s.entityHeatConfig()
	def := entity.DefaultHeatConfig()
	if got.WarmThreshold != def.WarmThreshold {
		t.Fatalf("warm=%v want default %v", got.WarmThreshold, def.WarmThreshold)
	}
}
