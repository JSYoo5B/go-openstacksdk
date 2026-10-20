package jsonfilter

import "encoding/json"

// BooleanJSON applies response-only JSON truthiness for audited bool
// descriptors. Null remains null; booleans are unchanged; empty strings,
// arrays, objects and numeric zero are false. Other values are true.
// Numbers retain exact decimal zero/nonzero semantics without float64
// conversion or expanding their exponent. Caller filters are never coerced.
func BooleanJSON(raw json.RawMessage) (json.RawMessage, error) {
	value, err := filterJSON(raw)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return json.RawMessage("null"), nil
	}
	truthy := false
	switch value := value.(type) {
	case bool:
		truthy = value
	case string:
		truthy = len(value) != 0
	case []any:
		truthy = len(value) != 0
	case map[string]any:
		truthy = len(value) != 0
	case json.Number:
		// The decoder validated the grammar. Only coefficient digits decide
		// zero; the exponent cannot turn a nonzero exact decimal into zero.
		for _, digit := range value.String() {
			if digit == 'e' || digit == 'E' {
				break
			}
			if digit >= '1' && digit <= '9' {
				truthy = true
				break
			}
		}
	}
	if truthy {
		return json.RawMessage("true"), nil
	}
	return json.RawMessage("false"), nil
}
