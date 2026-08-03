package caddy

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

func TestSettingsGetAndSet(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/caddy/general/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"caddy": map[string]any{"general": map[string]any{
				"enabled": "1", "HttpPort": "8080", "HttpsPort": "8443", "TlsEmail": "ops@example.test",
				"TlsAutoHttps": selected(""), "TlsDnsProvider": selected(""), "DisableSuperuser": []map[string]any{
					{"value": "root", "selected": 0}, {"value": "www", "selected": 1},
				},
				"GracePeriod": "10", "HttpVersions": map[string]any{
					"h1": map[string]any{"value": "HTTP/1.1", "selected": 1},
					"h2": map[string]any{"value": "HTTP/2", "selected": 1},
					"h3": map[string]any{"value": "HTTP/3", "selected": 0},
				},
				"LogAccessPlainKeep": "10", "AuthToTls": []map[string]any{
					{"value": "http://", "selected": 1}, {"value": "https://", "selected": 0},
				},
			}}})
		case "/api/caddy/general/set":
			var body map[string]Settings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode settings body: %v", err)
			}
			got := body["caddy"].General
			if got.HTTPPort != "8080" || got.HTTPSPort != "8443" || got.RunAsUser.String() != "1" {
				t.Fatalf("unexpected settings body: %+v", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := testController(server)
	settings, err := controller.SettingsGet(context.Background())
	if err != nil {
		t.Fatalf("SettingsGet() error = %v", err)
	}
	general := settings.Caddy.General
	if general.RunAsUser.String() != "1" || general.AuthUpstreamProtocol.String() != "0" {
		t.Fatalf("numeric option fields not decoded: %+v", general)
	}
	if general.HTTPVersions.String() != "h1,h2" {
		t.Fatalf("unexpected HTTP versions: %q", general.HTTPVersions.String())
	}
	if _, err := controller.SettingsSet(context.Background(), &settings.Caddy); err != nil {
		t.Fatalf("SettingsSet() error = %v", err)
	}
}

func TestCaddyCRUDContracts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/caddy/reverse_proxy/add_reverse_proxy":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "domain-id"})
		case "/api/caddy/reverse_proxy/get_reverse_proxy/domain-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"reverse": map[string]any{
				"enabled": "1", "FromDomain": "app.example.test", "DisableTls": []map[string]any{
					{"value": "https://", "selected": 1}, {"value": "http://", "selected": 0},
				}, "CustomCertificate": selected("cert-id"),
			}})
		case "/api/caddy/reverse_proxy/add_handle":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "handler-id"})
		case "/api/caddy/reverse_proxy/get_handle/handler-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"handle": map[string]any{
				"enabled": "1", "reverse": selected("domain-id"), "HandleType": selected("handle"),
				"HandleDirective": selected("reverse_proxy"), "ToDomain": selected("10.0.0.10"),
				"ToPort": "8080", "HttpTls": []map[string]any{
					{"value": "http://", "selected": 1}, {"value": "https://", "selected": 0},
				},
			}})
		case "/api/caddy/reverse_proxy/add_access_list":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "acl-id"})
		case "/api/caddy/reverse_proxy/get_access_list/acl-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"accesslist": map[string]any{
				"accesslistName": "management", "clientIps": selected("10.0.0.0/24"),
				"RequestMatcher": selected("client_ip"),
			}})
		case "/api/caddy/service/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	controller := testController(server)
	domainID, err := controller.AddDomain(ctx, &Domain{Enabled: "1", Domain: "app.example.test", DisableTLS: api.SelectedMap("0")})
	if err != nil || domainID != "domain-id" {
		t.Fatalf("AddDomain() = %q, %v", domainID, err)
	}
	domain, err := controller.GetDomain(ctx, domainID)
	if err != nil || domain.DisableTLS.String() != "0" || domain.CustomCertificate.String() != "cert-id" {
		t.Fatalf("GetDomain() = %+v, %v", domain, err)
	}

	handlerID, err := controller.AddHandler(ctx, &Handler{Domain: api.SelectedMap(domainID), UpstreamDomains: api.SelectedMapList{"10.0.0.10"}})
	if err != nil || handlerID != "handler-id" {
		t.Fatalf("AddHandler() = %q, %v", handlerID, err)
	}
	handler, err := controller.GetHandler(ctx, handlerID)
	if err != nil || handler.UpstreamDomains.String() != "10.0.0.10" || handler.UpstreamProtocol.String() != "0" {
		t.Fatalf("GetHandler() = %+v, %v", handler, err)
	}

	accessID, err := controller.AddAccessList(ctx, &AccessList{Name: "management", ClientIPs: api.SelectedMapList{"10.0.0.0/24"}})
	if err != nil || accessID != "acl-id" {
		t.Fatalf("AddAccessList() = %q, %v", accessID, err)
	}
	access, err := controller.GetAccessList(ctx, accessID)
	if err != nil || access.ClientIPs.String() != "10.0.0.0/24" || access.RequestMatcher.String() != "client_ip" {
		t.Fatalf("GetAccessList() = %+v, %v", access, err)
	}
}
