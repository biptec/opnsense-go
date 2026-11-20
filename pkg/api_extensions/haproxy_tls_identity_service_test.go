package api_extensions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestHAProxyTLSIdentityServicePreflight(t *testing.T) {
	var requestBody map[string]map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/api_extensions/haproxy_tls_identity/preflight" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","count":1,"message":"FreeIPA prerequisites are ready"}`))
	}))
	defer server.Close()

	client := api.NewClient(api.Options{
		Uri:            server.URL,
		APIKey:         "test-key",
		APISecret:      "test-secret",
		RequestTimeout: 5,
	})
	controller := &Controller{Api: client}
	identity := &HAProxyTLSIdentity{
		FrontendName: "arcturus-shared-https",
		DNSName:      "s3.central.sandbox.biptec.net",
		PolicyID:     api.SelectedMap("endpoint"),
	}

	result, err := controller.HAProxyTLSIdentityServicePreflight(context.Background(), identity)
	if err != nil {
		t.Fatalf("preflight call failed: %v", err)
	}
	if result.Status != "ok" || result.Count != 1 {
		t.Fatalf("unexpected preflight result: %#v", result)
	}
	if result.Message != "FreeIPA prerequisites are ready" {
		t.Fatalf("unexpected preflight message: %q", result.Message)
	}
	posted, ok := requestBody["identity"]
	if !ok {
		t.Fatalf("identity body parameter missing: %#v", requestBody)
	}
	if posted["frontend_name"] != identity.FrontendName ||
		posted["dns_name"] != identity.DNSName ||
		posted["policy_id"] != "endpoint" {
		t.Fatalf("unexpected identity body: %#v", posted)
	}
}
