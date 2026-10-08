package cloudfilter

import (
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

// mapping deliberately checks actual truthiness before filter shape, and
// consumes keys in source insertion order. An earlier mismatch can therefore
// leave an invalid later key or nested scalar unvisited.
func mapping(filter, actual *node, path string, guard func() error) (bool, error) {
	if err := check(guard); err != nil {
		return false, err
	}
	if !actual.truthy() {
		return false, nil
	}
	if filter.kind != '{' {
		return false, fmt.Errorf("filter %s has no mapping keys", path)
	}
	for _, item := range filter.members {
		if err := check(guard); err != nil {
			return false, err
		}
		fieldPath := path + "[" + fmt.Sprintf("%q", item.key) + "]"
		value, err := mappingValue(actual, item.key, fieldPath)
		if err != nil {
			return false, err
		}
		var matches bool
		if item.value.kind == '{' {
			matches, err = mapping(item.value, value, fieldPath, guard)
		} else {
			matches, err = jsonfilter.EqualPythonJSON(value.raw, item.value.raw)
		}
		if err != nil {
			return false, fmt.Errorf("filter %s: %w", fieldPath, err)
		}
		if !matches {
			return false, nil
		}
	}
	return true, check(guard)
}

func mappingValue(actual *node, key, path string) (*node, error) {
	switch actual.kind {
	case '{':
		if value, exists := actual.fields[key]; exists {
			return value, nil
		}
		return nil, fmt.Errorf("filter field %s is not an attribute", path)
	case '"':
		if !strings.Contains(actual.text, key) {
			return nil, fmt.Errorf("filter field %s is not an attribute", path)
		}
		return nil, fmt.Errorf("filter field %s cannot call get on a string", path)
	case '[':
		for _, value := range actual.items {
			if value.kind == '"' && value.text == key {
				return nil, fmt.Errorf("filter field %s cannot call get on an array", path)
			}
		}
		return nil, fmt.Errorf("filter field %s is not an attribute", path)
	default:
		return nil, fmt.Errorf("filter field %s requires membership in an iterable value", path)
	}
}
