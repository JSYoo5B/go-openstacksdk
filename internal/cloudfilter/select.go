package cloudfilter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/jmespath"
)

// Result contains an owned normalized JSON value. Ordinary selection retains
// source row indices; expression output is arbitrary JSON and has no invented
// source-row association.
type Result struct {
	Value      json.RawMessage
	Indices    []int
	Expression bool
}

// Select applies the identifier phase before inspecting filters. Rows must
// already be normalized JSON objects. Nonempty string filters evaluate the
// full JMESPath engine even when identifier selection is empty. Other truthy
// filters are consumed lazily, in raw JSON object insertion order.
func Select(rows []json.RawMessage, nameOrID string, filters *json.RawMessage, guard func() error) (result Result, err error) {
	defer func() {
		if observed := check(guard); observed != nil {
			result = Result{}
			err = errors.Join(err, observed)
		}
	}()
	owned := make([]json.RawMessage, len(rows))
	for i, row := range rows {
		owned[i] = bytes.Clone(row)
	}
	var supplied *json.RawMessage
	if filters != nil {
		value := json.RawMessage(bytes.Clone(*filters))
		supplied = &value
	}
	if err := check(guard); err != nil {
		return Result{}, err
	}
	views := make([]*node, len(owned))
	for i, raw := range owned {
		if err := check(guard); err != nil {
			return Result{}, err
		}
		view, err := parse(raw)
		if err != nil {
			return Result{}, fmt.Errorf("row %d: %w", i, err)
		}
		if view.kind != '{' {
			return Result{}, fmt.Errorf("row %d must be a JSON object", i)
		}
		views[i] = view
	}
	indices, err := identify(views, nameOrID, guard)
	if err != nil {
		return Result{}, err
	}
	if err := check(guard); err != nil {
		return Result{}, err
	}
	if supplied == nil {
		return selected(views, indices, guard)
	}
	filter, err := parse(*supplied)
	if err != nil {
		return Result{}, fmt.Errorf("filters: %w", err)
	}
	if !filter.truthy() {
		return selected(views, indices, guard)
	}
	if filter.kind == '"' {
		return expression(views, indices, filter.text, guard)
	}
	matched := make([]int, 0, len(indices))
	for _, index := range indices {
		matches, err := mapping(filter, views[index], "$", guard)
		if err != nil {
			return Result{}, err
		}
		if matches {
			matched = append(matched, index)
		}
	}
	return selected(views, matched, guard)
}

func identify(views []*node, pattern string, guard func() error) ([]int, error) {
	indices := make([]int, 0, len(views))
	var tokens []globToken
	if pattern != "" {
		var err error
		tokens, err = compileGlob(pattern)
		if err != nil {
			return nil, err
		}
	}
	absent := &node{raw: json.RawMessage("null"), kind: 'n'}
	for i, view := range views {
		if err := check(guard); err != nil {
			return nil, err
		}
		if pattern == "" {
			indices = append(indices, i)
			continue
		}
		id, name := view.fields["id"], view.fields["name"]
		if id == nil {
			id = absent
		}
		if name == nil {
			name = absent
		}
		idText, err := pythonString(id, guard)
		if err != nil {
			return nil, err
		}
		nameText, err := pythonString(name, guard)
		if err != nil {
			return nil, err
		}
		// Exact matching is a per-row shortcut. Other rows may still match
		// the same pattern as a wildcard; duplicates are never removed.
		matches := idText != "" && idText == pattern || nameText != "" && nameText == pattern
		if !matches && idText != "" {
			matches, err = matchGlob(tokens, idText, guard)
			if err != nil {
				return nil, err
			}
		}
		if !matches && nameText != "" {
			matches, err = matchGlob(tokens, nameText, guard)
			if err != nil {
				return nil, err
			}
		}
		if matches {
			indices = append(indices, i)
		}
	}
	return indices, check(guard)
}

func selected(views []*node, indices []int, guard func() error) (Result, error) {
	if err := check(guard); err != nil {
		return Result{}, err
	}
	rows := make([]json.RawMessage, len(indices))
	ownedIndices := make([]int, len(indices))
	copy(ownedIndices, indices)
	for i, index := range indices {
		rows[i] = views[index].raw
	}
	value, err := json.Marshal(rows)
	if err != nil {
		return Result{}, err
	}
	if err := check(guard); err != nil {
		return Result{}, err
	}
	return Result{Value: value, Indices: ownedIndices}, nil
}

func expression(views []*node, indices []int, source string, guard func() error) (Result, error) {
	input, err := selected(views, indices, guard)
	if err != nil {
		return Result{}, err
	}
	d := json.NewDecoder(bytes.NewReader(input.Value))
	d.UseNumber()
	var data any
	if err := d.Decode(&data); err != nil {
		return Result{}, err
	}
	if err := check(guard); err != nil {
		return Result{}, err
	}
	output, err := jmespath.Search(source, data)
	if err != nil {
		return Result{}, err
	}
	if err := check(guard); err != nil {
		return Result{}, err
	}
	value, err := json.Marshal(output)
	if err != nil {
		return Result{}, fmt.Errorf("expression result JSON: %w", err)
	}
	if err := check(guard); err != nil {
		return Result{}, err
	}
	return Result{Value: value, Expression: true}, nil
}
