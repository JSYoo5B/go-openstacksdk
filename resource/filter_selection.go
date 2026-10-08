package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

// FilterSelection is an SDK-owned snapshot of the existing semantic filter
// classifier. Query uses server spellings, Body uses local canonical fields,
// while the original attributes are available to audited member GET bindings.
type FilterSelection struct {
	Query    url.Values
	Body     map[string]json.RawMessage
	original map[string]filterValue
}

// PrepareFilterSelection reuses the collection classifier without doing HTTP.
// Bindings use it when one operation needs original member parameters and a
// transposed, locally filtered list fallback. Unknown list attributes retain
// the ordinary ignored policy; their encoding is checked only if read as
// original member parameters.
func PrepareFilterSelection(descriptor *FilterDescriptor, options ...ListOption) (*FilterSelection, error) {
	selected := listOptions{query: make(url.Values)}
	for _, option := range options {
		if option == nil {
			return nil, invalid("nil semantic filter option")
		}
		if err := option(&selected); err != nil {
			return nil, err
		}
	}
	if selected.name != nil || len(selected.query) != 0 || selected.status || selected.control != (ListControl{}) || len(selected.bodyFilters) != 0 {
		return nil, fmt.Errorf("%w: filter selection accepts semantic WithFilter/WithFilters only", ErrUnsupported)
	}
	bodyFields := make(map[string]string)
	if descriptor != nil {
		for _, key := range descriptor.Body {
			bodyFields[key] = key
		}
	}
	query, body, err := prepareFilters(cloneFilterDescriptor(descriptor), selected.filters, bodyFields)
	if err != nil {
		return nil, err
	}
	original := make(map[string]filterValue)
	for _, update := range selected.filters {
		if update.replace {
			original = make(map[string]filterValue)
		}
		for name, value := range update.values {
			value.raw = bytes.Clone(value.raw)
			original[name] = value
		}
	}
	return &FilterSelection{Query: query, Body: body, original: original}, nil
}

// Attribute retains absent versus explicit null for a declared source policy.
func (s *FilterSelection) Attribute(name string) (json.RawMessage, bool, error) {
	if s == nil {
		return nil, false, invalid("filter selection is required")
	}
	value, present := s.original[name]
	if !present {
		return nil, false, nil
	}
	if err := filterEncodingError(name, value); err != nil {
		return nil, true, err
	}
	return bytes.Clone(value.raw), true, nil
}

// OriginalAttributes provides an owned body seed for a member lookup. It does
// not apply list aliases or discard vendor parameters.
func (s *FilterSelection) OriginalAttributes() (map[string]json.RawMessage, error) {
	if s == nil {
		return nil, invalid("filter selection is required")
	}
	result := make(map[string]json.RawMessage, len(s.original))
	for _, name := range sortedBodyKeys(s.original) {
		raw, _, err := s.Attribute(name)
		if err != nil {
			return nil, err
		}
		result[name] = raw
	}
	return result, nil
}

// OriginalQuery uses the same scalar/array value encoding as the list
// classifier, retaining the caller's original attribute spelling for GET.
func (s *FilterSelection) OriginalQuery() (url.Values, error) {
	attributes, err := s.OriginalAttributes()
	if err != nil {
		return nil, err
	}
	query := make(url.Values, len(attributes))
	for _, name := range sortedBodyKeys(attributes) {
		values, err := encodeFilterQuery(attributes[name])
		if err != nil {
			return nil, fmt.Errorf("%w: member query %q: %w", ErrInvalidOption, name, err)
		}
		query[name] = values
	}
	return query, nil
}
