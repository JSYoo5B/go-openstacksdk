package cloudfilter

import (
	"encoding/json"
	"fmt"
)

// PythonTruthy applies ordinary Python truthiness in the JSON domain, where
// numerical zero is false. A nil value represents an absent/None result;
// nonnil empty bytes are an invalid JSON document.
func PythonTruthy(raw json.RawMessage) (bool, error) {
	if raw == nil {
		return false, nil
	}
	n, err := parse(raw)
	if err != nil {
		return false, fmt.Errorf("JSON truthiness: %w", err)
	}
	return n.truthy(), nil
}
