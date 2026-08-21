package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bidshard/parser/internal/crm/admin"
	crmmetrics "github.com/bidshard/parser/internal/crm/metrics"
	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/sink"
)

// Parser POSTs here after sink.WrapWebhook upserts the lead into shared Mongo.
// This handler only validates JSON shape and auth; it does not write again.
const maxBodyBytes = 64 << 10

type Handler struct {
	secret string
}

func NewHandler(secret string) *Handler {
	return &Handler{secret: strings.TrimSpace(secret)}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/v1/leads" {
		http.NotFound(w, r)
		return
	}
	if !h.authBearer(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if _, err := decodeLeadDoc(w, r); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	crmmetrics.IncWebhookAccepted()
	w.WriteHeader(http.StatusAccepted)
}

func (h *Handler) authBearer(r *http.Request) bool {
	if h == nil {
		return false
	}
	// Empty secret skips Bearer check (local dev only; set CRM_WEBHOOK_SECRET in prod).
	if h.secret == "" {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
	return token == h.secret
}

func decodeLeadDoc(w http.ResponseWriter, r *http.Request) (sink.LeadDoc, error) {
	if r == nil || r.Body == nil {
		return sink.LeadDoc{}, errors.New("request body missing")
	}
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer func() { _ = body.Close() }()

	var doc sink.LeadDoc
	if err := json.NewDecoder(body).Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return sink.LeadDoc{}, fmt.Errorf("empty body")
		}
		return sink.LeadDoc{}, err
	}
	doc.HashID = strings.TrimSpace(doc.HashID)
	if doc.HashID == "" {
		return sink.LeadDoc{}, fmt.Errorf("hash_id required")
	}
	return doc, nil
}

type Server struct {
	addr   string
	server *http.Server
}

// NewMux serves parser webhook ingest and /v1/admin/* CRM API on one listener.
// Admin routes have no in-process auth; Caddy basicauth sits in front on VPS.
 func NewMux(webhookSecret string, leadStore *store.LeadStore) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/v1/leads", NewHandler(webhookSecret))
	if leadStore != nil {
		mux.Handle("/v1/admin/", admin.NewHandler(leadStore))
	}
	return mux
}

func NewServer(addr string, handler http.Handler) *Server {
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	return &Server{
		addr: addr,
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

func (s *Server) ListenAndServe() error {
	if s == nil || s.server == nil {
		return fmt.Errorf("webhook server not initialized")
	}
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
