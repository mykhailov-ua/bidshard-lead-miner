package gemini

import (
	"context"
	"testing"
	"time"
)

func TestLimitsForModelFlash(t *testing.T) {
	l := LimitsForModel("gemini-2.0-flash")
	if l.RPM != 15 || l.RPD != 1500 {
		t.Fatalf("unexpected flash limits: %+v", l)
	}
}

func TestQuotaLimiterRPM(t *testing.T) {
	q := NewQuotaLimiter(LimitConfig{ModelLimits: ModelLimits{RPM: 60, TPM: 1_000_000, RPD: 1000}})
	ctx := context.Background()
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := q.WaitGenerate(ctx, 10); err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		t.Fatalf("expected rpm throttle, elapsed=%v", elapsed)
	}
}

func TestQuotaLimiterRPD(t *testing.T) {
	q := NewQuotaLimiter(LimitConfig{ModelLimits: ModelLimits{RPM: 1000, TPM: 1_000_000, RPD: 2}})
	ctx := context.Background()
	if err := q.WaitGenerate(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := q.WaitGenerate(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := q.WaitGenerate(ctx, 1); err == nil {
		t.Fatal("expected rpd error")
	}
}
