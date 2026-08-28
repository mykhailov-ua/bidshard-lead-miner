package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bidshard/parser/internal/crm/store"
)

const maxBodyBytes = 16 << 10

// Handler exposes /v1/admin/* on the crm-bot HTTP listener.
// Auth is enforced by Caddy basicauth in production, not inside this package.
type Handler struct {
	store     *store.LeadStore
	explainer LeadExplainer
}

func NewHandler(leadStore *store.LeadStore, explainer LeadExplainer) *Handler {
	return &Handler{store: leadStore, explainer: explainer}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	if h.store == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	switch {
	case r.Method == http.MethodGet && path == "/v1/admin/stats":
		h.handleStats(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/feedback":
		h.handleFeedback(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/leads/explain":
		h.handleExplainLead(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/leads/search":
		h.handleSearchLeads(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/leads/get":
		h.handleGetLead(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/leads/notes":
		h.handleListNotes(w, r)
	case r.Method == http.MethodPost && path == "/v1/admin/leads/notes":
		h.handleAddNote(w, r)
	case r.Method == http.MethodPost && path == "/v1/admin/leads/tags":
		h.handleAddTag(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/leads":
		h.handleListLeads(w, r)
	case r.Method == http.MethodPost && path == "/v1/admin/leads/purge":
		h.handlePurge(w, r)
	case r.Method == http.MethodDelete && path == "/v1/admin/leads":
		h.handleDeleteOne(w, r)
	case r.Method == http.MethodPatch && path == "/v1/admin/leads":
		h.handlePatchLead(w, r)
	case r.Method == http.MethodPost && path == "/v1/admin/leads/outcome":
		h.handleSetOutcome(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/entities/list":
		h.handleListEntities(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/entities/get":
		h.handleGetEntity(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/entities/leads":
		h.handleListEntityLeads(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/entities/inbox":
		h.handleEntityInbox(w, r)
	case r.Method == http.MethodGet && path == "/v1/admin/entities/suggestions":
		h.handleEntitySuggestions(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.DBStats(r.Context())
	if err != nil {
		http.Error(w, "stats failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, stats)
}

func (h *Handler) handleFeedback(w http.ResponseWriter, r *http.Request) {
	channels := strings.TrimSpace(r.URL.Query().Get("channels"))
	if channels == "" {
		channels = "data/runtime/discovered_telegram_channels.json"
	}
	disabledDorks := strings.TrimSpace(r.URL.Query().Get("disabled_dorks"))
	if disabledDorks == "" {
		disabledDorks = "data/runtime/disabled_dorks.json"
	}
	report, err := h.store.FeedbackReport(r.Context(), channels, disabledDorks)
	if err != nil {
		http.Error(w, "feedback failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, report)
}

func (h *Handler) handleListLeads(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 50)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	inbox := parseInboxOnly(r.URL.Query().Get("inbox"), status)
	filter := store.ListFilter{
		Status:            status,
		SourcePrefix:      strings.TrimSpace(r.URL.Query().Get("source_prefix")),
		ContactChannel:    strings.TrimSpace(r.URL.Query().Get("contact_channel")),
		NextAction:        strings.TrimSpace(r.URL.Query().Get("next_action")),
		Outcome:           strings.TrimSpace(r.URL.Query().Get("outcome")),
		ScoreMax:          parseScoreMax(r.URL.Query().Get("score_max")),
		Limit:             limit,
		InboxOnly:         inbox,
		MinEngagePriority: store.ResolveEngagePriorityMin(r.URL.Query().Get("engage_min"), inbox),
		Sort:              parseListSort(r.URL.Query().Get("sort"), status),
	}
	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleSearchLeads(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Error(w, "q required", http.StatusBadRequest)
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	result, err := h.store.Search(r.Context(), query, limit, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleGetLead(w http.ResponseWriter, r *http.Request) {
	hashID := strings.TrimSpace(r.URL.Query().Get("hash_id"))
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	hashID, err := h.store.ResolveHashID(r.Context(), hashID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, store.ErrAmbiguousHash) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "get failed", http.StatusInternalServerError)
		return
	}
	lead, err := h.store.GetByHashID(r.Context(), hashID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "get failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, lead)
}

func (h *Handler) handleExplainLead(w http.ResponseWriter, r *http.Request) {
	if h.explainer == nil {
		http.Error(w, "explain not configured", http.StatusServiceUnavailable)
		return
	}
	hashID := strings.TrimSpace(r.URL.Query().Get("hash_id"))
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	summary, err := h.explainer.Explain(r.Context(), hashID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "explain failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"hash_id": hashID, "summary": summary})
}

func (h *Handler) handleListNotes(w http.ResponseWriter, r *http.Request) {
	hashID := strings.TrimSpace(r.URL.Query().Get("hash_id"))
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	notes, err := h.store.ListNotes(r.Context(), hashID, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"notes": notes})
}

type addNoteRequest struct {
	HashID string `json:"hash_id"`
	Text   string `json:"text"`
}

func (h *Handler) handleAddNote(w http.ResponseWriter, r *http.Request) {
	var req addNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hashID := strings.TrimSpace(req.HashID)
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	if err := h.store.AddNote(r.Context(), hashID, req.Text, 0); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"hash_id": hashID, "status": "ok"})
}

type addTagRequest struct {
	HashID string `json:"hash_id"`
	Tag    string `json:"tag"`
}

func (h *Handler) handleAddTag(w http.ResponseWriter, r *http.Request) {
	var req addTagRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hashID := strings.TrimSpace(req.HashID)
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	meta, err := h.store.AddTag(r.Context(), hashID, req.Tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, meta)
}

type purgeRequest struct {
	Confirm      string `json:"confirm"`
	HashID       string `json:"hash_id"`
	Status       string `json:"status"`
	SourcePrefix string `json:"source_prefix"`
	ScoreMax     int    `json:"score_max"`
	All          bool   `json:"all"`
}

func (h *Handler) handlePurge(w http.ResponseWriter, r *http.Request) {
	var req purgeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	filter := store.DeleteFilter{
		HashID:       req.HashID,
		Status:       req.Status,
		SourcePrefix: req.SourcePrefix,
		ScoreMax:     req.ScoreMax,
		All:          req.All,
	}
	if err := filter.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Confirm) != "purge" {
		http.Error(w, "confirm must be purge", http.StatusBadRequest)
		return
	}
	result, err := h.store.DeleteLeads(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleDeleteOne(w http.ResponseWriter, r *http.Request) {
	hashID := strings.TrimSpace(r.URL.Query().Get("hash_id"))
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	result, err := h.store.DeleteLeads(r.Context(), store.DeleteFilter{HashID: hashID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, result)
}

type patchLeadRequest struct {
	HashID string `json:"hash_id"`
	Status string `json:"status"`
}

func (h *Handler) handlePatchLead(w http.ResponseWriter, r *http.Request) {
	var req patchLeadRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hashID := strings.TrimSpace(req.HashID)
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateStatus(r.Context(), hashID, req.Status); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"hash_id": hashID, "status": strings.TrimSpace(req.Status)})
}

type setOutcomeRequest struct {
	HashID  string `json:"hash_id"`
	Outcome string `json:"outcome"`
	Note    string `json:"note,omitempty"`
}

func (h *Handler) handleSetOutcome(w http.ResponseWriter, r *http.Request) {
	var req setOutcomeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hashID := strings.TrimSpace(req.HashID)
	if hashID == "" {
		http.Error(w, "hash_id required", http.StatusBadRequest)
		return
	}
	if err := h.store.SetOutcome(r.Context(), hashID, req.Outcome, req.Note); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, store.ErrInvalidOutcome) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	outcome, _ := store.NormalizeOutcome(req.Outcome)
	writeJSON(w, map[string]string{"hash_id": hashID, "outcome": outcome})
}

func (h *Handler) handleListEntities(w http.ResponseWriter, r *http.Request) {
	if !h.store.EntitiesEnabled() {
		http.Error(w, "entities not configured", http.StatusServiceUnavailable)
		return
	}
	result, err := h.store.ListEntities(r.Context(), store.EntityListFilter{
		MinTier: strings.TrimSpace(r.URL.Query().Get("min_tier")),
		Limit:   parseLimit(r.URL.Query().Get("limit"), 20),
	})
	if err != nil {
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleGetEntity(w http.ResponseWriter, r *http.Request) {
	if !h.store.EntitiesEnabled() {
		http.Error(w, "entities not configured", http.StatusServiceUnavailable)
		return
	}
	entityID := strings.TrimSpace(r.URL.Query().Get("entity_id"))
	if entityID == "" {
		http.Error(w, "entity_id required", http.StatusBadRequest)
		return
	}
	doc, err := h.store.GetEntity(r.Context(), entityID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "get failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, doc)
}

func (h *Handler) handleListEntityLeads(w http.ResponseWriter, r *http.Request) {
	if !h.store.EntitiesEnabled() {
		http.Error(w, "entities not configured", http.StatusServiceUnavailable)
		return
	}
	entityID := strings.TrimSpace(r.URL.Query().Get("entity_id"))
	if entityID == "" {
		http.Error(w, "entity_id required", http.StatusBadRequest)
		return
	}
	leads, err := h.store.ListEntityLeads(r.Context(), entityID, parseLimit(r.URL.Query().Get("limit"), 20))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "list failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"entity_id": entityID, "leads": leads})
}

func (h *Handler) handleEntityInbox(w http.ResponseWriter, r *http.Request) {
	if !h.store.EntitiesEnabled() {
		http.Error(w, "entities not configured", http.StatusServiceUnavailable)
		return
	}
	entityID := strings.TrimSpace(r.URL.Query().Get("entity_id"))
	if entityID != "" {
		card, err := h.store.GetEntityInbox(r.Context(), entityID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, "get failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, card)
		return
	}
	minSightings := 2
	if raw := strings.TrimSpace(r.URL.Query().Get("min_sightings")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			minSightings = n
		}
	}
	result, err := h.store.ListEntityInbox(r.Context(), store.EntityInboxFilter{
		MinTier:           strings.TrimSpace(r.URL.Query().Get("min_tier")),
		MinSightings:      minSightings,
		OnlyNeedsWork:     parseQueryBool(r.URL.Query().Get("needs_work")),
		MinEngagePriority: store.ResolveEngagePriorityMin(r.URL.Query().Get("engage_min"), true),
		Limit:             parseLimit(r.URL.Query().Get("limit"), 20),
	})
	if err != nil {
		http.Error(w, "inbox failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (h *Handler) handleEntitySuggestions(w http.ResponseWriter, r *http.Request) {
	if !h.store.EntitiesEnabled() {
		http.Error(w, "entities not configured", http.StatusServiceUnavailable)
		return
	}
	docs, err := h.store.ListPendingReviewSuggestions(r.Context(), parseLimit(r.URL.Query().Get("limit"), 20))
	if err != nil {
		http.Error(w, "suggestions failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"entities": docs})
}

func parseQueryBool(raw string) bool {
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && v
}

// parseListSort defaults to engage_priority for sales inbox (status=new).
func parseListSort(raw, status string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "score", "heat", "engage":
		return raw
	default:
		if strings.EqualFold(strings.TrimSpace(status), "new") || strings.TrimSpace(status) == "" {
			return "engage"
		}
		return "heat"
	}
}

func parseLimit(raw string, fallback int64) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// parseScoreMax returns 0 when unset or invalid; 0 means "no score cap" in list/purge filters.
func parseScoreMax(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func decodeJSON(w http.ResponseWriter, r *http.Request, out any) error {
	if r == nil || r.Body == nil {
		return fmt.Errorf("body missing")
	}
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer func() { _ = body.Close() }()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	// Reject {"a":1}{"b":2} style bodies after the first object.
	if err := dec.Decode(&struct{}{}); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("trailing json")
	}
	return nil
}

// parseInboxOnly enables sales inbox filtering. Default on for status=new unless inbox=0|false.
func parseInboxOnly(raw, status string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	switch raw {
	case "0", "false", "no":
		return false
	case "1", "true", "yes":
		return true
	default:
		return strings.EqualFold(strings.TrimSpace(status), "new")
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}
