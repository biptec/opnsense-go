package firewall

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func firewallTestController(server *httptest.Server) *Controller {
	return &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
}

func selectedFirewallValue(value string) map[string]any {
	return map[string]any{value: map[string]any{"value": value, "selected": true}}
}

func TestGroupGetAndSearch(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/firewall/group/getItem/group-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"group": map[string]any{
				"ifname": "TEST_GROUP", "members": selectedFirewallValue("lan"),
				"nogroup": "0", "sequence": "10", "descr": "test",
			}})
		case "/api/firewall/group/searchItem":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{
					"uuid": "group-id", "ifname": "TEST_GROUP", "members": "lan,opt1",
					"nogroup": "0", "sequence": "10", "descr": "test",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := firewallTestController(server)
	group, err := controller.GetGroup(context.Background(), "group-id")
	if err != nil {
		t.Fatalf("GetGroup() error = %v", err)
	}
	if group.Name != "TEST_GROUP" || group.Members.String() != "lan" {
		t.Fatalf("unexpected group: %+v", group)
	}
	result, err := controller.SearchGroup(context.Background())
	if err != nil {
		t.Fatalf("SearchGroup() error = %v", err)
	}
	if result.Total != 1 || result.Rows[0].UUID != "group-id" || result.Rows[0].Members.String() != "lan,opt1" {
		t.Fatalf("unexpected group search: %+v", result)
	}
}

func TestNptGetAndSearch(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/firewall/npt/getRule/npt-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"rule": map[string]any{
				"enabled": "1", "log": "0", "sequence": "10", "categories": map[string]any{},
				"description": "test", "interface": selectedFirewallValue("lan"),
				"source_net": "fd00:1::/48", "destination_net": "2001:db8:1::/48",
				"trackif": map[string]any{},
			}})
		case "/api/firewall/npt/searchRule":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{
					"uuid": "npt-id", "enabled": "1", "sequence": "10", "interface": "lan",
					"source_net": "fd00:1::/48", "destination_net": "2001:db8:1::/48",
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := firewallTestController(server)
	npt, err := controller.GetNpt(context.Background(), "npt-id")
	if err != nil {
		t.Fatalf("GetNpt() error = %v", err)
	}
	if npt.Interface.String() != "lan" || npt.SourceNet != "fd00:1::/48" || npt.DestinationNet != "2001:db8:1::/48" {
		t.Fatalf("unexpected NPT rule: %+v", npt)
	}
	result, err := controller.SearchNpt(context.Background())
	if err != nil {
		t.Fatalf("SearchNpt() error = %v", err)
	}
	if result.Total != 1 || result.Rows[0].UUID != "npt-id" || result.Rows[0].SourceNet != "fd00:1::/48" {
		t.Fatalf("unexpected NPT search: %+v", result)
	}
}
