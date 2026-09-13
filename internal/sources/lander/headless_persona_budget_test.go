package lander

import (
	"path/filepath"
	"testing"
)

func TestPersonaDailyBudgetResetsPerDay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget.json")
	InitHeadlessPersonaBudget(2, path)
	if !AllowHeadlessPersona(0) {
		t.Fatal("expected allow")
	}
	RecordHeadlessPersona(0)
	RecordHeadlessPersona(0)
	if AllowHeadlessPersona(0) {
		t.Fatal("expected cap")
	}
	if !AllowHeadlessPersona(1) {
		t.Fatal("other proxy should allow")
	}
	ResetHeadlessPersonaBudgetForTest()
}
