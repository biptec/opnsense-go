package quagga

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestOSPFAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/quagga/general/get" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"general": map[string]any{"enabled": "1", "enablecarp": "0"}})
		case r.URL.Path == "/api/quagga/general/set" && r.Method == http.MethodPost:
			var body map[string]FRRGeneralSettings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode general: %v", err)
			}
			if body["general"].Enabled != "1" || body["general"].EnableCARP != "0" {
				t.Fatalf("unexpected general settings: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved"})
		case r.URL.Path == "/api/quagga/ospfsettings/get" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"ospf": map[string]any{
				"enabled": "1", "carp_demote": "0", "routerid": "10.255.0.1",
			}})
		case r.URL.Path == "/api/quagga/ospf6settings/get" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"ospf6": map[string]any{
				"enabled": "1", "carp_demote": "0", "routerid": "10.255.0.1",
			}})
		case r.URL.Path == "/api/quagga/diagnostics/ospfoverview" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"response": map[string]any{"routerId": "10.255.0.1"}})
		case r.URL.Path == "/api/quagga/diagnostics/searchOspfneighbor" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"neighborid": "10.255.0.2", "state": "Full"}},
			})
		case r.URL.Path == "/api/quagga/diagnostics/searchOspfroute" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"network": "10.16.18.53/32", "via": "10.16.224.2"}},
			})
		case r.URL.Path == "/api/quagga/diagnostics/ospfv3overview" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"response": map[string]any{"routerId": "10.255.0.1"}})
		case r.URL.Path == "/api/quagga/diagnostics/searchOspfv3route" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"network": "2a07:e580:a10:1234::2/128"}},
			})
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	ctx := context.Background()

	general, err := controller.FRRGeneralGet(ctx)
	if err != nil || general.General.Enabled != "1" || general.General.EnableCARP != "0" {
		t.Fatalf("FRRGeneralGet() = %#v, %v", general, err)
	}
	if result, err := controller.FRRGeneralSet(ctx, &FRRGeneralSettings{Enabled: "1", EnableCARP: "0"}); err != nil || result.Result != "saved" {
		t.Fatalf("FRRGeneralSet() = %#v, %v", result, err)
	}

	ospf, err := controller.OSPFSettingsGet(ctx)
	if err != nil || ospf.OSPF.Enabled != "1" || ospf.OSPF.CarpDemote != "0" || ospf.OSPF.RouterID != "10.255.0.1" {
		t.Fatalf("OSPFSettingsGet() = %#v, %v", ospf, err)
	}
	ospf6, err := controller.OSPF6SettingsGet(ctx)
	if err != nil || ospf6.OSPF6.Enabled != "1" || ospf6.OSPF6.CarpDemote != "0" {
		t.Fatalf("OSPF6SettingsGet() = %#v, %v", ospf6, err)
	}
	overview, err := controller.OSPFOverview(ctx)
	if err != nil || overview.Response["routerId"] != "10.255.0.1" {
		t.Fatalf("OSPFOverview() = %#v, %v", overview, err)
	}
	neighbors, err := controller.SearchOSPFNeighbors(ctx)
	if err != nil || len(neighbors.Rows) != 1 || neighbors.Rows[0]["state"] != "Full" {
		t.Fatalf("SearchOSPFNeighbors() = %#v, %v", neighbors, err)
	}
	routes, err := controller.SearchOSPFRoutes(ctx)
	if err != nil || len(routes.Rows) != 1 || routes.Rows[0]["network"] != "10.16.18.53/32" {
		t.Fatalf("SearchOSPFRoutes() = %#v, %v", routes, err)
	}
	overview6, err := controller.OSPFv3Overview(ctx)
	if err != nil || overview6.Response["routerId"] != "10.255.0.1" {
		t.Fatalf("OSPFv3Overview() = %#v, %v", overview6, err)
	}
	routes6, err := controller.SearchOSPFv3Routes(ctx)
	if err != nil || len(routes6.Rows) != 1 || routes6.Rows[0]["network"] != "2a07:e580:a10:1234::2/128" {
		t.Fatalf("SearchOSPFv3Routes() = %#v, %v", routes6, err)
	}
}

func TestOSPFEndpointContracts(t *testing.T) {
	if OSPFInterfaceOpts.Create.Path != "/quagga/ospfsettings/addInterface" || OSPFInterfaceOpts.Reconfigure.Path != "/quagga/service/reconfigure" {
		t.Fatalf("unexpected OSPF interface endpoints: %#v", OSPFInterfaceOpts)
	}
	if OSPF6InterfaceOpts.Create.Path != "/quagga/ospf6settings/addInterface" || OSPF6InterfaceOpts.Reconfigure.Path != "/quagga/service/reconfigure" {
		t.Fatalf("unexpected OSPFv3 interface endpoints: %#v", OSPF6InterfaceOpts)
	}
	if OSPFNetworkOpts.Search.Path != "/quagga/ospfsettings/searchNetwork" || OSPF6NetworkOpts.Search.Path != "/quagga/ospf6settings/searchNetwork" {
		t.Fatalf("unexpected OSPF network endpoints: v2=%#v v3=%#v", OSPFNetworkOpts, OSPF6NetworkOpts)
	}
}
