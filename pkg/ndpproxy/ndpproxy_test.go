package ndpproxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestSettingsAndReconfigure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/ndpproxy/general/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"ndpproxy": map[string]any{"general": map[string]any{
				"enabled":    "1",
				"upstream":   map[string]any{"wan": map[string]any{"value": "wan", "selected": 1}},
				"downstream": map[string]any{"opt1": map[string]any{"value": "opt1", "selected": 1}},
				"ra":         "0", "routes": "0", "carp_depend_on": "1",
			}}})
		case "/api/ndpproxy/general/set":
			var body map[string]Settings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode settings body: %v", err)
			}
			got := body["ndpproxy"].General
			if got.Enabled != "1" || got.Upstream.String() != "wan" || got.Downstream.String() != "opt1" || got.Routes != "0" || got.CarpDependOn != "1" {
				t.Fatalf("unexpected settings body: %+v", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved"})
		case "/api/ndpproxy/service/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	settings, err := controller.SettingsGet(context.Background())
	if err != nil {
		t.Fatalf("SettingsGet() error = %v", err)
	}
	general := settings.NdpProxy.General
	if general.Upstream.String() != "wan" || general.Downstream.String() != "opt1" || general.Routes != "0" || general.CarpDependOn != "1" {
		t.Fatalf("unexpected settings: %+v", general)
	}
	if _, err := controller.SettingsSet(context.Background(), &settings.NdpProxy); err != nil {
		t.Fatalf("SettingsSet() error = %v", err)
	}
	if _, err := controller.ServiceReconfigure(context.Background()); err != nil {
		t.Fatalf("ServiceReconfigure() error = %v", err)
	}
}
