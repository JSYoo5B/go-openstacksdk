package senlin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
)

// MatchFilters applies the Body-filter semantics of pinned Python Resource.list
// (resource.py:2256-2267, 2324-2336). Object filters recursively match a subset
// of a nonempty actual object. Other JSON values, including arrays and objects
// inside arrays, compare exactly. Missing fields compare as null.
//
// Go deliberately distinguishes JSON booleans from numbers, unlike Python's
// True == 1 and False == 0. Numbers compare by exact decimal value without
// float64 conversion or expanding exponent-sized integers. A non-object actual
// value does not match an object filter, rather than invoking Python truthiness
// or raising an attribute error.
//
// Only referenced body fields are decoded. All filters and referenced fields
// are validated before comparison, so malformed JSON cannot be hidden by an
// earlier mismatch. The inputs are never modified.
func MatchFilters(body map[string]json.RawMessage, filters map[string]json.RawMessage) (bool, error) {
	requested := make(map[string]any, len(filters))
	actual := make(map[string]any, len(filters))
	for key, raw := range filters {
		value, err := filterJSON(raw)
		if err != nil {
			return false, fmt.Errorf("Senlin filter %q: %w", key, err)
		}
		requested[key] = value
	}
	for key := range filters {
		if raw, exists := body[key]; exists {
			value, err := filterJSON(raw)
			if err != nil {
				return false, fmt.Errorf("Senlin body field %q: %w", key, err)
			}
			actual[key] = value
		}
	}
	for key, value := range requested {
		if !matchFilterValue(actual[key], value) {
			return false, nil
		}
	}
	return true, nil
}

func filterJSON(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

func matchFilterValue(actual, filter any) bool {
	if subset, ok := filter.(map[string]any); ok {
		object, ok := actual.(map[string]any)
		if !ok || len(object) == 0 {
			return false
		}
		for key, value := range subset {
			if !matchFilterValue(object[key], value) {
				return false
			}
		}
		return true
	}
	return equalFilterJSON(actual, filter)
}

func equalFilterJSON(actual, expected any) bool {
	switch value := expected.(type) {
	case nil:
		return actual == nil
	case bool:
		other, ok := actual.(bool)
		return ok && other == value
	case string:
		other, ok := actual.(string)
		return ok && other == value
	case json.Number:
		other, ok := actual.(json.Number)
		if !ok {
			return false
		}
		left, right := canonicalFilterNumber(other), canonicalFilterNumber(value)
		return left.negative == right.negative && left.digits == right.digits && left.exponent.Cmp(&right.exponent) == 0
	case []any:
		other, ok := actual.([]any)
		if !ok || len(other) != len(value) {
			return false
		}
		for i := range value {
			if !equalFilterJSON(other[i], value[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		other, ok := actual.(map[string]any)
		if !ok || len(other) != len(value) {
			return false
		}
		for key, entry := range value {
			actualEntry, exists := other[key]
			if !exists || !equalFilterJSON(actualEntry, entry) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

type filterNumber struct {
	negative bool
	digits   string
	exponent big.Int
}

// Validated JSON numbers normalize to sign * digits * 10^exponent, with no
// leading/trailing zero digits. Only the exponent itself uses big.Int; memory
// therefore follows input length, even when the exponent exceeds machine ints.
func canonicalFilterNumber(number json.Number) filterNumber {
	text := number.String()
	var result filterNumber
	if text[0] == '-' {
		result.negative = true
		text = text[1:]
	}
	exponent := "0"
	if i := strings.IndexAny(text, "eE"); i >= 0 {
		exponent, text = text[i+1:], text[:i]
	}
	fraction := 0
	if i := strings.IndexByte(text, '.'); i >= 0 {
		fraction = len(text) - i - 1
		text = text[:i] + text[i+1:]
	}
	text = strings.TrimLeft(text, "0")
	if text == "" {
		// All spellings of zero compare equally, regardless of sign/exponent.
		return filterNumber{digits: "0"}
	}
	result.digits = strings.TrimRight(text, "0")
	trailing := len(text) - len(result.digits)
	result.exponent.SetString(exponent, 10) // Decoder already validated grammar.
	result.exponent.Add(&result.exponent, big.NewInt(int64(trailing-fraction)))
	return result
}
