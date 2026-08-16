package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestHasyncGetSetReconfigure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/core/hasync/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"hasync": map[string]any{
				"disablepreempt": "0", "disconnectppps": "0",
				"pfsyncinterface": selectedMapForHasync("lan"), "pfsyncpeerip": "10.0.0.2",
				"pfsyncversion": selectedMapForHasync("1400"), "pfsyncdefer": "1",
				"synchronizetoip": "10.0.0.2", "verifypeer": "1",
				"username": "ha-sync", "password": "secret",
				"syncitems": selectedListForHasync("nat", "rules", "virtualip"),
			}})
		case "/api/core/hasync/set":
			var body map[string]HasyncSettings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode hasync body: %v", err)
			}
			got := body["hasync"]
			if got.PfsyncInterface.String() != "lan" || got.PfsyncVersion.String() != "1400" {
				t.Fatalf("unexpected pfsync settings: %+v", got)
			}
			if got.SyncItems.String() != "nat,rules,virtualip" || got.Password != "secret" {
				t.Fatalf("unexpected HA sync settings: %+v", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved"})
		case "/api/core/hasync/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	ctx := context.Background()
	result, err := controller.HasyncGet(ctx)
	if err != nil {
		t.Fatalf("HasyncGet() error = %v", err)
	}
	if result.Hasync.PfsyncInterface.String() != "lan" || result.Hasync.SyncItems.String() != "nat,rules,virtualip" {
		t.Fatalf("unexpected hasync response: %+v", result.Hasync)
	}
	if _, err := controller.HasyncSet(ctx, &result.Hasync); err != nil {
		t.Fatalf("HasyncSet() error = %v", err)
	}
	if _, err := controller.HasyncReconfigure(ctx); err != nil {
		t.Fatalf("HasyncReconfigure() error = %v", err)
	}
}

func selectedMapForHasync(value string) map[string]any {
	return map[string]any{value: map[string]any{"value": value, "selected": 1}}
}

func selectedListForHasync(values ...string) map[string]any {
	out := make(map[string]any, len(values))
	for _, value := range values {
		out[value] = map[string]any{"value": value, "selected": 1}
	}
	return out
}
