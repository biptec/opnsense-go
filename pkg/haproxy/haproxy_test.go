package haproxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func haproxySelected(value string) map[string]any {
	return map[string]any{value: map[string]any{"value": value, "selected": 1}}
}

func haproxySelectedList(values ...string) map[string]any {
	out := make(map[string]any, len(values))
	for _, value := range values {
		out[value] = map[string]any{"value": value, "selected": 1}
	}
	return out
}

func TestHAProxySettingsAndServiceContracts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/haproxy/settings/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"haproxy": map[string]any{"general": map[string]any{
				"enabled": "1", "showIntro": "0", "gracefulStop": "1", "hardStopAfter": "60s",
				"closeSpreadTime": "5s", "seamlessReload": "1",
			}}})
		case "/api/haproxy/settings/set":
			var body map[string]Settings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode settings body: %v", err)
			}
			if body["haproxy"].General.Enabled != "1" || body["haproxy"].General.ShowIntro != "0" || body["haproxy"].General.SeamlessReload != "1" {
				t.Fatalf("unexpected HAProxy settings body: %+v", body["haproxy"].General)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved"})
		case "/api/haproxy/service/configtest":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "Configuration file is valid"})
		case "/api/haproxy/service/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	ctx := context.Background()
	settings, err := controller.SettingsGet(ctx)
	if err != nil {
		t.Fatalf("SettingsGet() error = %v", err)
	}
	if settings.HAProxy.General.Enabled != "1" || settings.HAProxy.General.ShowIntro != "0" || settings.HAProxy.General.CloseSpreadTime != "5s" {
		t.Fatalf("unexpected settings response: %+v", settings.HAProxy.General)
	}
	if _, err := controller.SettingsSet(ctx, &settings.HAProxy); err != nil {
		t.Fatalf("SettingsSet() error = %v", err)
	}
	result, err := controller.ServiceConfigtest(ctx)
	if err != nil || result.Result != "Configuration file is valid" {
		t.Fatalf("ServiceConfigtest() = %+v, %v", result, err)
	}
}

func TestHAProxyL4SNIResourceContracts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/haproxy/settings/add_frontend":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "fe-id"})
		case "/api/haproxy/settings/get_frontend/fe-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"frontend": map[string]any{
				"enabled": "1", "name": "tls_ingress", "bind": haproxySelectedList("10.0.0.10:443"),
				"mode": haproxySelected("tcp"), "linkedActions": haproxySelectedList("act-id"),
			}})
		case "/api/haproxy/settings/add_backend":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "be-id"})
		case "/api/haproxy/settings/get_backend/be-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"backend": map[string]any{
				"enabled": "1", "name": "web_rigi", "mode": haproxySelected("tcp"),
				"algorithm": haproxySelected("roundrobin"), "linkedServers": haproxySelectedList("srv-id"),
			}})
		case "/api/haproxy/settings/add_server":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "srv-id"})
		case "/api/haproxy/settings/get_server/srv-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"server": map[string]any{
				"enabled": "1", "name": "rigi", "address": "10.0.0.20", "port": "443",
				"mode": haproxySelected("active"), "type": haproxySelected("static"),
			}})
		case "/api/haproxy/settings/add_healthcheck":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "hc-id"})
		case "/api/haproxy/settings/get_healthcheck/hc-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"healthcheck": map[string]any{
				"name": "tcp_tls", "type": haproxySelected("tcp"), "interval": "5s",
			}})
		case "/api/haproxy/settings/add_acl":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "acl-id"})
		case "/api/haproxy/settings/get_acl/acl-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"acl": map[string]any{
				"name": "sni_rigi", "expression": haproxySelected("ssl_fc_sni"), "ssl_fc_sni": "web.rigi.example.test",
			}})
		case "/api/haproxy/settings/add_action":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "act-id"})
		case "/api/haproxy/settings/get_action/act-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"action": map[string]any{
				"enabled": "1", "name": "route_rigi", "testType": haproxySelected("if"),
				"linkedAcls": haproxySelectedList("acl-id"), "operator": haproxySelected("and"),
				"type": haproxySelected("use_backend"), "use_backend": haproxySelected("be-id"),
			}})
		case "/api/haproxy/service/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	ctx := context.Background()
	srvID, err := controller.AddServer(ctx, &Server{Enabled: "1", Name: "rigi", Address: "10.0.0.20", Port: "443", Mode: api.SelectedMap("active"), Type: api.SelectedMap("static")})
	if err != nil || srvID != "srv-id" {
		t.Fatalf("AddServer() = %q, %v", srvID, err)
	}
	srv, err := controller.GetServer(ctx, srvID)
	if err != nil || srv.Address != "10.0.0.20" || srv.Mode.String() != "active" {
		t.Fatalf("GetServer() = %+v, %v", srv, err)
	}

	hcID, err := controller.AddHealthcheck(ctx, &Healthcheck{Name: "tcp_tls", Type: api.SelectedMap("tcp"), Interval: "5s"})
	if err != nil || hcID != "hc-id" {
		t.Fatalf("AddHealthcheck() = %q, %v", hcID, err)
	}
	hc, err := controller.GetHealthcheck(ctx, hcID)
	if err != nil || hc.Type.String() != "tcp" {
		t.Fatalf("GetHealthcheck() = %+v, %v", hc, err)
	}

	beID, err := controller.AddBackend(ctx, &Backend{Enabled: "1", Name: "web_rigi", Mode: api.SelectedMap("tcp"), Algorithm: api.SelectedMap("roundrobin"), LinkedServers: api.SelectedMapList{srvID}})
	if err != nil || beID != "be-id" {
		t.Fatalf("AddBackend() = %q, %v", beID, err)
	}
	backend, err := controller.GetBackend(ctx, beID)
	if err != nil || backend.LinkedServers.String() != "srv-id" {
		t.Fatalf("GetBackend() = %+v, %v", backend, err)
	}

	aclID, err := controller.AddACL(ctx, &ACL{Name: "sni_rigi", Expression: api.SelectedMap("ssl_fc_sni"), SSLFCSNI: "web.rigi.example.test"})
	if err != nil || aclID != "acl-id" {
		t.Fatalf("AddACL() = %q, %v", aclID, err)
	}
	acl, err := controller.GetACL(ctx, aclID)
	if err != nil || acl.Expression.String() != "ssl_fc_sni" {
		t.Fatalf("GetACL() = %+v, %v", acl, err)
	}

	actionID, err := controller.AddAction(ctx, &Action{Enabled: "1", Name: "route_rigi", TestType: api.SelectedMap("if"), LinkedACLs: api.SelectedMapList{aclID}, Operator: api.SelectedMap("and"), Type: api.SelectedMap("use_backend"), UseBackend: api.SelectedMap(beID)})
	if err != nil || actionID != "act-id" {
		t.Fatalf("AddAction() = %q, %v", actionID, err)
	}
	action, err := controller.GetAction(ctx, actionID)
	if err != nil || action.UseBackend.String() != "be-id" {
		t.Fatalf("GetAction() = %+v, %v", action, err)
	}

	feID, err := controller.AddFrontend(ctx, &Frontend{Enabled: "1", Name: "tls_ingress", Bind: api.SelectedMapList{"10.0.0.10:443"}, Mode: api.SelectedMap("tcp"), LinkedActions: api.SelectedMapList{actionID}})
	if err != nil || feID != "fe-id" {
		t.Fatalf("AddFrontend() = %q, %v", feID, err)
	}
	frontend, err := controller.GetFrontend(ctx, feID)
	if err != nil || frontend.Mode.String() != "tcp" || frontend.LinkedActions.String() != "act-id" {
		t.Fatalf("GetFrontend() = %+v, %v", frontend, err)
	}
}
