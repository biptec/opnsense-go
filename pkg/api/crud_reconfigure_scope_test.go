package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
)

type scopedCRUDTestResource struct {
	Name string `json:"name"`
}

func TestAddForwardsReconfigureScope(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var reconfigureBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/test/add":
			if r.Method != http.MethodPost {
				t.Errorf("add method = %s, want POST", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": "saved",
				"uuid":   "test-uuid",
				"reconfigure": map[string]any{
					"interfaces": []string{"core_transit"},
				},
			})
		case "/api/test/reconfigure":
			if r.Method != http.MethodPost {
				t.Errorf("reconfigure method = %s, want POST", r.Method)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode reconfigure body: %v", err)
			}
			mu.Lock()
			reconfigureBody = body
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Options{Uri: server.URL})
	opts := ReqOpts{
		Create:      Endpoint{Path: "/test/add", Method: http.MethodPost},
		Reconfigure: Endpoint{Path: "/test/reconfigure", Method: http.MethodPost},
		Monad:       "item",
	}
	id, err := Add(client, context.Background(), opts, &scopedCRUDTestResource{Name: "test"})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if id != "test-uuid" {
		t.Fatalf("Add() id = %q, want %q", id, "test-uuid")
	}

	mu.Lock()
	got := reconfigureBody
	mu.Unlock()
	want := map[string]any{"interfaces": []any{"core_transit"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reconfigure body = %#v, want %#v", got, want)
	}
}

func TestDeleteForwardsReconfigureScope(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var reconfigureBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/test/delete/test-uuid":
			if r.Method != http.MethodPost {
				t.Errorf("delete method = %s, want POST", r.Method)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"result": "deleted",
				"reconfigure": map[string]any{
					"interfaces": []string{"core_transit"},
				},
			})
		case "/api/test/reconfigure":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode reconfigure body: %v", err)
			}
			mu.Lock()
			reconfigureBody = body
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Options{Uri: server.URL})
	opts := ReqOpts{
		Delete:      Endpoint{Path: "/test/delete", Method: http.MethodPost},
		Reconfigure: Endpoint{Path: "/test/reconfigure", Method: http.MethodPost},
		Monad:       "item",
	}
	if err := Delete(client, context.Background(), opts, "test-uuid"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	mu.Lock()
	got := reconfigureBody
	mu.Unlock()
	want := map[string]any{"interfaces": []any{"core_transit"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("reconfigure body = %#v, want %#v", got, want)
	}
}

func TestReconfigureServiceWithoutScopeKeepsLegacyEmptyBody(t *testing.T) {
	t.Parallel()

	var bodyBytes int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes = r.ContentLength
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(Options{Uri: server.URL})
	if err := client.ReconfigureService(context.Background(), Endpoint{Path: "/test/reconfigure", Method: http.MethodPost}); err != nil {
		t.Fatalf("ReconfigureService() error = %v", err)
	}
	if bodyBytes > 0 {
		t.Fatalf("legacy reconfigure sent body content length %d, want zero/unknown", bodyBytes)
	}
}
