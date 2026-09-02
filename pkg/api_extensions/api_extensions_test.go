package api_extensions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func newTestController(t *testing.T) (*Controller, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/api_extensions/webgui/get":
			if r.Method != http.MethodGet {
				t.Errorf("webgui get method = %q, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"webgui": map[string]any{
				"protocol":              "https",
				"port":                  443,
				"interfaces":            []string{"lan"},
				"certificate_ref":       "certificate-reference",
				"session_timeout":       300,
				"hsts":                  true,
				"disable_http_redirect": false,
				"alternate_hostnames":   []string{"router.internal"},
			}})
		case "/api/api_extensions/webgui/set":
			if r.Method != http.MethodPost {
				t.Errorf("webgui set method = %q, want POST", r.Method)
			}
			var body struct {
				Webgui WebguiSettings `json:"webgui"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode webgui body: %v", err)
			}
			if body.Webgui.Protocol != "https" || !reflect.DeepEqual(body.Webgui.Interfaces, []string{"lan"}) {
				t.Errorf("unexpected webgui body: %#v", body.Webgui)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/api_extensions/webgui/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "msg_uuid": "webgui-uuid"})
		case "/api/api_extensions/ssh/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"ssh": map[string]any{
				"enabled":                 true,
				"port":                    22,
				"interfaces":              []string{"lan"},
				"password_authentication": false,
				"permit_root_login":       false,
			}})
		case "/api/api_extensions/ssh/set":
			var body struct {
				SSH SshSettings `json:"ssh"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode ssh body: %v", err)
			}
			if !body.SSH.Enabled || body.SSH.Port != 22 || !reflect.DeepEqual(body.SSH.Interfaces, []string{"lan"}) {
				t.Errorf("unexpected ssh body: %#v", body.SSH)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/api_extensions/ssh/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "result": "restarted"})
		case "/api/api_extensions/ntp/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"ntp": map[string]any{
				"enabled": true,
				"servers": []map[string]any{{
					"host": "time.example.net", "noselect": false, "prefer": true, "iburst": true, "pool": false,
				}},
				"interfaces":             []string{"opt1"},
				"orphan":                 12,
				"max_clock":              10,
				"client_mode":            false,
				"kiss_of_death":          true,
				"rate_limiting":          true,
				"deny_modifications":     true,
				"disable_queries":        true,
				"disable_serving":        false,
				"deny_peer_associations": true,
				"deny_trap_service":      true,
			}})
		case "/api/api_extensions/ntp/set":
			var body struct {
				NTP NtpSettings `json:"ntp"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode ntp body: %v", err)
			}
			if !body.NTP.Enabled || len(body.NTP.Servers) != 1 || body.NTP.Servers[0].Host != "time.example.net" {
				t.Errorf("unexpected ntp body: %#v", body.NTP)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/api_extensions/ntp/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/api_extensions/package/get/os-api-extensions":
			if r.Method != http.MethodGet {
				t.Errorf("package get method = %q, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "ok",
				"package": map[string]any{
					"name":       "os-api-extensions",
					"installed":  true,
					"provided":   true,
					"version":    "0.12",
					"locked":     false,
					"repository": "OPNsense",
					"origin":     "opnsense/os-api-extensions",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	client := api.NewClient(api.Options{Uri: server.URL})
	return &Controller{Api: client}, server
}

func TestWebguiAPI(t *testing.T) {
	controller, server := newTestController(t)
	defer server.Close()
	ctx := context.Background()

	response, err := controller.WebguiGet(ctx)
	if err != nil {
		t.Fatalf("WebguiGet() error = %v", err)
	}
	if response.Webgui.Protocol != "https" || response.Webgui.Port != 443 {
		t.Fatalf("unexpected webgui response: %#v", response.Webgui)
	}

	timeout := 300
	settings := &WebguiSettings{
		Protocol:           "https",
		Port:               443,
		Interfaces:         []string{"lan"},
		CertificateRef:     "certificate-reference",
		SessionTimeout:     &timeout,
		HSTS:               true,
		AlternateHostnames: []string{"router.internal"},
	}
	result, err := controller.WebguiSet(ctx, settings)
	if err != nil || result.Status != "ok" {
		t.Fatalf("WebguiSet() = %#v, %v", result, err)
	}
	result, err = controller.WebguiReconfigure(ctx)
	if err != nil || result.MessageUUID != "webgui-uuid" {
		t.Fatalf("WebguiReconfigure() = %#v, %v", result, err)
	}
}

func TestSshAPI(t *testing.T) {
	controller, server := newTestController(t)
	defer server.Close()
	ctx := context.Background()

	response, err := controller.SshGet(ctx)
	if err != nil {
		t.Fatalf("SshGet() error = %v", err)
	}
	if !response.SSH.Enabled || response.SSH.Port != 22 {
		t.Fatalf("unexpected ssh response: %#v", response.SSH)
	}

	settings := &SshSettings{Enabled: true, Port: 22, Interfaces: []string{"lan"}}
	result, err := controller.SshSet(ctx, settings)
	if err != nil || result.Status != "ok" {
		t.Fatalf("SshSet() = %#v, %v", result, err)
	}
	result, err = controller.SshReconfigure(ctx)
	if err != nil || result.Result != "restarted" {
		t.Fatalf("SshReconfigure() = %#v, %v", result, err)
	}
}

func TestNtpAPI(t *testing.T) {
	controller, server := newTestController(t)
	defer server.Close()
	ctx := context.Background()

	response, err := controller.NtpGet(ctx)
	if err != nil {
		t.Fatalf("NtpGet() error = %v", err)
	}
	if !response.NTP.Enabled || len(response.NTP.Servers) != 1 || response.NTP.Interfaces[0] != "opt1" {
		t.Fatalf("unexpected ntp response: %#v", response.NTP)
	}

	settings := &NtpSettings{
		Enabled:    true,
		Servers:    []NtpServer{{Host: "time.example.net", Prefer: true, IBurst: true}},
		Interfaces: []string{"opt1"},
		Orphan:     12,
		MaxClock:   10,
	}
	result, err := controller.NtpSet(ctx, settings)
	if err != nil || result.Status != "ok" {
		t.Fatalf("NtpSet() = %#v, %v", result, err)
	}
	result, err = controller.NtpReconfigure(ctx)
	if err != nil || result.Status != "ok" {
		t.Fatalf("NtpReconfigure() = %#v, %v", result, err)
	}
}

func TestPackageAPI(t *testing.T) {
	controller, server := newTestController(t)
	defer server.Close()

	response, err := controller.PackageGet(context.Background(), "os-api-extensions")
	if err != nil {
		t.Fatalf("PackageGet() error = %v", err)
	}
	if response.Status != "ok" {
		t.Fatalf("PackageGet() status = %q, want ok", response.Status)
	}
	if response.Package.Name != "os-api-extensions" || !response.Package.Installed || !response.Package.Provided {
		t.Fatalf("unexpected package response: %#v", response.Package)
	}
	if response.Package.Version != "0.12" || response.Package.Locked || response.Package.Repository != "OPNsense" {
		t.Fatalf("unexpected package metadata: %#v", response.Package)
	}
}
