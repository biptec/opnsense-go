package bind

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// RNDCStatus accepts the object returned by dnssecStatus.py and the empty
// array produced when PHP re-encodes an empty associative array.
type RNDCStatus map[string]string

func (s *RNDCStatus) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("rndc_status is empty")
	}

	switch data[0] {
	case '{':
		var values map[string]string
		if err := json.Unmarshal(data, &values); err != nil {
			return fmt.Errorf("decode rndc_status object: %w", err)
		}
		if values == nil {
			values = map[string]string{}
		}
		*s = RNDCStatus(values)
		return nil
	case '[':
		var values []json.RawMessage
		if err := json.Unmarshal(data, &values); err != nil {
			return fmt.Errorf("decode rndc_status array: %w", err)
		}
		if len(values) != 0 {
			return fmt.Errorf("rndc_status must be an object or an empty array")
		}
		*s = RNDCStatus{}
		return nil
	case 'n':
		if !bytes.Equal(data, []byte("null")) {
			return fmt.Errorf("rndc_status has invalid null value")
		}
		*s = RNDCStatus{}
		return nil
	default:
		return fmt.Errorf("rndc_status must be an object, empty array, or null")
	}
}
