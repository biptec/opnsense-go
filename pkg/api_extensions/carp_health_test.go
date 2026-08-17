package api_extensions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestCarpHealthAPI(t *testing.T) {
	const uuid = "11111111-2222-4333-8444-555555555555"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/api_extensions/carp_health/get" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"carp_health": map[string]any{
				"enabled": "1", "interval": "1", "failure_threshold": "2", "recovery_threshold": "2",
			}})
		case r.URL.Path == "/api/api_extensions/carp_health/set" && r.Method == http.MethodPost:
			var body map[string]CarpHealthSettings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode settings: %v", err)
			}
			if body["carp_health"].Enabled != "1" || body["carp_health"].Interval != "1" {
				t.Fatalf("unexpected settings: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved"})
		case r.URL.Path == "/api/api_extensions/carp_health/status" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok", "enabled": true, "ready": true, "healthy": true, "running": true,
				"timestamp": 123.0, "config_signature": "sig",
				"checks": []map[string]any{{
					"uuid": uuid, "name": "leaf", "interface": "opt2", "device": "vlan02",
					"target": "192.0.2.2", "healthy": true, "failures": 0, "successes": 0,
				}},
			})
		case r.URL.Path == "/api/api_extensions/carp_health/addCheck" && r.Method == http.MethodPost:
			var body map[string]CarpHealthCheck
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode check: %v", err)
			}
			if string(body["check"].Interface) != "opt2" || body["check"].Target != "192.0.2.2" {
				t.Fatalf("unexpected check: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved", "uuid": uuid})
		case r.URL.Path == "/api/api_extensions/carp_health/getCheck/"+uuid && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"check": map[string]any{
				"enabled": "1", "name": "leaf", "interface": "opt2", "target": "192.0.2.2",
			}})
		case r.URL.Path == "/api/api_extensions/carp_health/setCheck/"+uuid && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved", "uuid": uuid})
		case r.URL.Path == "/api/api_extensions/carp_health/delCheck/"+uuid && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "deleted"})
		case r.URL.Path == "/api/api_extensions/carp_health/searchCheck" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"enabled": true, "name": "leaf", "interface": "opt2", "target": "192.0.2.2"}},
			})
		case r.URL.Path == "/api/api_extensions/carp_health/reconfigure" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "result": "ok"})
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	ctx := context.Background()

	got, err := controller.CarpHealthGet(ctx)
	if err != nil || got.CarpHealth.Enabled != "1" || got.CarpHealth.FailureThreshold != "2" {
		t.Fatalf("CarpHealthGet() = %#v, %v", got, err)
	}
	settings := &CarpHealthSettings{Enabled: "1", Interval: "1", FailureThreshold: "2", RecoveryThreshold: "2"}
	if result, err := controller.CarpHealthSet(ctx, settings); err != nil || result.Result != "saved" {
		t.Fatalf("CarpHealthSet() = %#v, %v", result, err)
	}
	status, err := controller.CarpHealthStatus(ctx)
	if err != nil || !status.Running || !status.Healthy || len(status.Checks) != 1 {
		t.Fatalf("CarpHealthStatus() = %#v, %v", status, err)
	}

	check := &CarpHealthCheck{Enabled: "1", Name: "leaf", Interface: api.SelectedMap("opt2"), Target: "192.0.2.2"}
	id, err := controller.AddCarpHealthCheck(ctx, check)
	if err != nil || id != uuid {
		t.Fatalf("AddCarpHealthCheck() = %q, %v", id, err)
	}
	retrieved, err := controller.GetCarpHealthCheck(ctx, uuid)
	if err != nil || retrieved.Name != "leaf" || retrieved.Enabled != "1" || retrieved.Interface.String() != "opt2" {
		t.Fatalf("GetCarpHealthCheck() = %#v, %v", retrieved, err)
	}
	search, err := controller.SearchCarpHealthCheck(ctx)
	if err != nil || search.Total != 1 || search.Rows[0].Enabled != "1" {
		t.Fatalf("SearchCarpHealthCheck() = %#v, %v", search, err)
	}
	check.Target = "192.0.2.3"
	if err := controller.UpdateCarpHealthCheck(ctx, uuid, check); err != nil {
		t.Fatalf("UpdateCarpHealthCheck(): %v", err)
	}
	if err := controller.DeleteCarpHealthCheck(ctx, uuid); err != nil {
		t.Fatalf("DeleteCarpHealthCheck(): %v", err)
	}
}

func TestFilterLogAcceptsBoolean(t *testing.T) {
	var filter struct {
		Log api.BoolString `json:"log"`
	}
	for input, want := range map[string]api.BoolString{`{"log":false}`: "0", `{"log":true}`: "1", `{"log":"0"}`: "0"} {
		if err := json.NewDecoder(strings.NewReader(input)).Decode(&filter); err != nil {
			t.Fatalf("decode %s: %v", input, err)
		}
		if filter.Log != want {
			t.Fatalf("decode %s log=%q want=%q", input, filter.Log, want)
		}
	}
}
