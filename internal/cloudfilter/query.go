package cloudfilter

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// JSONMember preserves parsed dictionary order. Duplicate keys replace the
// value at their first insertion position, as ordinary Python JSON loading does.
type JSONMember struct {
	Key   string
	Value json.RawMessage
}

func ObjectMembers(raw json.RawMessage) ([]JSONMember, error) {
	n, err := parse(raw)
	if err != nil {
		return nil, err
	}
	if n.kind != '{' {
		return nil, fmt.Errorf("JSON value must be an object")
	}
	result := make([]JSONMember, len(n.members))
	for i, member := range n.members {
		result[i] = JSONMember{Key: member.key, Value: bytes.Clone(member.value.raw)}
	}
	return result, nil
}

// RequestQueryValues follows ordinary Requests' iterable query expansion for
// JSON-domain values. Null items are omitted; dictionary values iterate keys.
// Number and nested-container spelling uses the audited identifier repr policy.
// This transport dependency is separate from the pinned OpenStack SDK source.
func RequestQueryValues(raw json.RawMessage) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	n, err := parse(raw)
	if err != nil {
		return nil, err
	}
	var items []*node
	switch n.kind {
	case 'n':
		return nil, nil
	case '[':
		items = n.items
	case '{':
		result := make([]string, len(n.members))
		for i, member := range n.members {
			result[i] = member.key
		}
		return result, nil
	default:
		items = []*node{n}
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item.kind == 'n' {
			continue
		}
		// Requests has expanded the outer iterable; urlencode's doseq=True
		// expands one further level. Inner null is serialized as "None".
		if item.kind == '{' {
			for _, member := range item.members {
				result = append(result, member.key)
			}
			continue
		}
		children := []*node{item}
		if item.kind == '[' {
			children = item.items
		}
		for _, child := range children {
			value, err := pythonString(child, nil)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
	}
	return result, nil
}
