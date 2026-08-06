package firewall

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestSourceNATSettingsRPC(t *testing.T) {
	t.Parallel()
	setCalled := false
	applyCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/firewall/source_nat/get":
			if r.Method != http.MethodGet {
				t.Fatalf("GET method = %s", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"filter": map[string]any{"general": map[string]any{
				"snat_mode": map[string]any{
					"automatic": map[string]any{"value": "Automatic", "selected": 1},
					"hybrid":    map[string]any{"value": "Hybrid", "selected": 0},
					"advanced":  map[string]any{"value": "Manual", "selected": 0},
					"disabled":  map[string]any{"value": "Disabled", "selected": 0},
				},
			}}})
		case "/api/firewall/source_nat/set":
			if r.Method != http.MethodPost {
				t.Fatalf("SET method = %s", r.Method)
			}
			var payload struct {
				Filter SourceNATSettings `json:"filter"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode SET body: %v", err)
			}
			if payload.Filter.General.Mode.String() != "hybrid" {
				t.Fatalf("SET mode = %q", payload.Filter.General.Mode.String())
			}
			setCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved"})
		case "/api/firewall/source_nat/apply":
			if r.Method != http.MethodPost {
				t.Fatalf("APPLY method = %s", r.Method)
			}
			applyCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "OK\n\n"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := firewallTestController(server)
	ctx := context.Background()
	settings, err := controller.SourceNATSettingsGet(ctx)
	if err != nil {
		t.Fatalf("SourceNATSettingsGet() error = %v", err)
	}
	if settings.Filter.General.Mode.String() != "automatic" {
		t.Fatalf("GET mode = %q", settings.Filter.General.Mode.String())
	}

	updated := &SourceNATSettings{General: SourceNATGeneralSettings{Mode: api.SelectedMap("hybrid")}}
	if _, err = controller.SourceNATSettingsSet(ctx, updated); err != nil {
		t.Fatalf("SourceNATSettingsSet() error = %v", err)
	}
	applyResult, err := controller.SourceNATSettingsApply(ctx)
	if err != nil {
		t.Fatalf("SourceNATSettingsApply() error = %v", err)
	}
	if applyResult.Status != "OK\n\n" {
		t.Fatalf("APPLY status = %q", applyResult.Status)
	}
	if !setCalled || !applyCalled {
		t.Fatalf("RPC calls missing: set=%t apply=%t", setCalled, applyCalled)
	}
}
