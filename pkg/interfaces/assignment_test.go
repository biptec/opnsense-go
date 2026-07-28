package interfaces

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestAssignmentLifecycleUsesCanonicalAPIPaths(t *testing.T) {
	t.Parallel()

	var (
		mu       sync.Mutex
		requests []string
	)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()

		if user, secret, ok := r.BasicAuth(); !ok || user != "key" || secret != "secret" {
			t.Errorf("unexpected basic authentication: user=%q ok=%t", user, ok)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /api/interfaces/assignment/add_item":
			assertAssignmentPayload(t, r, "vtnet1", "LAN")
			io.WriteString(w, `{"result":"saved","uuid":"transient-uuid"}`)
		case "POST /api/interfaces/assignment/reconfigure":
			io.WriteString(w, `{"status":"ok"}`)
		case "POST /api/interfaces/assignment/search_item":
			assertSearchPayload(t, r, "-1")
			io.WriteString(w, `{"total":1,"rowCount":1,"current":1,"rows":[{"identifier":"opt1"}]}`)
		case "GET /api/interfaces/assignment/get_item/opt1":
			io.WriteString(w, `{"interface":{"identifier":"opt1","descr":"LAN","if":{"vtnet1":{"selected":1,"value":"vtnet1"}},"type":{"static":{"selected":1,"value":"Static IPv4"}},"type6":{"none":{"selected":1,"value":"None"}},"ipaddr":"192.0.2.1","subnet":"24"}}`)
		case "POST /api/interfaces/assignment/set_item/opt1":
			assertAssignmentPayload(t, r, "vtnet1", "LAN updated")
			io.WriteString(w, `{"result":"saved"}`)
		case "POST /api/interfaces/assignment/del_item/opt1":
			io.WriteString(w, `{"result":"deleted"}`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"error":"unexpected request"}`)
		}
	}))
	defer server.Close()

	client := api.NewClient(api.Options{
		Uri:           server.URL,
		APIKey:        "key",
		APISecret:     "secret",
		AllowInsecure: true,
		MaxRetries:    1,
		Logger:        log.New(io.Discard, "", 0),
	})
	controller := Controller{Api: client}
	ctx := context.Background()

	assignment := &Assignment{
		Description:      "LAN",
		Device:           api.SelectedMap("vtnet1"),
		Lock:             "0",
		Enabled:          "1",
		IPv4Mode:         api.SelectedMap("static"),
		IPv4Address:      "192.0.2.1",
		IPv4PrefixLength: "24",
		IPv6Mode:         api.SelectedMap("none"),
	}

	identifier, err := controller.AddAssignment(ctx, assignment)
	if err != nil {
		t.Fatalf("AddAssignment failed: %v", err)
	}
	if identifier != "opt1" {
		t.Fatalf("AddAssignment identifier = %q, want opt1", identifier)
	}

	assignment.Description = "LAN updated"
	if err := controller.UpdateAssignment(ctx, identifier, assignment); err != nil {
		t.Fatalf("UpdateAssignment failed: %v", err)
	}
	if err := controller.DeleteAssignment(ctx, identifier); err != nil {
		t.Fatalf("DeleteAssignment failed: %v", err)
	}

	want := []string{
		"POST /api/interfaces/assignment/add_item",
		"POST /api/interfaces/assignment/reconfigure",
		"POST /api/interfaces/assignment/search_item",
		"GET /api/interfaces/assignment/get_item/opt1",
		"POST /api/interfaces/assignment/set_item/opt1",
		"POST /api/interfaces/assignment/reconfigure",
		"POST /api/interfaces/assignment/del_item/opt1",
		"POST /api/interfaces/assignment/reconfigure",
	}
	mu.Lock()
	got := append([]string(nil), requests...)
	mu.Unlock()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("request sequence mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestAddAssignmentRequiresDevice(t *testing.T) {
	controller := Controller{}

	if _, err := controller.AddAssignment(context.Background(), nil); err == nil {
		t.Fatal("expected nil assignment error")
	}
	if _, err := controller.AddAssignment(context.Background(), &Assignment{}); err == nil {
		t.Fatal("expected empty device error")
	}
}

func assertAssignmentPayload(t *testing.T, r *http.Request, wantDevice, wantDescription string) {
	t.Helper()

	var body struct {
		Interface struct {
			Device      string `json:"if"`
			Description string `json:"descr"`
		} `json:"interface"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	if body.Interface.Device != wantDevice {
		t.Errorf("interface.if = %q, want %q", body.Interface.Device, wantDevice)
	}
	if body.Interface.Description != wantDescription {
		t.Errorf("interface.descr = %q, want %q", body.Interface.Description, wantDescription)
	}
}

func assertSearchPayload(t *testing.T, r *http.Request, wantRowCount string) {
	t.Helper()

	var body struct {
		RowCount string `json:"rowCount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode search request body: %v", err)
	}
	if body.RowCount != wantRowCount {
		t.Errorf("rowCount = %q, want %q", body.RowCount, wantRowCount)
	}
}
