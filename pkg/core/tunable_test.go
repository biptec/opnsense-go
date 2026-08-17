package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestTunableAPI(t *testing.T) {
	const uuid = "11111111-2222-4333-8444-555555555555"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/core/tunables/add_item" && r.Method == http.MethodPost:
			var body map[string]Tunable
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode tunable: %v", err)
			}
			if body["sysctl"].Tunable != "kern.ipc.maxsockbuf" || body["sysctl"].Value != "33554432" {
				t.Fatalf("unexpected tunable body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved", "uuid": uuid})
		case r.URL.Path == "/api/core/tunables/reconfigure" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case r.URL.Path == "/api/core/tunables/get_item/"+uuid && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"sysctl": map[string]any{
				"tunable": "kern.ipc.maxsockbuf", "value": "33554432", "descr": "FRR socket buffer",
				"default_value": "4262144", "type": "integer",
			}})
		case r.URL.Path == "/api/core/tunables/search_item" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"uuid": uuid, "tunable": "kern.ipc.maxsockbuf", "value": "33554432"}},
			})
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	ctx := context.Background()
	id, err := controller.AddTunable(ctx, &Tunable{
		Tunable: "kern.ipc.maxsockbuf", Value: "33554432", Description: "FRR socket buffer",
	})
	if err != nil || id != uuid {
		t.Fatalf("AddTunable() = %q, %v", id, err)
	}
	got, err := controller.GetTunable(ctx, uuid)
	if err != nil || got.Tunable != "kern.ipc.maxsockbuf" || got.Value != "33554432" {
		t.Fatalf("GetTunable() = %#v, %v", got, err)
	}
	search, err := controller.SearchTunable(ctx)
	if err != nil || len(search.Rows) != 1 || search.Rows[0].Tunable != "kern.ipc.maxsockbuf" {
		t.Fatalf("SearchTunable() = %#v, %v", search, err)
	}
}

func TestTunableEndpointContract(t *testing.T) {
	if TunableOpts.Monad != "sysctl" || TunableOpts.Create.Path != "/core/tunables/add_item" || TunableOpts.Reconfigure.Path != "/core/tunables/reconfigure" {
		t.Fatalf("unexpected tunable endpoints: %#v", TunableOpts)
	}
}
