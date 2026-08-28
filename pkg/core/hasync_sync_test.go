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
		t.Logf("request method=%s path=%s content_type=%s", r.Method, r.URL.Path, r.Header.Get("Content-Type"))
		if r.Method != http.MethodPost || r.URL.Path != "/api/core/hasync_status/sync" {
			http.NotFound(w, r)
			return
		}
		var body struct {
			Items []string `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode sync request: %v", err)
		}
		if len(body.Items) != 3 || body.Items[0] != "interface_vlans" || body.Items[1] != "virtualip" || body.Items[2] != "rules" {
			t.Fatalf("unexpected sync request items: %#v", body.Items)
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
