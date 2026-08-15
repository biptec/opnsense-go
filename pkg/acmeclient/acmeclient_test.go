package acmeclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func acmeSelected(value string) map[string]any {
	return map[string]any{value: map[string]any{"value": value, "selected": true}}
}

func TestSettingsAndActionsWireContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/acmeclient/settings/get":
			_ = json.NewEncoder(w).Encode(map[string]any{"acmeclient": map[string]any{"settings": map[string]any{
				"enabled": "1", "autoRenewal": "1", "environment": acmeSelected("stg"),
				"challengePort": "43580", "TLSchallengePort": "43581", "restartTimeout": "600",
				"haproxyIntegration": "0", "logLevel": acmeSelected("normal"), "showIntro": "0",
			}}})
		case "/api/acmeclient/settings/set":
			if r.Method != http.MethodPost {
				t.Fatalf("settings set method = %s", r.Method)
			}
			var body map[string]SettingsRoot
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			settings := body["acmeclient"].Settings
			if settings.Environment.String() != "stg" {
				t.Fatalf("unexpected settings: %+v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved"})
		case "/api/acmeclient/accounts/register/account-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"response": "registered"})
		case "/api/acmeclient/certificates/sign/cert-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"response": "sign started"})
		case "/api/acmeclient/certificates/automation/cert-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"response": "automation started"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	ctx := context.Background()
	got, err := c.SettingsGet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.AcmeClient.Settings.Environment.String() != "stg" {
		t.Fatalf("environment = %q", got.AcmeClient.Settings.Environment)
	}
	if _, err := c.SettingsSet(ctx, &got.AcmeClient); err != nil {
		t.Fatal(err)
	}
	if _, err := c.AccountRegister(ctx, "account-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CertificateSign(ctx, "cert-id"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CertificateAutomation(ctx, "cert-id"); err != nil {
		t.Fatal(err)
	}
}

func TestValidationCarriesRFC2136KeyOnlyOnWrite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/acmeclient/validations/add":
			var body map[string]Validation
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			v := body["validation"]
			if v.DNSService.String() != "dns_nsupdate" || v.DNSNsupdateKey != "runtime-key" {
				t.Fatalf("unexpected validation: %+v", v)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "saved", "uuid": "validation-id"})
		case "/api/acmeclient/validations/get/validation-id":
			_ = json.NewEncoder(w).Encode(map[string]any{"validation": map[string]any{
				"enabled": "1", "name": "endpoint", "method": acmeSelected("dns01"), "dns_service": acmeSelected("dns_nsupdate"),
				"dns_nsupdate_server": "10.16.16.53", "dns_nsupdate_zone": "acme.biptec.net", "dns_nsupdate_key": "runtime-key",
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := &Controller{Api: api.NewClient(api.Options{Uri: server.URL})}
	id, err := c.AddValidation(context.Background(), &Validation{
		Enabled: "1", Name: "endpoint", Method: api.SelectedMap("dns01"), DNSService: api.SelectedMap("dns_nsupdate"),
		DNSNsupdateServer: "10.16.16.53", DNSNsupdateZone: "acme.biptec.net", DNSNsupdateKey: "runtime-key",
	})
	if err != nil || id != "validation-id" {
		t.Fatalf("AddValidation() = %q, %v", id, err)
	}
}
