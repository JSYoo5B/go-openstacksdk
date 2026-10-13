package jsonfilter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"
)

// EqualPythonLoadedJSON compares JSON as CPython json.loads followed by ==:
// integer tokens retain arbitrary precision, decimal/exponent tokens become
// IEEE-754 floats, and booleans are integer subclasses. It is an explicit
// source-policy opt-in; EqualPythonJSON retains its exact decimal behavior.
func EqualPythonLoadedJSON(left, right json.RawMessage) (bool, error) {
	decode := func(raw json.RawMessage) (any, error) {
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, fmt.Errorf("Python loaded equality requires complete UTF-8 JSON")
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		return value, nil
	}
	l, err := decode(left)
	if err != nil {
		return false, err
	}
	r, err := decode(right)
	if err != nil {
		return false, err
	}
	return equalPythonLoaded(l, r)
}

type pythonLoadedNumber struct {
	integer  *big.Int
	floating float64
}

func pythonLoadedNumeric(value any) (pythonLoadedNumber, bool, error) {
	switch value := value.(type) {
	case bool:
		number := int64(0)
		if value {
			number = 1
		}
		return pythonLoadedNumber{integer: big.NewInt(number)}, true, nil
	case json.Number:
		token := string(value)
		if strings.ContainsAny(token, ".eE") {
			number, err := strconv.ParseFloat(token, 64)
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				return pythonLoadedNumber{}, true, err
			}
			return pythonLoadedNumber{floating: number}, true, nil
		}
		number, ok := new(big.Int).SetString(token, 10)
		if !ok {
			return pythonLoadedNumber{}, true, fmt.Errorf("invalid JSON integer")
		}
		return pythonLoadedNumber{integer: number}, true, nil
	}
	return pythonLoadedNumber{}, false, nil
}

func equalPythonLoaded(left, right any) (bool, error) {
	l, ln, err := pythonLoadedNumeric(left)
	if err != nil {
		return false, err
	}
	r, rn, err := pythonLoadedNumeric(right)
	if err != nil {
		return false, err
	}
	if ln || rn {
		if !ln || !rn {
			return false, nil
		}
		if l.integer != nil && r.integer != nil {
			return l.integer.Cmp(r.integer) == 0, nil
		}
		if l.integer == nil && r.integer == nil {
			return l.floating == r.floating, nil
		}
		if l.integer == nil {
			l, r = r, l
		}
		// SetFloat64 is exact and returns nil for infinity. A rounded float may
		// equal the adjacent integer, but cannot equal the pre-rounding integer.
		ratio := new(big.Rat).SetFloat64(r.floating)
		return ratio != nil && ratio.Cmp(new(big.Rat).SetInt(l.integer)) == 0, nil
	}
	switch left := left.(type) {
	case nil:
		return right == nil, nil
	case string:
		value, ok := right.(string)
		return ok && left == value, nil
	case []any:
		values, ok := right.([]any)
		if !ok || len(left) != len(values) {
			return false, nil
		}
		for index, value := range left {
			equal, err := equalPythonLoaded(value, values[index])
			if err != nil || !equal {
				return equal, err
			}
		}
		return true, nil
	case map[string]any:
		values, ok := right.(map[string]any)
		if !ok || len(left) != len(values) {
			return false, nil
		}
		for key, value := range left {
			other, ok := values[key]
			if !ok {
				return false, nil
			}
			equal, err := equalPythonLoaded(value, other)
			if err != nil || !equal {
				return equal, err
			}
		}
		return true, nil
	}
	return false, fmt.Errorf("unsupported decoded JSON value %T", left)
}
