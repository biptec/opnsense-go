package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestAddGatewayGroupResolvedRetriesStaleOptions(t *testing.T) {
	var addCalls, searchCalls, reconfigureCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routing/group_settings/add":
			addCalls++
			if addCalls < 3 {
				_ = json.NewEncoder(w).Encode(map[string]any{"result": "failed", "validations": map[string]any{"gateway_group.item": "Option [GW_A] not in list."}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "group-id"})
		case "/api/routing/settings/searchGateway":
			searchCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"total": 1, "rowCount": 1, "current": 1, "rows": []map[string]any{{"name": "GW_A"}}})
		case "/api/routing/group_settings/reconfigure":
			reconfigureCalls++
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	group := &GatewayGroup{Name: "GROUP", Tier1: api.SelectedMapList{"GW_A"}}
	id, err := testController(server).addGatewayGroupResolved(context.Background(), group, 0, time.Second)
	if err != nil {
		t.Fatalf("addGatewayGroupResolved(): %v", err)
	}
	if id != "group-id" || addCalls != 3 || searchCalls != 1 || reconfigureCalls != 1 {
		t.Fatalf("id=%q add=%d search=%d reconfigure=%d", id, addCalls, searchCalls, reconfigureCalls)
	}
}

func TestAddGatewayGroupResolvedDoesNotRetryMissingGateway(t *testing.T) {
	var addCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routing/group_settings/add":
			addCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "failed", "validations": map[string]any{"gateway_group.item": "Option [GW_MISSING] not in list."}})
		case "/api/routing/settings/searchGateway":
			_ = json.NewEncoder(w).Encode(map[string]any{"total": 1, "rowCount": 1, "current": 1, "rows": []map[string]any{{"name": "GW_OTHER"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	group := &GatewayGroup{Name: "GROUP", Tier1: api.SelectedMapList{"GW_MISSING"}}
	_, err := testController(server).addGatewayGroupResolved(context.Background(), group, 0, time.Second)
	if err == nil || !strings.Contains(err.Error(), "GW_MISSING") {
		t.Fatalf("error = %v, want missing gateway validation", err)
	}
	if addCalls != 1 {
		t.Fatalf("add calls = %d, want 1", addCalls)
	}
}

func TestAddGatewayGroupResolvedDoesNotRetryUnrelatedOptionError(t *testing.T) {
	var addCalls, searchCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routing/group_settings/add":
			addCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "failed", "validations": map[string]any{"gateway_group.trigger": "Option [invalid] not in list."}})
		case "/api/routing/settings/searchGateway":
			searchCalls++
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	group := &GatewayGroup{Name: "GROUP", Tier1: api.SelectedMapList{"GW_A"}}
	_, err := testController(server).addGatewayGroupResolved(context.Background(), group, 0, time.Second)
	if err == nil || !strings.Contains(err.Error(), "gateway_group.trigger") {
		t.Fatalf("error = %v, want trigger validation", err)
	}
	if addCalls != 1 || searchCalls != 0 {
		t.Fatalf("add calls = %d, search calls = %d; want 1, 0", addCalls, searchCalls)
	}
}
