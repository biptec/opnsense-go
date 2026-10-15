package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestTimeoutBoundsRetryLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := NewClient(Options{
		Uri:            server.URL,
		APIKey:         "key",
		APISecret:      "secret",
		RequestTimeout: 1,
	})
	started := time.Now()
	err := client.doEndpointRequest(
		context.Background(),
		Endpoint{Path: "/slow", Method: http.MethodGet},
		nil,
		&map[string]any{},
	)
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected context deadline error, got %v", err)
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("request exceeded bounded timeout: %s", elapsed)
	}
}
