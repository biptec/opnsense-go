package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestHasyncSync(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/core/hasync_status/sync/interface_vlans,virtualip,rules" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"items":  []string{"interface_vlans", "virtualip", "rules"},
		})
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	result, err := controller.HasyncSync(context.Background(), []string{"interface_vlans", "virtualip", "rules", "rules"})
	if err != nil {
		t.Fatalf("HasyncSync() error = %v", err)
	}
	if result.Status != "ok" || len(result.Items) != 3 {
		t.Fatalf("unexpected HA sync result: %+v", result)
	}
}

func TestHasyncSyncRejectsEmptyItems(t *testing.T) {
	t.Parallel()
	controller := &Controller{}
	if _, err := controller.HasyncSync(context.Background(), nil); err == nil {
		t.Fatal("HasyncSync() must reject an empty item list")
	}
}
