package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// StartServer launches the CRM metrics endpoint until ctx is cancelled.
func StartServer(ctx context.Context, addr string) *http.Server {
	if addr == "" {
		return nil
	}
	server := &http.Server{
		Addr:    addr,
		Handler: Handler(),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("crm metrics server stopped", "error", err)
		}
	}()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	return server
}
