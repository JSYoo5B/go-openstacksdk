package jsonfilter

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// EqualPythonJSON compares complete JSON values with Python's recursive
// boolean/number equality. Numbers use the existing exact decimal comparison;
// this does not emulate Python floating-point rounding or custom objects.
// EqualJSON and ordinary Body filters keep their original boolean distinction.
func EqualPythonJSON(left, right json.RawMessage) (bool, error) {
	if !utf8.Valid(left) || !utf8.Valid(right) {
		return false, fmt.Errorf("JSON equality values must be valid UTF-8")
	}
	actual, err := filterJSON(left)
	if err != nil {
		return false, fmt.Errorf("left JSON: %w", err)
	}
	expected, err := filterJSON(right)
	if err != nil {
		return false, fmt.Errorf("right JSON: %w", err)
	}
	return equalFilterJSON(pythonEqualityValue(actual), pythonEqualityValue(expected)), nil
}

func pythonEqualityValue(value any) any {
	switch value := value.(type) {
	case bool:
		if value {
			return json.Number("1")
		}
		return json.Number("0")
	case []any:
		values := make([]any, len(value))
		for index, item := range value {
			values[index] = pythonEqualityValue(item)
		}
		return values
	case map[string]any:
		values := make(map[string]any, len(value))
		for key, item := range value {
			values[key] = pythonEqualityValue(item)
		}
		return values
	default:
		return value
	}
}
