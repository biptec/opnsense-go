package api_extensions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestDnsAPI(t *testing.T) {
	var setSeen bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/api_extensions/dns/get":
			json.NewEncoder(w).Encode(map[string]any{
				"dns": map[string]any{
					"servers":           []string{"10.16.18.53", "10.16.16.53"},
					"allow_override":    false,
					"use_local_service": false,
				},
			})
		case "/api/api_extensions/dns/set":
			setSeen = true
			var body map[string]DnsSettings
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body["dns"].Servers; len(got) != 2 || got[0] != "10.16.18.53" || got[1] != "10.16.16.53" {
				t.Fatalf("unexpected servers: %#v", got)
			}
			json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
		case "/api/api_extensions/dns/reconfigure":
			json.NewEncoder(w).Encode(map[string]any{"status": "ok", "result": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := api.NewClient(api.Options{
		Uri:           server.URL,
		APIKey:        "key",
		APISecret:     "secret",
		AllowInsecure: true,
	})
	controller := &Controller{Api: client}
	ctx := context.Background()

	got, err := controller.DnsGet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.DNS.Servers) != 2 || got.DNS.AllowOverride || got.DNS.UseLocalService {
		t.Fatalf("unexpected DNS response: %#v", got.DNS)
	}

	settings := &DnsSettings{
		Servers:         []string{"10.16.18.53", "10.16.16.53"},
		AllowOverride:   false,
		UseLocalService: false,
	}
	if _, err := controller.DnsSet(ctx, settings); err != nil {
		t.Fatal(err)
	}
	if !setSeen {
		t.Fatal("DNS set endpoint was not called")
	}
	if _, err := controller.DnsReconfigure(ctx); err != nil {
		t.Fatal(err)
	}
}
