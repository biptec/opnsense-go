package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEndpointValidate(t *testing.T) {
	tests := []struct {
		name     string
		endpoint Endpoint
		wantErr  bool
	}{
		{name: "valid", endpoint: Endpoint{Path: "/service/get", Method: http.MethodGet}},
		{name: "missing path", endpoint: Endpoint{Method: http.MethodGet}, wantErr: true},
		{name: "relative path", endpoint: Endpoint{Path: "service/get", Method: http.MethodGet}, wantErr: true},
		{name: "missing method", endpoint: Endpoint{Path: "/service/get"}, wantErr: true},
		{name: "lowercase method", endpoint: Endpoint{Path: "/service/get", Method: "get"}, wantErr: true},
		{name: "invalid method token", endpoint: Endpoint{Path: "/service/get", Method: "GET-POST"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.endpoint.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestEndpointWithPathSegmentEscapesValue(t *testing.T) {
	endpoint := Endpoint{Path: "/service/item", Method: http.MethodGet}
	got := endpoint.WithPathSegment("name/with space")
	if got.Path != "/service/item/name%2Fwith%20space" {
		t.Fatalf("WithPathSegment() path = %q", got.Path)
	}
	if endpoint.Path != "/service/item" {
		t.Fatalf("WithPathSegment() mutated receiver: %q", endpoint.Path)
	}
}

func TestDoEndpointRequestUsesConfiguredMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if r.URL.Path != "/api/service/item" {
			t.Errorf("path = %q, want /api/service/item", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"result": "ok"})
	}))
	defer server.Close()

	client := NewClient(Options{Uri: server.URL})
	var response map[string]string
	err := client.doEndpointRequest(
		context.Background(),
		Endpoint{Path: "/service/item", Method: http.MethodPatch},
		nil,
		&response,
	)
	if err != nil {
		t.Fatalf("doEndpointRequest() error = %v", err)
	}
}
