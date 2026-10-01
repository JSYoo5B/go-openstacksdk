package senlin

import "encoding/json"

// EqualJSON compares complete valid JSON values without losing number precision.
// Object order and decimal spelling do not affect equality; absent, invalid,
// boolean and numeric values remain distinct.
func EqualJSON(left, right json.RawMessage) bool {
	actual, err := filterJSON(left)
	if err != nil {
		return false
	}
	expected, err := filterJSON(right)
	if err != nil {
		return false
	}
	return equalFilterJSON(actual, expected)
}
