package routes

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

func routeTestController(server *httptest.Server) *Controller {
	return &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
}

func TestAddRouteResolvedRetriesStaleGatewayOptions(t *testing.T) {
	var addCalls, searchCalls, reconfigureCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routes/routes/addroute":
			addCalls++
			if addCalls < 3 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"result": "failed",
					"validations": map[string]any{
						"route.gateway": "Specify a valid gateway from the list matching the networks ip protocol.",
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "route-id"})
		case "/api/routing/settings/searchGateway":
			searchCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"name": "GW_A", "ipprotocol": "inet", "disabled": false}},
			})
		case "/api/routes/routes/reconfigure":
			reconfigureCalls++
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	route := &Route{Gateway: api.SelectedMap("GW_A"), Network: "192.0.2.0/24", Enabled: "1"}
	id, err := routeTestController(server).addRouteResolved(context.Background(), route, 0, time.Second)
	if err != nil {
		t.Fatalf("addRouteResolved(): %v", err)
	}
	if id != "route-id" || addCalls != 3 || searchCalls != 1 || reconfigureCalls != 1 {
		t.Fatalf("id=%q add=%d search=%d reconfigure=%d", id, addCalls, searchCalls, reconfigureCalls)
	}
}

func TestAddRouteResolvedDoesNotRetryMissingGateway(t *testing.T) {
	var addCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routes/routes/addroute":
			addCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result":      "failed",
				"validations": map[string]any{"route.gateway": "Specify a valid gateway from the list matching the networks ip protocol."},
			})
		case "/api/routing/settings/searchGateway":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"name": "GW_OTHER", "ipprotocol": "inet", "disabled": false}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	route := &Route{Gateway: api.SelectedMap("GW_MISSING"), Network: "192.0.2.0/24", Enabled: "1"}
	_, err := routeTestController(server).addRouteResolved(context.Background(), route, 0, time.Second)
	if err == nil || !strings.Contains(err.Error(), "route.gateway") {
		t.Fatalf("error = %v, want gateway validation", err)
	}
	if addCalls != 1 {
		t.Fatalf("add calls = %d, want 1", addCalls)
	}
}

func TestAddRouteResolvedDoesNotRetryWrongFamily(t *testing.T) {
	var addCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routes/routes/addroute":
			addCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result":      "failed",
				"validations": map[string]any{"route.gateway": "Specify a valid gateway from the list matching the networks ip protocol."},
			})
		case "/api/routing/settings/searchGateway":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"name": "GW_V4", "ipprotocol": "inet", "disabled": false}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	route := &Route{Gateway: api.SelectedMap("GW_V4"), Network: "2001:db8::/32", Enabled: "1"}
	_, err := routeTestController(server).addRouteResolved(context.Background(), route, 0, time.Second)
	if err == nil || !strings.Contains(err.Error(), "route.gateway") {
		t.Fatalf("error = %v, want family validation", err)
	}
	if addCalls != 1 {
		t.Fatalf("add calls = %d, want 1", addCalls)
	}
}

func TestUpdateRouteResolvedRetriesStaleGatewayOptions(t *testing.T) {
	var updateCalls, searchCalls, reconfigureCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routes/routes/setroute/route-id":
			updateCalls++
			if updateCalls == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"result":      "failed",
					"validations": map[string]any{"route.gateway": "Specify a valid gateway from the list matching the networks ip protocol."},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "route-id"})
		case "/api/routing/settings/searchGateway":
			searchCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"name": "GW_A", "ipprotocol": "inet", "disabled": false}},
			})
		case "/api/routes/routes/reconfigure":
			reconfigureCalls++
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	route := &Route{Gateway: api.SelectedMap("GW_A"), Network: "192.0.2.0/24", Enabled: "1"}
	err := routeTestController(server).updateRouteResolved(context.Background(), "route-id", route, 0, time.Second)
	if err != nil {
		t.Fatalf("updateRouteResolved(): %v", err)
	}
	if updateCalls != 2 || searchCalls != 1 || reconfigureCalls != 1 {
		t.Fatalf("update=%d search=%d reconfigure=%d", updateCalls, searchCalls, reconfigureCalls)
	}
}

func TestAddRouteResolvedWaitsForGatewaySearchPropagation(t *testing.T) {
	var addCalls, searchCalls, reconfigureCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routes/routes/addroute":
			addCalls++
			if addCalls == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"result": "failed",
					"validations": map[string]any{
						"route.gateway": "Specify a valid gateway from the list matching the networks ip protocol.",
					},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "route-id"})
		case "/api/routing/settings/searchGateway":
			searchCalls++
			rows := []map[string]any{}
			if searchCalls > 1 {
				rows = append(rows, map[string]any{
					"name": "GW_A", "gateway": "192.0.2.1", "disabled": false,
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": len(rows), "rowCount": len(rows), "current": 1, "rows": rows,
			})
		case "/api/routes/routes/reconfigure":
			reconfigureCalls++
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	route := &Route{Gateway: api.SelectedMap("GW_A"), Network: "192.0.2.0/24", Enabled: "1"}
	settings := routeGatewayRetrySettings{
		routeTimeout:  time.Second,
		verifyTimeout: time.Second,
	}
	id, err := routeTestController(server).addRouteResolvedWithSettings(context.Background(), route, settings)
	if err != nil {
		t.Fatalf("addRouteResolvedWithSettings(): %v", err)
	}
	if id != "route-id" || addCalls != 2 || searchCalls != 2 || reconfigureCalls != 1 {
		t.Fatalf("id=%q add=%d search=%d reconfigure=%d", id, addCalls, searchCalls, reconfigureCalls)
	}
}
