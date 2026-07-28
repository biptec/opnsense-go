package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestInterfaceResourceEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		opts       api.ReqOpts
		controller string
		monad      string
	}{
		{"assignment", AssignmentOpts, "assignment", "interface"},
		{"vlan", VlanOpts, "vlan_settings", "vlan"},
		{"vxlan", VxlanOpts, "vxlan_settings", "vxlan"},
		{"bridge", BridgeOpts, "bridge_settings", "bridge"},
		{"lagg", LaggOpts, "lagg_settings", "lagg"},
		{"gre", GreOpts, "gre_settings", "gre"},
		{"gif", GifOpts, "gif_settings", "gif"},
		{"loopback", LoopbackOpts, "loopback_settings", "loopback"},
		{"neighbor", NeighborOpts, "neighbor_settings", "neighbor"},
		{"vip", VipOpts, "vip_settings", "vip"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			base := "/interfaces/" + test.controller
			want := api.ReqOpts{
				AddEndpoint: base + "/add_item", GetEndpoint: base + "/get_item",
				UpdateEndpoint: base + "/set_item", DeleteEndpoint: base + "/del_item",
				SearchEndpoint: base + "/search_item", ReconfigureEndpoint: base + "/reconfigure",
				Monad: test.monad,
			}
			if !reflect.DeepEqual(test.opts, want) {
				t.Fatalf("options = %#v, want %#v", test.opts, want)
			}
		})
	}
}

func TestAssignmentGetDecodesExtendedModel(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/interfaces/assignment/get_item/lan" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"interface": map[string]any{
			"identifier": "lan", "descr": "LAN", "if": selected("vtnet1"),
			"enable": "1", "blockpriv": "1", "type": selected("static"),
			"ipaddr": "10.10.10.1", "subnet": "24", "type6": selected("track6"),
			"track6-interface": "wan", "track6-prefix-id": "1",
		}})
	}))
	defer server.Close()

	controller := Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	got, err := controller.GetAssignment(context.Background(), "lan")
	if err != nil {
		t.Fatalf("GetAssignment() error = %v", err)
	}
	if got.Identifier != "lan" || got.Device.String() != "vtnet1" || got.IPv4Mode.String() != "static" || got.IPv6Mode.String() != "track6" {
		t.Fatalf("GetAssignment() = %#v", got)
	}
}

func TestBridgeGetDecodesSelectedLists(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"bridge": map[string]any{
			"bridgeif": "bridge0", "members": selectedMany("lan", "opt1"),
			"proto": selected("rstp"), "edge": selectedMany("lan"), "descr": "LAN bridge",
		}})
	}))
	defer server.Close()

	controller := Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	got, err := controller.GetBridge(context.Background(), "bridge-id")
	if err != nil {
		t.Fatalf("GetBridge() error = %v", err)
	}
	if got.Device != "bridge0" || !reflect.DeepEqual([]string(got.Members), []string{"lan", "opt1"}) || got.Protocol.String() != "rstp" {
		t.Fatalf("GetBridge() = %#v", got)
	}
}

func TestSettingsSetAndReconfigure(t *testing.T) {
	t.Parallel()

	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/interfaces/settings/set":
			var body map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if body["settings"]["disableipv6"] != "1" {
				t.Errorf("settings body = %#v", body)
				http.Error(w, "invalid settings", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved"})
		case "/api/interfaces/settings/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
	}))
	defer server.Close()

	controller := Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	if _, err := controller.SettingsSet(context.Background(), &InterfaceSettings{DisableIPv6: "1"}); err != nil {
		t.Fatalf("SettingsSet() error = %v", err)
	}
	if _, err := controller.SettingsReconfigure(context.Background()); err != nil {
		t.Fatalf("SettingsReconfigure() error = %v", err)
	}
	if !reflect.DeepEqual(paths, []string{"/api/interfaces/settings/set", "/api/interfaces/settings/reconfigure"}) {
		t.Fatalf("paths = %#v", paths)
	}
}

func selected(key string) map[string]any {
	return map[string]any{key: map[string]any{"selected": 1, "value": key}}
}

func selectedMany(keys ...string) map[string]any {
	result := make(map[string]any, len(keys))
	for _, key := range keys {
		result[key] = map[string]any{"selected": 1, "value": key}
	}
	return result
}

func TestAddAssignmentResolved(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/interfaces/assignment/add_item":
			_ = json.NewEncoder(w).Encode(map[string]string{"result": "saved", "uuid": "temporary-id"})
		case "/api/interfaces/assignment/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/api/interfaces/assignment/search_item":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"total": 1, "rowCount": 1, "current": 1,
				"rows": []map[string]any{{"identifier": "opt1", "if": "vtnet2", "descr": "Transit"}},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	identifier, err := controller.AddAssignmentResolved(context.Background(), &Assignment{
		Device: api.SelectedMap("vtnet2"), Description: "Transit",
	})
	if err != nil {
		t.Fatalf("AddAssignmentResolved() error = %v", err)
	}
	if identifier != "opt1" {
		t.Fatalf("identifier = %q, want opt1", identifier)
	}
}
