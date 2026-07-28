package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientEndpointURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseURI  string
		endpoint string
		want     string
	}{
		{name: "standard", baseURI: "https://firewall.example", endpoint: "/interfaces/assignment/get_item/lan", want: "https://firewall.example/api/interfaces/assignment/get_item/lan"},
		{name: "trailing base slash", baseURI: "https://firewall.example/", endpoint: "/interfaces/vlan_settings/getItem/id", want: "https://firewall.example/api/interfaces/vlan_settings/getItem/id"},
		{name: "endpoint without slash", baseURI: "https://firewall.example", endpoint: "interfaces/overview/interfaces_info", want: "https://firewall.example/api/interfaces/overview/interfaces_info"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(Options{Uri: test.baseURI})
			if got := client.endpointURL(test.endpoint); got != test.want {
				t.Fatalf("endpointURL() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestClientRequestAddsAPIPrefix(t *testing.T) {
	t.Parallel()

	const key = "test-key"
	const secret = "test-secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/interfaces/assignment/get_item/lan" {
			t.Errorf("request path = %q, want %q", r.URL.Path, "/api/interfaces/assignment/get_item/lan")
		}
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(key+":"+secret))
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Errorf("Authorization = %q, want %q", got, wantAuth)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"result": "ok"}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(Options{Uri: server.URL, APIKey: key, APISecret: secret})
	var response map[string]string
	if err := client.doRequest(context.Background(), http.MethodGet, "/interfaces/assignment/get_item/lan", nil, &response); err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	if response["result"] != "ok" {
		t.Fatalf("response result = %q, want %q", response["result"], "ok")
	}
}
