package firewall

import (
	"encoding/json"
	"testing"
)

func TestFilterLogAcceptsBooleanOrString(t *testing.T) {
	tests := map[string]string{
		`{"log":false}`: "0",
		`{"log":true}`:  "1",
		`{"log":"0"}`:   "0",
		`{"log":"1"}`:   "1",
	}
	for input, want := range tests {
		var filter Filter
		if err := json.Unmarshal([]byte(input), &filter); err != nil {
			t.Fatalf("json.Unmarshal(%s): %v", input, err)
		}
		if filter.Log.String() != want {
			t.Fatalf("json.Unmarshal(%s) log=%q want=%q", input, filter.Log, want)
		}
	}
}
