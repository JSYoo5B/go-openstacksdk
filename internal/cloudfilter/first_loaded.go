package cloudfilter

import "encoding/json"

// FirstLoaded applies Python JSON-loaded truthiness before filtered get's
// length/index selection. In particular a float underflow is false even when
// the original JSON decimal is nonzero. First retains its existing contract.
func FirstLoaded(value json.RawMessage) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	truth, err := PythonJSONTruthy(value)
	if err != nil {
		return nil, err
	}
	if !truth {
		return nil, nil
	}
	return First(value)
}
