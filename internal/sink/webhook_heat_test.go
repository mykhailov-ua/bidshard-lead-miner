package sink

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bidshard/parser/internal/entity"
	"github.com/bidshard/parser/internal/model"
)

func TestWebhookHeatGateAllows(t *testing.T) {
	gate := WebhookHeatGate{MinTier: entity.HeatTierHot}
	if !gate.Allows(model.Lead{HeatTier: entity.HeatTierHot}) {
		t.Fatal("expected hot allowed")
	}
	if gate.Allows(model.Lead{HeatTier: entity.HeatTierWarm}) {
		t.Fatal("expected warm blocked")
	}
	if gate.Allows(model.Lead{}) {
		t.Fatal("expected empty tier blocked when min hot")
	}
}

func TestWebhookHeatGateColdDisabled(t *testing.T) {
	gate := WebhookHeatGate{MinTier: entity.HeatTierCold}
	if !gate.Allows(model.Lead{}) {
		t.Fatal("expected cold min to allow empty tier")
	}
}

func TestWebhookClientSkipsBelowHeatMin(t *testing.T) {
	var mu sync.Mutex
	posted := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		posted++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewWebhookClient(srv.URL, "", time.Second).WithHeatMin(entity.HeatTierHot)

	client.NotifyLead(model.Lead{HashID: "a", HeatTier: entity.HeatTierWarm})
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	if posted != 0 {
		t.Fatalf("posted=%d want 0", posted)
	}
	mu.Unlock()

	client.NotifyLead(model.Lead{HashID: "b", HeatTier: entity.HeatTierHot})
	deadline := time.Now().Add(2 * time.Second)
	for {
		mu.Lock()
		n := posted
		mu.Unlock()
		if n == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("posted=%d want 1", n)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
