package gemini

import (
	"context"
	"testing"
	"time"
)

func TestLimitsForModelFlash(t *testing.T) {
	l := LimitsForModel("gemini-2.5-flash")
	if l.RPM != 20 || l.RPD != 250 {
		t.Fatalf("unexpected 2.5-flash limits: %+v", l)
	}
	l36 := LimitsForModel("gemini-3.6-flash")
	if l36.RPM != 15 || l36.RPD != 1500 {
		t.Fatalf("unexpected 3.6-flash limits: %+v", l36)
	}
}

func TestQuotaLimiterRPM(t *testing.T) {
	q := NewQuotaLimiter(LimitConfig{
		ModelLimits: ModelLimits{RPM: 60, TPM: 1_000_000, RPD: 1000},
		QuotaSplit:  QuotaSplit{Critical: 25, High: 25, Normal: 25, Low: 25},
	})
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := q.WaitGenerate(ctx, PriorityNormal, 10); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		t.Fatalf("expected rpm throttle, elapsed=%v", elapsed)
	}
}

func TestQuotaSplitRPM(t *testing.T) {
	split := DefaultQuotaSplit()
	if split.RPM(20, PriorityCritical) != 4 {
		t.Fatalf("critical rpm=%d want 4", split.RPM(20, PriorityCritical))
	}
	if split.RPM(20, PriorityHigh) != 8 {
		t.Fatalf("high rpm=%d want 8", split.RPM(20, PriorityHigh))
	}
}

func TestQuotaLimiterRPD(t *testing.T) {
	q := NewQuotaLimiter(LimitConfig{ModelLimits: ModelLimits{RPM: 1000, TPM: 1_000_000, RPD: 2}})
	ctx := context.Background()
	if err := q.WaitGenerate(ctx, PriorityNormal, 1); err != nil {
		t.Fatal(err)
	}
	if err := q.WaitGenerate(ctx, PriorityNormal, 1); err != nil {
		t.Fatal(err)
	}
	if err := q.WaitGenerate(ctx, PriorityNormal, 1); err == nil {
		t.Fatal("expected rpd error")
	}
}
