// Adapted from github.com/jmespath/go-jmespath v0.4.0.
// Copyright 2015 James Saryerwinnie; licensed under Apache-2.0.
// See LICENSE and README.md for attribution and local changes.

package jmespath

import (
	"encoding/json"
	"errors"
	"sort"
)

// isFalse follows JMESPath truthiness: numeric zero is true.
func isFalse(value any) bool {
	switch value := value.(type) {
	case bool:
		return !value
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	case string:
		return value == ""
	case nil:
		return true
	default:
		return false
	}
}

// Scalar bool/number equality is distinct. Once comparing a complete array
// or object, the selected Python reference uses ordinary recursive equality.
func objsEqual(left, right any) (bool, error) { return jsonEqual(left, right, false) }

func jsonEqual(left, right any, ordinary bool) (bool, error) {
	switch left := left.(type) {
	case nil:
		return right == nil, nil
	case json.Number:
		r, ok := right.(json.Number)
		if !ok && ordinary {
			if b, yes := right.(bool); yes {
				if b {
					r = json.Number("1")
				} else {
					r = json.Number("0")
				}
				ok = true
			}
		}
		if !ok {
			return false, nil
		}
		comparison, err := compareNumbers(left, r)
		return comparison == 0, err
	case bool:
		if r, ok := right.(bool); ok {
			return left == r, nil
		}
		if r, ok := right.(json.Number); ok && ordinary {
			l := json.Number("0")
			if left {
				l = json.Number("1")
			}
			comparison, err := compareNumbers(l, r)
			return comparison == 0, err
		}
		return false, nil
	case string:
		r, ok := right.(string)
		return ok && left == r, nil
	case []any:
		r, ok := right.([]any)
		if !ok || len(left) != len(r) {
			return false, nil
		}
		for i, item := range left {
			equal, err := jsonEqual(item, r[i], true)
			if err != nil || !equal {
				return false, err
			}
		}
		return true, nil
	case map[string]any:
		r, ok := right.(map[string]any)
		if !ok || len(left) != len(r) {
			return false, nil
		}
		for _, key := range sortedKeys(left) {
			item, exists := r[key]
			if !exists {
				return false, nil
			}
			equal, err := jsonEqual(left[key], item, true)
			if err != nil || !equal {
				return false, err
			}
		}
		return true, nil
	default:
		return false, nil
	}
}

func sortedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// SliceParam refers to a single part of a slice.
// A slice consists of a start, a stop, and a step, similar to
// python slices.
type sliceParam struct {
	N         int
	Specified bool
}

// Slice supports [start:stop:step] style slicing that's supported in JMESPath.
func slice(slice []interface{}, parts []sliceParam) ([]interface{}, error) {
	computed, err := computeSliceParams(len(slice), parts)
	if err != nil {
		return nil, err
	}
	start, stop, step := computed[0], computed[1], computed[2]
	result := []interface{}{}
	if step > 0 {
		for i := start; i < stop; {
			result = append(result, slice[i])
			if step >= stop-i {
				break
			}
			i += step
		}
	} else {
		for i := start; i > stop; {
			result = append(result, slice[i])
			if step <= stop-i {
				break
			}
			i += step
		}
	}
	return result, nil
}

func computeSliceParams(length int, parts []sliceParam) ([]int, error) {
	var start, stop, step int
	if !parts[2].Specified {
		step = 1
	} else if parts[2].N == 0 {
		return nil, errors.New("Invalid slice, step cannot be 0")
	} else {
		step = parts[2].N
	}
	var stepValueNegative bool
	if step < 0 {
		stepValueNegative = true
	} else {
		stepValueNegative = false
	}

	if !parts[0].Specified {
		if stepValueNegative {
			start = length - 1
		} else {
			start = 0
		}
	} else {
		start = capSlice(length, parts[0].N, step)
	}

	if !parts[1].Specified {
		if stepValueNegative {
			stop = -1
		} else {
			stop = length
		}
	} else {
		stop = capSlice(length, parts[1].N, step)
	}
	return []int{start, stop, step}, nil
}

func capSlice(length int, actual int, step int) int {
	if actual < 0 {
		actual += length
		if actual < 0 {
			if step < 0 {
				actual = -1
			} else {
				actual = 0
			}
		}
	} else if actual >= length {
		if step < 0 {
			actual = length - 1
		} else {
			actual = length
		}
	}
	return actual
}

func toArrayNum(data any) ([]json.Number, bool) {
	values, ok := data.([]any)
	if !ok {
		return nil, false
	}
	result := make([]json.Number, len(values))
	for i, value := range values {
		number, ok := value.(json.Number)
		if !ok {
			return nil, false
		}
		result[i] = number
	}
	return result, true
}

func toArrayStr(data any) ([]string, bool) {
	values, ok := data.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, len(values))
	for i, value := range values {
		text, ok := value.(string)
		if !ok {
			return nil, false
		}
		result[i] = text
	}
	return result, true
}

func isSliceType(value any) bool { _, ok := value.([]any); return ok }
