package haproxy

import (
	"encoding/json"
	"testing"

	"github.com/biptec/opnsense-go/pkg/api"
)

func TestBackendMarshalPreservesEmptyStickinessPattern(t *testing.T) {
	payload, err := json.Marshal(&Backend{
		Enabled:           "1",
		Name:              "leastconn-backend",
		Mode:              api.SelectedMap("tcp"),
		Algorithm:         api.SelectedMap("leastconn"),
		StickinessPattern: api.SelectedMap(""),
		LinkedServers:     api.SelectedMapList{},
	})
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	value, ok := got["stickiness_pattern"]
	if !ok {
		t.Fatalf("stickiness_pattern omitted from payload: %s", payload)
	}
	if value != "" {
		t.Fatalf("stickiness_pattern = %#v, want empty string", value)
	}
}
