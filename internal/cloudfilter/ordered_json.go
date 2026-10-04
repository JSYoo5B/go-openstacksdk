// Package cloudfilter implements the local selection phases of Python cloud
// helpers over owned, normalized JSON resource views.
package cloudfilter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

type member struct {
	key   string
	value *node
}

type node struct {
	raw     json.RawMessage
	kind    byte
	text    string
	members []member
	fields  map[string]*node
	items   []*node
}

// parse validates a complete ordinary JSON document. Object member order is
// retained; repeated keys replace their value at the original insertion slot.
func parse(raw json.RawMessage) (*node, error) {
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("JSON value must be valid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	var document json.RawMessage
	if err := d.Decode(&document); err != nil {
		return nil, err
	}
	var extra json.RawMessage
	if err := d.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return parseValid(document)
}

func parseValid(raw json.RawMessage) (*node, error) {
	raw = bytes.Clone(bytes.TrimSpace(raw))
	n := &node{raw: raw, kind: raw[0]}
	switch n.kind {
	case '{':
		n.fields = make(map[string]*node)
		positions := make(map[string]int)
		d := json.NewDecoder(bytes.NewReader(raw))
		if _, err := d.Token(); err != nil {
			return nil, err
		}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return nil, err
			}
			key, ok := token.(string)
			if !ok {
				return nil, fmt.Errorf("JSON object key must be a string")
			}
			var value json.RawMessage
			if err := d.Decode(&value); err != nil {
				return nil, err
			}
			child, err := parseValid(value)
			if err != nil {
				return nil, err
			}
			n.fields[key] = child
			if position, exists := positions[key]; exists {
				n.members[position].value = child
			} else {
				positions[key] = len(n.members)
				n.members = append(n.members, member{key: key, value: child})
			}
		}
	case '[':
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		n.items = make([]*node, len(values))
		for i, value := range values {
			child, err := parseValid(value)
			if err != nil {
				return nil, err
			}
			n.items[i] = child
		}
	case '"':
		if err := json.Unmarshal(raw, &n.text); err != nil {
			return nil, err
		}
	}
	return n, nil
}

func (n *node) truthy() bool {
	switch n.kind {
	case 'n', 'f':
		return false
	case 't':
		return true
	case '"':
		return n.text != ""
	case '[':
		return len(n.items) != 0
	case '{':
		return len(n.members) != 0
	default:
		// A JSON decimal is zero exactly when every mantissa digit is zero.
		// Do not round large integers or underflow small decimals to float64.
		for _, c := range n.raw {
			if c == 'e' || c == 'E' {
				break
			}
			if c >= '1' && c <= '9' {
				return true
			}
		}
		return false
	}
}

func check(guard func() error) error {
	if guard != nil {
		return guard()
	}
	return nil
}
