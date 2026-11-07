package haproxy

import (
	"encoding/json"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestFrontendMarshalPreservesEmptyDefaultBackend(t *testing.T) {
	payload, err := json.Marshal(&Frontend{
		Enabled:        "1",
		Name:           "shared-internal",
		Bind:           api.SelectedMapList{"10.0.0.1:443"},
		Mode:           api.SelectedMap("http"),
		DefaultBackend: api.SelectedMap(""),
		LinkedActions:  api.SelectedMapList{},
	})
	if err != nil {
		t.Fatalf("marshal frontend: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal frontend payload: %v", err)
	}
	value, ok := got["defaultBackend"]
	if !ok {
		t.Fatalf("defaultBackend omitted from payload: %s", payload)
	}
	if value != "" {
		t.Fatalf("defaultBackend = %#v, want empty string", value)
	}
}
