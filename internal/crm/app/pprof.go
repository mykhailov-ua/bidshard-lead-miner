package app

import (
	"context"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"time"
)

// StartPprofServer exposes net/http/pprof on addr until ctx is cancelled.
func StartPprofServer(ctx context.Context, addr string) *http.Server {
	if addr == "" {
		return nil
	}
	server := &http.Server{
		Addr:    addr,
		Handler: http.DefaultServeMux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("crm pprof server stopped", "error", err)
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
