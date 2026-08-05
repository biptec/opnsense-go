package bind

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRNDCStatusUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    RNDCStatus
		wantErr bool
	}{
		{name: "object", input: `{"secure":"yes","inline_signing":"yes"}`, want: RNDCStatus{"secure": "yes", "inline_signing": "yes"}},
		{name: "empty object", input: `{}`, want: RNDCStatus{}},
		{name: "empty array", input: `[]`, want: RNDCStatus{}},
		{name: "null", input: `null`, want: RNDCStatus{}},
		{name: "non-empty array", input: `["unexpected"]`, wantErr: true},
		{name: "scalar", input: `"unexpected"`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var got RNDCStatus
			err := json.Unmarshal([]byte(test.input), &got)
			if test.wantErr {
				if err == nil {
					t.Fatalf("Unmarshal(%s) succeeded, want error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal(%s): %v", test.input, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Unmarshal(%s) = %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}
