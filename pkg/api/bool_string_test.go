package api

import (
	"encoding/json"
	"testing"
)

func TestBoolStringJSON(t *testing.T) {
	tests := []struct {
		input string
		want  BoolString
	}{
		{`true`, "1"},
		{`false`, "0"},
		{`"1"`, "1"},
		{`"0"`, "0"},
		{`"true"`, "1"},
		{`"false"`, "0"},
		{`1`, "1"},
		{`0`, "0"},
	}
	for _, test := range tests {
		var got BoolString
		if err := json.Unmarshal([]byte(test.input), &got); err != nil {
			t.Fatalf("Unmarshal(%s): %v", test.input, err)
		}
		if got != test.want {
			t.Fatalf("Unmarshal(%s) = %q, want %q", test.input, got, test.want)
		}
		encoded, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("Marshal(%q): %v", got, err)
		}
		if string(encoded) != `"`+test.want.String()+`"` {
			t.Fatalf("Marshal(%q) = %s", got, encoded)
		}
	}
}
