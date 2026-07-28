package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func testController(server *httptest.Server) *Controller {
	return &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
}

func selected(value string) map[string]any {
	return map[string]any{value: map[string]any{"value": value, "selected": true}}
}

func TestGatewayGetAndSearch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/routing/settings/getGateway/gw-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"gateway_item": map[string]any{
				"name": "GW_TEST", "interface": selected("lan"), "ipprotocol": selected("inet"),
				"gateway": "192.0.2.1", "priority": "200", "weight": "1",
			}})
		case "/api/routing/settings/searchGateway":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{
					"uuid": "gw-id", "name": "GW_TEST", "interface": "lan",
					"ipprotocol": "inet", "gateway": "192.0.2.1",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := testController(server)
	gateway, err := controller.GetGateway(context.Background(), "gw-id")
	if err != nil {
		t.Fatalf("GetGateway() error = %v", err)
	}
	if gateway.Name != "GW_TEST" || gateway.Interface.String() != "lan" || gateway.IPProtocol.String() != "inet" {
		t.Fatalf("unexpected gateway: %+v", gateway)
	}

	result, err := controller.SearchGateway(context.Background())
	if err != nil {
		t.Fatalf("SearchGateway() error = %v", err)
	}
	if result.Total != 1 || len(result.Rows) != 1 || result.Rows[0].UUID != "gw-id" {
		t.Fatalf("unexpected search result: %+v", result)
	}
}

func TestGatewayGroupGet(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/routing/group_settings/get/group-id" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"gateway_group": map[string]any{
			"name": "GW_GROUP", "item": selected("GW_A"), "item2": selected("GW_B"),
			"item3": map[string]any{}, "item4": map[string]any{}, "item5": map[string]any{},
			"trigger": selected("down"), "poolopts": selected("round-robin"), "descr": "test",
		}})
	}))
	defer server.Close()

	group, err := testController(server).GetGatewayGroup(context.Background(), "group-id")
	if err != nil {
		t.Fatalf("GetGatewayGroup() error = %v", err)
	}
	if group.Name != "GW_GROUP" || group.Tier1.String() != "GW_A" || group.Tier2.String() != "GW_B" {
		t.Fatalf("unexpected gateway group: %+v", group)
	}
	if group.Trigger.String() != "down" || group.PoolOptions.String() != "round-robin" {
		t.Fatalf("unexpected gateway group options: %+v", group)
	}
}

func TestGatewayStatusGet(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/routes/gateway/status" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"items": []map[string]any{{
				"name": "WAN_DHCP", "address": "192.0.2.1", "monitor": "192.0.2.1",
				"delay": "1.2ms", "stddev": "0.2ms", "loss": "0.0%",
				"status": "none", "status_translated": "Online",
			}},
		})
	}))
	defer server.Close()

	result, err := testController(server).GatewayStatusGet(context.Background())
	if err != nil {
		t.Fatalf("GatewayStatusGet() error = %v", err)
	}
	if result.Status != "ok" || len(result.Items) != 1 || result.Items[0].Name != "WAN_DHCP" {
		t.Fatalf("unexpected gateway status: %+v", result)
	}
}
