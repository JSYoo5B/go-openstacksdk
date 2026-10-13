package neutronfind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

// descriptorValue preserves the top-level null bypass and applies list_type
// to elements after scalar wrapping. Source dict(null) in a list becomes {}.
func (e *Engine) descriptorValue(raw json.RawMessage, field Descriptor) (json.RawMessage, error) {
	value, err := e.descriptorJSON(raw, field.Kind)
	if err != nil || field.Kind != List || field.ListItemKind == Untyped || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return value, err
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(value, &elements); err != nil {
		return nil, err
	}
	for i, item := range elements {
		if field.ListItemKind == Dictionary && (len(bytes.TrimSpace(item)) == 0 || bytes.TrimSpace(item)[0] != '{') {
			elements[i] = json.RawMessage("{}")
			continue
		}
		elements[i], err = e.descriptorJSON(item, field.ListItemKind)
		if err != nil {
			return nil, err
		}
	}
	return json.Marshal(elements)
}

func (e *Engine) codes() []int {
	if !e.profile.SourceStatusPolicy {
		return findCodes()
	}
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	return codes
}

func (e *Engine) projectDescriptor(ctx context.Context, raw json.RawMessage, field Descriptor) (json.RawMessage, error) {
	value, err := e.descriptorValue(raw, field)
	if err != nil || field.Project == nil {
		return value, err
	}
	if err := e.Check(ctx); err != nil {
		return nil, err
	}
	projected, err := field.Project(bytes.Clone(value))
	if err = errors.Join(err, e.Check(ctx)); err != nil {
		return nil, err
	}
	projected = bytes.Clone(projected)
	if !utf8.Valid(projected) || !json.Valid(projected) {
		return nil, findInvalid("descriptor projection must return complete UTF-8 JSON")
	}
	return projected, nil
}

func (e *Engine) matchDescriptor(ctx context.Context, value *Record, key string, filter json.RawMessage) (bool, error) {
	for _, field := range e.profile.Body {
		if field.Name != key || field.Match == nil {
			continue
		}
		if err := e.Check(ctx); err != nil {
			return false, err
		}
		var wire json.RawMessage
		if value.Wire != nil {
			wire = bytes.Clone(value.Wire.Body[field.Wire])
		}
		matched, handled, err := field.Match(bytes.Clone(value.Resource.Body[key]), wire, bytes.Clone(filter))
		if err = errors.Join(err, e.Check(ctx)); err != nil {
			return false, err
		}
		if handled {
			return matched, nil
		}
	}
	return e.localMatch(value.Resource.Body[key], filter)
}
