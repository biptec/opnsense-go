package bind

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func bindTestController(server *httptest.Server) *Controller {
	return &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
}

func bindSelected(value string) map[string]any {
	return map[string]any{value: map[string]any{"value": value, "selected": true}}
}

func TestBindSettingsAndServiceContracts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/bind/general/get":
			if r.Method != http.MethodGet {
				t.Fatalf("settings get method = %s, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"general": map[string]any{
				"enabled": "1", "disablev6": "0", "listenv4": bindSelected("10.0.0.1"),
				"listenv6": bindSelected("::1"), "port": "53", "general_log_level": bindSelected("info"),
				"dnssecvalidation": bindSelected("auto"), "ratelimitexcept": bindSelected("127.0.0.1"),
			}})
		case "/api/bind/general/set":
			if r.Method != http.MethodPost {
				t.Fatalf("settings set method = %s, want POST", r.Method)
			}
			var body map[string]GeneralSettings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode settings: %v", err)
			}
			general := body["general"]
			if general.Port != "53" || general.ListenIPv4.String() != "10.0.0.1" || general.DNSSECValidation.String() != "auto" {
				t.Fatalf("unexpected settings body: %+v", general)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved"})
		case "/api/bind/service/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "running"})
		case "/api/bind/service/reconfigure":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	controller := bindTestController(server)
	settings, err := controller.SettingsGet(context.Background())
	if err != nil {
		t.Fatalf("SettingsGet(): %v", err)
	}
	if settings.General.Port != "53" || settings.General.ListenIPv4.String() != "10.0.0.1" {
		t.Fatalf("unexpected settings: %+v", settings.General)
	}
	if _, err := controller.SettingsSet(context.Background(), &settings.General); err != nil {
		t.Fatalf("SettingsSet(): %v", err)
	}
	status, err := controller.ServiceStatus(context.Background())
	if err != nil || status.Status != "running" {
		t.Fatalf("ServiceStatus() = %+v, %v", status, err)
	}
	if _, err := controller.ServiceReconfigure(context.Background()); err != nil {
		t.Fatalf("ServiceReconfigure(): %v", err)
	}
}

func TestBindViewTsigAndDNSSECContracts(t *testing.T) {
	t.Parallel()

	reconfigureCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/bind/view/add_view":
			var body map[string]View
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode view: %v", err)
			}
			gotView := body["view"]
			if gotView.Name != "internal" || gotView.MatchClients.String() != "acl-id" || gotView.MatchDestinations.String() != "destination-acl-id" {
				t.Fatalf("unexpected view body: %+v", gotView)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "view-id"})
		case "/api/bind/view/get_view/view-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"view": map[string]any{
				"enabled": "1", "sequence": "10", "name": "internal", "matchany": "0",
				"matchclients": bindSelected("acl-id"), "matchdestinations": bindSelected("destination-acl-id"), "recursion": "1",
				"allowrecursion": bindSelected("acl-id"), "allowquery": bindSelected("acl-id"),
				"dnssecvalidation": bindSelected("auto"),
			}})
		case "/api/bind/tsig/generate_secret":
			_ = json.NewEncoder(w).Encode(map[string]any{"secret": "dGVzdC1zZWNyZXQ="})
		case "/api/bind/tsig/add_key":
			var body map[string]TsigKey
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode TSIG key: %v", err)
			}
			gotKey := body["key"]
			if gotKey.Name != "acme" || gotKey.Algorithm.String() != "hmac-sha256" {
				t.Fatalf("unexpected TSIG metadata: name=%q algorithm=%q secret_present=%t",
					gotKey.Name, gotKey.Algorithm.String(), gotKey.Secret != "")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "key-id"})
		case "/api/bind/general/dnssec_status":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode DNSSEC request: %v", err)
			}
			if body["zone"] != "example.test" || body["uuid"] != "zone-id" {
				t.Fatalf("unexpected DNSSEC request: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"zone": "example.test", "view": "external", "secure": true, "inline_signing": true,
				"ds_records": []string{"example.test. IN DS 12345 13 2 ABCD"},
				"keys":       []map[string]any{{"file": "Kexample.test.+013+12345.key", "key_tag": "12345", "algorithm": "13"}},
			})
		case "/api/bind/service/reconfigure":
			reconfigureCalls++
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	controller := bindTestController(server)
	viewID, err := controller.AddView(ctx, &View{
		Enabled: "1", Sequence: "10", Name: "internal", MatchClients: api.SelectedMapList{"acl-id"},
		MatchDestinations: api.SelectedMapList{"destination-acl-id"},
		Recursion:         "1", AllowRecursion: api.SelectedMapList{"acl-id"}, AllowQuery: api.SelectedMapList{"acl-id"},
		DNSSECValidation: api.SelectedMap("auto"),
	})
	if err != nil || viewID != "view-id" {
		t.Fatalf("AddView() = %q, %v", viewID, err)
	}
	view, err := controller.GetView(ctx, viewID)
	if err != nil || view.MatchClients.String() != "acl-id" || view.MatchDestinations.String() != "destination-acl-id" || view.DNSSECValidation.String() != "auto" {
		t.Fatalf("GetView() = %+v, %v", view, err)
	}
	secret, err := controller.TSIGGenerateSecret(ctx)
	if err != nil {
		t.Fatalf("TSIGGenerateSecret(): %v", err)
	}
	if secret.Secret == "" {
		t.Fatal("TSIGGenerateSecret() returned an empty secret")
	}
	keyID, err := controller.AddTsigKey(ctx, &TsigKey{
		Enabled: "1", Name: "acme", Algorithm: api.SelectedMap("hmac-sha256"), Secret: secret.Secret,
	})
	if err != nil || keyID != "key-id" {
		t.Fatalf("AddTsigKey() = %q, %v", keyID, err)
	}
	dnssec, err := controller.DNSSECStatus(ctx, "example.test", "zone-id")
	if err != nil || !dnssec.Secure || len(dnssec.DSRecords) != 1 || dnssec.Keys[0].KeyTag != "12345" {
		t.Fatalf("DNSSECStatus() = %+v, %v", dnssec, err)
	}
	if reconfigureCalls != 2 {
		t.Fatalf("reconfigure calls = %d, want 2", reconfigureCalls)
	}
}
