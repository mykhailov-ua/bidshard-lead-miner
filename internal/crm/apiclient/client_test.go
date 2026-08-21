package apiclient

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/parser/internal/crm/store"
	"github.com/bidshard/parser/internal/sink"
)

func TestClientListLeads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/admin/leads" {
			http.NotFound(w, r)
			return
		}
		if user, pass, ok := r.BasicAuth(); !ok || user != "sales" || pass != "secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(store.ListResult{
			Leads: []sink.LeadDoc{{HashID: "abc", Score: 90, Source: "forum:test", Status: "new"}},
		})
	}))
	defer srv.Close()

	client := New(Config{BaseURL: srv.URL, Username: "sales", Password: "secret"})
	var result store.ListResult
	if err := client.GetJSON(t.Context(), "/v1/admin/leads?status=new", &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Leads) != 1 || result.Leads[0].HashID != "abc" {
		t.Fatalf("leads=%+v", result.Leads)
	}
}
