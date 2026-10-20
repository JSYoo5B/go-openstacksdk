package resource

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

type bodyFilterUpdate struct {
	replace bool
	values  map[string]json.RawMessage
}

// WithBodyFilter adds a local response-field filter to an audited collection.
// The value is snapshotted as JSON now; options are validated lazily before HTTP.
// Later options for the same SDK-owned canonical field win. Arrays compare in
// order with exact length and nested values; no scalar coercion is performed.
// WithQuery remains independent and passes its value to the server.
func WithBodyFilter(field string, value any) ListOption {
	return bodyFilterOption(false, map[string]any{field: value})
}

// WithBodyFilters replaces the entire local filter set. Nil and empty maps clear
// it, while an individual nil value matches native null. This explicit option
// still requires a Body-filter-capable binding, even when clearing filters.
func WithBodyFilters(values map[string]any) ListOption {
	return bodyFilterOption(true, values)
}

func bodyFilterOption(replace bool, values map[string]any) ListOption {
	frozen := bodyFilterUpdate{replace: replace, values: make(map[string]json.RawMessage, len(values))}
	keys := sortedBodyKeys(values)
	var capturedErr error
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			capturedErr = invalid("Body filter field must not be empty")
			break
		}
		raw, err := json.Marshal(values[key])
		if err != nil {
			capturedErr = fmt.Errorf("%w: Body filter %q JSON: %w", ErrInvalidOption, key, err)
			break
		}
		frozen.values[key] = append(json.RawMessage(nil), raw...)
	}
	return func(o *listOptions) error {
		if capturedErr != nil {
			return capturedErr
		}
		owned := bodyFilterUpdate{replace: frozen.replace, values: cloneBodyFilters(frozen.values)}
		o.bodyFilters = append(o.bodyFilters, owned)
		return nil
	}
}

func sortedBodyKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func cloneBodyFilters(values map[string]json.RawMessage) map[string]json.RawMessage {
	owned := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		owned[key] = append(json.RawMessage(nil), raw...)
	}
	return owned
}

func (c *Collection[T]) prepareBodyFilters(updates []bodyFilterUpdate) (map[string]json.RawMessage, error) {
	if len(updates) == 0 {
		return nil, nil
	}
	fields := c.binding.BodyFilterFields
	rawProjection := c.binding.BodyFilterRecordValue != nil
	if len(fields) == 0 || (c.binding.BodyFilterValue == nil && !rawProjection) || (rawProjection && c.binding.IterateBodyControlled == nil) {
		return nil, c.wrap("list", ErrUnsupported)
	}
	for _, alias := range sortedBodyKeys(fields) {
		field := fields[alias]
		if strings.TrimSpace(alias) == "" || strings.TrimSpace(field) == "" || fields[field] != field {
			return nil, invalid("Body filter descriptor requires canonical field %q", field)
		}
	}
	prepared := make(map[string]json.RawMessage)
	for _, update := range updates {
		if update.replace {
			prepared = make(map[string]json.RawMessage, len(update.values))
		}
		seen := make(map[string]bool, len(update.values))
		for _, alias := range sortedBodyKeys(update.values) {
			field, known := fields[alias]
			if !known {
				return nil, invalid("unknown Body filter %q", alias)
			}
			if seen[field] {
				return nil, invalid("duplicate canonical Body filter %q", field)
			}
			seen[field] = true
			raw := update.values[alias]
			if !json.Valid(raw) {
				return nil, invalid("invalid Body filter JSON for %q", alias)
			}
			prepared[field] = append(json.RawMessage(nil), raw...)
		}
	}
	return prepared, nil
}

func (c *Collection[T]) matchBodyFilters(value *T, filters map[string]json.RawMessage) (bool, error) {
	if len(filters) == 0 {
		return true, nil
	}
	body := make(map[string]json.RawMessage, len(filters))
	for _, field := range sortedBodyKeys(filters) {
		raw, err := c.binding.BodyFilterValue(value, field)
		if err != nil {
			return false, fmt.Errorf("Body filter field %q: %w", field, err)
		}
		body[field] = raw
	}
	return jsonfilter.MatchFilters(body, filters)
}

func (c *Collection[T]) matchBodyRecordFilters(record *BodyRecord[T], filters map[string]json.RawMessage) (bool, error) {
	body := make(map[string]json.RawMessage, len(filters))
	for _, field := range sortedBodyKeys(filters) {
		raw, err := c.binding.BodyFilterRecordValue(record, field)
		if err != nil {
			return false, fmt.Errorf("Body filter field %q: %w", field, err)
		}
		body[field] = raw
	}
	return jsonfilter.MatchFilters(body, filters)
}
