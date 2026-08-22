package sink

import (
	"testing"

	"github.com/bidshard/parser/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
)

func TestLinkedLeadsFilter(t *testing.T) {
	filter := linkedLeadsFilter("ent-1", []string{"hash-a", "hash-b"})
	if filter == nil {
		t.Fatal("expected filter")
	}
	orClause, ok := filter["$or"].(bson.A)
	if !ok || len(orClause) != 2 {
		t.Fatalf("filter=%v", filter)
	}
}

func TestLinkedLeadsFilterEntityOnly(t *testing.T) {
	filter := linkedLeadsFilter("ent-1", nil)
	if filter["entity_id"] != "ent-1" {
		t.Fatalf("filter=%v", filter)
	}
}

func TestClassificationPatchBSONOmitsHeatUnlessDowngrade(t *testing.T) {
	patch := entity.EntityClassificationPatch{UnifiedPain: "pain"}
	set := classificationPatchBSON(patch)
	if _, ok := set["heat_tier"]; ok {
		t.Fatal("unexpected heat_tier")
	}
	patch.HeatTierDowngrade = entity.HeatTierWarm
	set = classificationPatchBSON(patch)
	if set["heat_tier"] != entity.HeatTierWarm {
		t.Fatalf("heat_tier=%v", set["heat_tier"])
	}
}
