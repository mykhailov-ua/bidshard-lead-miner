package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

var (
	webhookAcceptedTotal int64
)

func IncWebhookAccepted() {
	atomic.AddInt64(&webhookAcceptedTotal, 1)
}

func WebhookAcceptedTotal() int64 {
	return atomic.LoadInt64(&webhookAcceptedTotal)
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintln(w, "# HELP crm_webhook_accepted_total Parser lead webhook accepts")
		_, _ = fmt.Fprintln(w, "# TYPE crm_webhook_accepted_total counter")
		_, _ = fmt.Fprintf(w, "crm_webhook_accepted_total %d\n", WebhookAcceptedTotal())
	})
}
