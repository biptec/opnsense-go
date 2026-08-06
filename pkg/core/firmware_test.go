package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func newFirmwareTestController(t *testing.T) (*Controller, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/core/firmware/info":
			if r.Method != http.MethodGet {
				t.Errorf("info method = %q, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"product_id":      "opnsense",
				"product_version": "26.7.1",
				"plugin": []map[string]string{{
					"name":       "os-bind",
					"version":    "1.36",
					"installed":  "1",
					"provided":   "1",
					"configured": "1",
				}},
			})
		case "/api/core/firmware/install/os-caddy":
			if r.Method != http.MethodPost {
				t.Errorf("install method = %q, want POST", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "msg_uuid": "install-uuid"})
		case "/api/core/firmware/remove/os-caddy":
			if r.Method != http.MethodPost {
				t.Errorf("remove method = %q, want POST", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "msg_uuid": "remove-uuid"})
		case "/api/core/firmware/running":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "0"})
		case "/api/core/firmware/upgradestatus":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "done", "log": "***DONE***"})
		default:
			http.NotFound(w, r)
		}
	}))

	client := api.NewClient(api.Options{Uri: server.URL})
	return &Controller{Api: client}, server
}

func TestFirmwareInfo(t *testing.T) {
	controller, server := newFirmwareTestController(t)
	defer server.Close()

	result, err := controller.FirmwareInfo(context.Background())
	if err != nil {
		t.Fatalf("FirmwareInfo() error = %v", err)
	}
	if result.ProductVersion != "26.7.1" {
		t.Fatalf("ProductVersion = %q, want 26.7.1", result.ProductVersion)
	}
	if len(result.Plugins) != 1 || result.Plugins[0].Name != "os-bind" || result.Plugins[0].Installed != "1" {
		t.Fatalf("unexpected plugin response: %#v", result.Plugins)
	}
}

func TestFirmwarePackageActions(t *testing.T) {
	controller, server := newFirmwareTestController(t)
	defer server.Close()
	ctx := context.Background()

	installed, err := controller.FirmwareInstall(ctx, "os-caddy")
	if err != nil {
		t.Fatalf("FirmwareInstall() error = %v", err)
	}
	if installed.Status != "ok" || installed.MessageUUID != "install-uuid" {
		t.Fatalf("unexpected install result: %#v", installed)
	}

	removed, err := controller.FirmwareRemove(ctx, "os-caddy")
	if err != nil {
		t.Fatalf("FirmwareRemove() error = %v", err)
	}
	if removed.Status != "ok" || removed.MessageUUID != "remove-uuid" {
		t.Fatalf("unexpected remove result: %#v", removed)
	}
}

func TestFirmwareOperationStatus(t *testing.T) {
	controller, server := newFirmwareTestController(t)
	defer server.Close()
	ctx := context.Background()

	running, err := controller.FirmwareRunning(ctx)
	if err != nil {
		t.Fatalf("FirmwareRunning() error = %v", err)
	}
	if running.Status != "0" {
		t.Fatalf("running status = %q, want 0", running.Status)
	}

	status, err := controller.FirmwareUpgradeStatus(ctx)
	if err != nil {
		t.Fatalf("FirmwareUpgradeStatus() error = %v", err)
	}
	if status.Status != "done" || status.Log != "***DONE***" {
		t.Fatalf("unexpected upgrade status: %#v", status)
	}
}
