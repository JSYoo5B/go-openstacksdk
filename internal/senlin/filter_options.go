package senlin

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// BodyFilterSpec owns one operation's local response-field namespace. Fields
// maps client aliases and wire names to canonical wire keys. Every canonical
// wire key must also map to itself.
type BodyFilterSpec struct {
	Namespace string
	Fields    map[string]string
}

func snapshotBodyFilterSpec(spec BodyFilterSpec) BodyFilterSpec {
	spec.Fields = maps.Clone(spec.Fields)
	return spec
}

func validateBodyFilterSpec(spec BodyFilterSpec) error {
	if strings.TrimSpace(spec.Namespace) == "" {
		return fmt.Errorf("%w: Body filter namespace is required", resource.ErrInvalidOption)
	}
	for alias, field := range spec.Fields {
		if strings.TrimSpace(alias) == "" || strings.TrimSpace(field) == "" || spec.Fields[field] != field {
			return fmt.Errorf("%w: Body filter descriptor requires canonical field %q", resource.ErrInvalidOption, field)
		}
	}
	return nil
}

// WithBodyFilter snapshots a JSON value when the option is constructed. Later
// options targeting the same canonical field win, including client aliases.
// Each application owns its argument map and all raw JSON backing bytes.
func WithBodyFilter[T any](spec BodyFilterSpec, key string, value any) request.Option[T] {
	spec = snapshotBodyFilterSpec(spec)
	field, known := spec.Fields[key]
	descriptorErr := validateBodyFilterSpec(spec)
	encoded, encodeErr := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if descriptorErr != nil {
			return descriptorErr
		}
		if !known {
			return fmt.Errorf("%w: unknown Body filter %q", resource.ErrInvalidOption, key)
		}
		if encodeErr != nil {
			return fmt.Errorf("%w: Body filter %q JSON: %v", resource.ErrInvalidOption, key, encodeErr)
		}
		previous, _, err := request.Argument[map[string]json.RawMessage](*config, spec.Namespace)
		if err != nil {
			return err
		}
		owned := make(map[string]json.RawMessage, len(previous)+1)
		for key, raw := range previous {
			owned[key] = append(json.RawMessage(nil), raw...)
		}
		owned[field] = append(json.RawMessage(nil), encoded...)
		if config.Arguments == nil {
			config.Arguments = make(map[string]any)
		}
		config.Arguments[spec.Namespace] = owned
		return nil
	}
}

// PrepareBodyFilters validates custom option input and returns an independent
// canonical raw map. A foreign namespace and ambiguous alias/canonical entries
// are rejected rather than relying on map iteration order.
func PrepareBodyFilters[T any](config request.Config[T], spec BodyFilterSpec) (map[string]json.RawMessage, error) {
	if err := validateBodyFilterSpec(spec); err != nil {
		return nil, err
	}
	for key := range config.Arguments {
		if key != spec.Namespace {
			return nil, fmt.Errorf("%w: unsupported Body filter argument %q", resource.ErrInvalidOption, key)
		}
	}
	values, _, err := request.Argument[map[string]json.RawMessage](config, spec.Namespace)
	if err != nil {
		return nil, err
	}
	prepared := make(map[string]json.RawMessage, len(values))
	for key, raw := range values {
		field, known := spec.Fields[key]
		if !known {
			return nil, fmt.Errorf("%w: unknown Body filter %q", resource.ErrInvalidOption, key)
		}
		if !json.Valid(raw) {
			return nil, fmt.Errorf("%w: invalid Body filter JSON for %q", resource.ErrInvalidOption, key)
		}
		if _, exists := prepared[field]; exists {
			return nil, fmt.Errorf("%w: duplicate canonical Body filter %q", resource.ErrInvalidOption, field)
		}
		prepared[field] = append(json.RawMessage(nil), raw...)
	}
	return prepared, nil
}

// RejectBodyFilterQuery keeps every known Body field, including client aliases,
// out of extension queries. Local filtering requires the dedicated option.
func RejectBodyFilterQuery(query url.Values, spec BodyFilterSpec) error {
	if err := validateBodyFilterSpec(spec); err != nil {
		return err
	}
	for key := range query {
		if _, known := spec.Fields[key]; known {
			return fmt.Errorf("%w: Body query %q requires WithListFilter", resource.ErrInvalidOption, key)
		}
	}
	return nil
}
