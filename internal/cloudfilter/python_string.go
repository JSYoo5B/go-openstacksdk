package cloudfilter

import "encoding/json"

// PythonString applies the existing JSON-domain Python string policy without
// Requests' iterable query expansion. SDK string and BoolStr descriptors use
// this boundary; the parser and representation limits remain shared.
func PythonString(raw json.RawMessage) (string, error) {
	value, err := parse(raw)
	if err != nil {
		return "", err
	}
	return pythonString(value, nil)
}
