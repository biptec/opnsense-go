package api

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// BoolString is the string representation OPNsense uses for boolean model
// fields. Some search endpoints return native JSON booleans instead; this type
// accepts both forms and always marshals the canonical "1"/"0" string form.
type BoolString string

func (value BoolString) String() string {
	return string(value)
}

func (value BoolString) Bool() bool {
	return value == "1" || value == "true"
}

func (value *BoolString) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		switch text {
		case "true":
			text = "1"
		case "false":
			text = "0"
		}
		*value = BoolString(text)
		return nil
	}

	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		if boolean {
			*value = "1"
		} else {
			*value = "0"
		}
		return nil
	}

	if bytes.Equal(data, []byte("1")) {
		*value = "1"
		return nil
	}
	if bytes.Equal(data, []byte("0")) {
		*value = "0"
		return nil
	}
	return fmt.Errorf("expected boolean or boolean string, got %s", data)
}

func (value BoolString) MarshalJSON() ([]byte, error) {
	switch value {
	case "true":
		value = "1"
	case "false":
		value = "0"
	}
	return json.Marshal(string(value))
}
