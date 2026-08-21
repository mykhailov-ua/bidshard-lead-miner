package metrics_test

import (
	"testing"

	crmmetrics "github.com/bidshard/parser/internal/crm/metrics"
)

func TestWebhookAcceptedCounter(t *testing.T) {
	before := crmmetrics.WebhookAcceptedTotal()
	crmmetrics.IncWebhookAccepted()
	if got := crmmetrics.WebhookAcceptedTotal(); got != before+1 {
		t.Fatalf("got %d want %d", got, before+1)
	}
}
