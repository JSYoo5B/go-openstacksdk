package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// FilterDescriptor is supplied by audited SDK bindings. Query maps canonical
// attribute names to wire keys; their wire spellings are also accepted. Body
// maps declared local attribute names to canonical Body-filter fields. Reserved
// names are list controls requiring their dedicated options, not field values.
// Normal SDK callers obtain these policies through a service, without adapters.
type FilterDescriptor struct {
	Query    map[string]string
	Body     map[string]string
	Reserved []string
}

type filterValue struct {
	raw json.RawMessage
	err error
}

type filterUpdate struct {
	replace bool
	values  map[string]filterValue
}

type selectedFilter struct {
	attribute string
	value     filterValue
}

func cloneFilterDescriptor(descriptor *FilterDescriptor) *FilterDescriptor {
	if descriptor == nil {
		return nil
	}
	return &FilterDescriptor{Query: maps.Clone(descriptor.Query), Body: maps.Clone(descriptor.Body), Reserved: slices.Clone(descriptor.Reserved)}
}

func cloneFilterValues(values map[string]filterValue) map[string]filterValue {
	owned := make(map[string]filterValue, len(values))
	for key, value := range values {
		value.raw = append(json.RawMessage(nil), value.raw...)
		owned[key] = value
	}
	return owned
}

// WithFilter adds a declared service attribute. The SDK classifies it as a
// server query or local Body condition and transposes audited query aliases.
// Unknown attributes are discarded. Later individual options for the same
// target win. Values are snapshotted now; validation and HTTP remain lazy.
// Only the final selected values are validated; replacing or clearing a value
// also removes its captured encoding error.
// Raw WithQuery and explicit WithBodyFilter keep their separate namespaces.
func WithFilter(field string, value any) ListOption {
	return filterOption(false, map[string]any{field: value})
}

// WithFilters replaces only the semantic attribute set; nil/empty maps clear
// it. In one bulk map the canonical query attribute wins over its wire alias,
// including null, false and empty values. An explicit clear still requires an
// audited semantic binding. Raw query and explicit Body conditions are retained.
func WithFilters(values map[string]any) ListOption { return filterOption(true, values) }

func filterOption(replace bool, values map[string]any) ListOption {
	frozen := filterUpdate{replace: replace, values: make(map[string]filterValue, len(values))}
	for _, key := range sortedBodyKeys(values) {
		raw, err := json.Marshal(values[key])
		// Classification happens after selecting a binding. Unknown values and a
		// superseded bulk wire alias do not expose their captured encoding errors.
		frozen.values[key] = filterValue{raw: append(json.RawMessage(nil), raw...), err: err}
	}
	return func(o *listOptions) error {
		o.filters = append(o.filters, filterUpdate{replace: frozen.replace, values: cloneFilterValues(frozen.values)})
		return nil
	}
}

func validFilterName(name string) bool { return name != "" && strings.TrimSpace(name) == name }

func validateFilterDescriptor(descriptor *FilterDescriptor, bodyFields map[string]string) error {
	reserved := make(map[string]bool, len(descriptor.Reserved))
	for _, name := range descriptor.Reserved {
		if !validFilterName(name) || reserved[name] {
			return invalid("invalid or duplicate reserved filter name %q", name)
		}
		reserved[name] = true
	}
	owner := make(map[string]string)
	for _, canonical := range sortedBodyKeys(descriptor.Query) {
		wire := descriptor.Query[canonical]
		if !validFilterName(canonical) || !validFilterName(wire) || reserved[canonical] || reserved[wire] {
			return invalid("invalid query filter descriptor %q", canonical)
		}
		if previous, exists := owner[wire]; exists && previous != canonical {
			return invalid("query filter wire key %q has multiple canonical attributes", wire)
		}
		owner[wire] = canonical
		if _, exists := descriptor.Query[wire]; wire != canonical && exists {
			return invalid("query filter alias %q conflicts with a canonical attribute", wire)
		}
	}
	bodyOwner := make(map[string]string)
	for _, name := range sortedBodyKeys(descriptor.Body) {
		field := descriptor.Body[name]
		_, queryCanonical := descriptor.Query[name]
		if !validFilterName(name) || !validFilterName(field) || reserved[name] || queryCanonical || bodyFields[field] != field {
			return invalid("invalid local Body filter descriptor %q", name)
		}
		if previous, exists := bodyOwner[field]; exists && previous != name {
			return invalid("local Body field %q has multiple semantic attributes", field)
		}
		bodyOwner[field] = name
	}
	return nil
}

// prepareFilters is deliberately independent of a native model. Its complete
// descriptor comes from pinned Python declarations rather than native q tags.
func prepareFilters(descriptor *FilterDescriptor, updates []filterUpdate, bodyFields map[string]string) (url.Values, map[string]json.RawMessage, error) {
	if len(updates) == 0 {
		return nil, nil, nil
	}
	if descriptor == nil {
		return nil, nil, fmt.Errorf("%w: semantic field filters", ErrUnsupported)
	}
	if err := validateFilterDescriptor(descriptor, bodyFields); err != nil {
		return nil, nil, err
	}
	queryValues := make(map[string]selectedFilter)
	bodyValues := make(map[string]selectedFilter)
	controls := make(map[string]bool)
	reserved := make(map[string]bool, len(descriptor.Reserved))
	for _, name := range descriptor.Reserved {
		reserved[name] = true
	}
	for _, update := range updates {
		if update.replace {
			queryValues = make(map[string]selectedFilter)
			bodyValues = make(map[string]selectedFilter)
			controls = make(map[string]bool)
		}
		for _, name := range sortedBodyKeys(update.values) {
			if reserved[name] {
				controls[name] = true
			}
		}
		for _, canonical := range sortedBodyKeys(descriptor.Query) {
			wire := descriptor.Query[canonical]
			value, exists := update.values[canonical]
			attribute := canonical
			if !exists && wire != canonical {
				value, exists = update.values[wire]
				attribute = wire
			}
			if !exists {
				continue
			}
			queryValues[wire] = selectedFilter{attribute: attribute, value: value}
		}
		for _, name := range sortedBodyKeys(descriptor.Body) {
			value, exists := update.values[name]
			if !exists {
				continue
			}
			bodyValues[descriptor.Body[name]] = selectedFilter{attribute: name, value: value}
		}
	}
	for _, name := range sortedBodyKeys(controls) {
		switch name {
		case "max_items":
			return nil, nil, invalid("filter max_items requires WithMaxItems")
		case "paginated":
			return nil, nil, invalid("filter paginated requires WithPaginated")
		default:
			return nil, nil, fmt.Errorf("%w: list control %q is not a field filter", ErrUnsupported, name)
		}
	}
	query := make(url.Values, len(queryValues))
	for _, wire := range sortedBodyKeys(queryValues) {
		selected := queryValues[wire]
		if err := filterEncodingError(selected.attribute, selected.value); err != nil {
			return nil, nil, err
		}
		encoded, err := encodeFilterQuery(selected.value.raw)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: query filter %q: %w", ErrInvalidOption, selected.attribute, err)
		}
		// Presence is retained even for null/empty arrays, which produce no
		// URL values. It governs canonical precedence and collisions.
		query[wire] = encoded
	}
	body := make(map[string]json.RawMessage, len(bodyValues))
	for _, field := range sortedBodyKeys(bodyValues) {
		selected := bodyValues[field]
		if err := filterEncodingError(selected.attribute, selected.value); err != nil {
			return nil, nil, err
		}
		body[field] = append(json.RawMessage(nil), selected.value.raw...)
	}
	return query, body, nil
}

func filterEncodingError(name string, value filterValue) error {
	if value.err != nil {
		return fmt.Errorf("%w: filter %q JSON: %w", ErrInvalidOption, name, value.err)
	}
	if !json.Valid(value.raw) {
		return invalid("invalid filter JSON for %q", name)
	}
	return nil
}

func encodeFilterQuery(raw json.RawMessage) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	values, array := value.([]any)
	if !array {
		values = []any{value}
	}
	encoded := make([]string, 0, len(values))
	for _, value := range values {
		switch value := value.(type) {
		case nil:
			continue
		case string:
			encoded = append(encoded, value)
		case bool:
			encoded = append(encoded, strconv.FormatBool(value))
		case json.Number:
			encoded = append(encoded, value.String())
		default:
			return nil, fmt.Errorf("query values must be scalars or scalar arrays, got %T", value)
		}
	}
	return encoded, nil
}

func mergeFilterQuery(options *listOptions, query url.Values, nameHintKey string) error {
	for _, wire := range sortedBodyKeys(query) {
		if _, exists := options.query[wire]; exists {
			return invalid("semantic query %q conflicts with an explicit query or page option", wire)
		}
		if options.name != nil && wire == nameHintKey {
			return invalid("semantic query %q conflicts with WithName's query hint", wire)
		}
	}
	for wire, values := range query {
		options.query[wire] = append([]string(nil), values...)
	}
	return nil
}
