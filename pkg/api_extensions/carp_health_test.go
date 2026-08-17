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
			route := map[string]any{
				"key": uuid + ":inet", "check_uuid": uuid, "check": "wan-health", "family": "inet",
				"destination": "192.0.2.2", "gateway": "10.16.224.5", "desired_installed": true,
				"installed": true, "managed": true, "control_ok": true, "retired": false, "error": "",
			}
			vhidState := map[string]any{
				"key": "opt2:51", "interface": "opt2", "device": "vlan02", "vhid": 51,
				"checks": []string{"wan-health"}, "ready": true, "healthy": false, "desired_demoted": true,
				"desired_advskew": 200, "configured_advskew": 10, "current_advskew": 200, "carp_state": "MASTER",
				"control_ok": true, "retired": false, "error": "",
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok", "enabled": true, "ready": true, "healthy": false, "probe_healthy": false,
				"control_ok": true, "running": true, "timestamp": 123.0, "config_signature": "sig",
				"global": map[string]any{"active": false, "check_count": 0, "ready": true, "healthy": true},
				"vhids":  []map[string]any{vhidState},
				"routes": []map[string]any{route},
				"checks": []map[string]any{{
					"uuid": uuid, "name": "wan-health", "interface": "wan", "device": "vtnet1",
					"target": "192.0.2.1", "scope": "all_carp", "vhid": 0,
					"vhid_targets": []string{"opt2:51"}, "configured_vhid_targets": []string{}, "failure_advskew": 200,
					"vhid_states": []map[string]any{vhidState}, "fallback_routes": []map[string]any{route},
					"carp_state": "GROUP", "configured_advskew": nil, "current_advskew": nil, "control_ok": true,
					"healthy": false, "failures": 2, "successes": 0,
				}},
			})
		case r.URL.Path == "/api/api_extensions/carp_health/addCheck" && r.Method == http.MethodPost:
			var body map[string]CarpHealthCheck
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode check: %v", err)
			}
			check := body["check"]
			if check.Interface.String() != "wan" || check.Target != "192.0.2.1" || check.Scope.String() != "vhid_group" ||
				check.VHID != "0" || check.FailureAdvSkew != "200" || len(check.VHIDTargets) != 2 ||
				check.VHIDTargets[0] != "opt2:51" || check.VHIDTargets[1] != "opt3:52" ||
				check.FallbackIPv4Target != "192.0.2.2" || check.FallbackIPv4Gateway != "10.16.224.5" ||
				check.FallbackIPv6Target != "2001:db8:1::2" || check.FallbackIPv6Gateway != "2001:db8:2::1" {
				t.Fatalf("unexpected check: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved", "uuid": uuid})
		case r.URL.Path == "/api/api_extensions/carp_health/getCheck/"+uuid && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"check": map[string]any{
				"enabled": "1", "name": "leaf", "interface": "opt2", "target": "192.0.2.2",
				"scope": "interface", "vhid": "0", "failure_advskew": "254", "vhid_targets": "",
				"fallback_ipv4_target": "", "fallback_ipv4_gateway": "", "fallback_ipv6_target": "", "fallback_ipv6_gateway": "",
			}})
		case r.URL.Path == "/api/api_extensions/carp_health/setCheck/"+uuid && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved", "uuid": uuid})
		case r.URL.Path == "/api/api_extensions/carp_health/delCheck/"+uuid && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "deleted"})
		case r.URL.Path == "/api/api_extensions/carp_health/searchCheck" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"enabled": true, "name": "leaf", "interface": "opt2", "target": "192.0.2.2", "scope": "interface", "vhid": "0"}},
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
	if err != nil || !status.Running || status.Healthy || status.ProbeHealthy || !status.ControlOK ||
		len(status.Checks) != 1 || len(status.VHIDs) != 1 || len(status.Routes) != 1 || status.VHIDs[0].VHID != 51 ||
		status.VHIDs[0].DesiredAdvSkew == nil || *status.VHIDs[0].DesiredAdvSkew != 200 ||
		status.Global.CheckCount != 0 || status.Checks[0].Scope != "all_carp" || status.Checks[0].VHID != 0 ||
		len(status.Checks[0].VHIDTargets) != 1 || status.Checks[0].VHIDTargets[0] != "opt2:51" ||
		len(status.Checks[0].ConfiguredVHIDTargets) != 0 || status.Checks[0].FailureAdvSkew != 200 ||
		len(status.Checks[0].VHIDStates) != 1 || len(status.Checks[0].FallbackRoutes) != 1 ||
		status.Routes[0].Gateway != "10.16.224.5" || !status.Routes[0].Managed || !status.Routes[0].Installed {
		t.Fatalf("CarpHealthStatus() = %#v, %v", status, err)
	}

	check := &CarpHealthCheck{
		Enabled: "1", Name: "wan-health", Interface: api.SelectedMap("wan"), Target: "192.0.2.1",
		Scope: api.SelectedMap("vhid_group"), VHID: "0", FailureAdvSkew: "200",
		VHIDTargets:        api.SelectedMapList{"opt2:51", "opt3:52"},
		FallbackIPv4Target: "192.0.2.2", FallbackIPv4Gateway: "10.16.224.5",
		FallbackIPv6Target: "2001:db8:1::2", FallbackIPv6Gateway: "2001:db8:2::1",
	}
	id, err := controller.AddCarpHealthCheck(ctx, check)
	if err != nil || id != uuid {
		t.Fatalf("AddCarpHealthCheck() = %q, %v", id, err)
	}
	retrieved, err := controller.GetCarpHealthCheck(ctx, uuid)
	if err != nil || retrieved.Name != "leaf" || retrieved.Enabled != "1" || retrieved.Interface.String() != "opt2" ||
		retrieved.Scope.String() != "interface" || retrieved.VHID != "0" || retrieved.FailureAdvSkew != "254" || len(retrieved.VHIDTargets) != 0 {
		t.Fatalf("GetCarpHealthCheck() = %#v, %v", retrieved, err)
	}
	search, err := controller.SearchCarpHealthCheck(ctx)
	if err != nil || search.Total != 1 || search.Rows[0].Enabled != "1" || search.Rows[0].Scope.String() != "interface" || search.Rows[0].VHID != "0" {
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
