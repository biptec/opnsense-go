package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type searchTestResource struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

func TestSearch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/interfaces/vxlan_settings/search_item" {
			t.Errorf("path = %q, want interface search path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1, "rowCount": 1, "current": 1,
			"rows": []map[string]string{{"uuid": "id-1", "name": "example"}},
		})
	}))
	defer server.Close()

	client := NewClient(Options{Uri: server.URL})
	result, err := Search[searchTestResource](client, context.Background(), "/interfaces/vxlan_settings/search_item")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 || len(result.Rows) != 1 || result.Rows[0].UUID != "id-1" {
		t.Fatalf("Search() result = %#v", result)
	}
}
