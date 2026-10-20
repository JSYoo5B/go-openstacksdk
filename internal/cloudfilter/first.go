package cloudfilter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// MultipleError describes Python's len(value) > 1 ambiguity check without
// inventing resource IDs for arbitrary expression results.
type MultipleError struct{ Length int }

func (e MultipleError) Error() string {
	return fmt.Sprintf("multiple matches found: length %d", e.Length)
}

// First applies filtered get's truthiness, then len, then integer index 0.
// The selected element is not tested for truthiness again: false, 0 and an
// empty string remain present results; only a selected JSON null becomes nil.
func First(value json.RawMessage) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	n, err := parse(value)
	if err != nil {
		return nil, fmt.Errorf("first selection: %w", err)
	}
	if !n.truthy() {
		return nil, nil
	}
	switch n.kind {
	case '[':
		if len(n.items) > 1 {
			return nil, &MultipleError{Length: len(n.items)}
		}
		first := n.items[0]
		if first.kind == 'n' {
			return nil, nil
		}
		return bytes.Clone(first.raw), nil
	case '"':
		length := utf8.RuneCountInString(n.text)
		if length > 1 {
			return nil, &MultipleError{Length: length}
		}
		return bytes.Clone(n.raw), nil
	case '{':
		if len(n.members) > 1 {
			return nil, &MultipleError{Length: len(n.members)}
		}
		return nil, fmt.Errorf("first selection: object has no integer key 0")
	default:
		return nil, fmt.Errorf("first selection: value has no length")
	}
}
