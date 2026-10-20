package resourcefilters

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordListOpts names controls that Python passes as kwargs. Only resource
// is a declared server query; filters/id/name are local Body predicates.
// Arbitrary unknown kwargs are discarded without invoking their marshalers.
type RecordListOpts struct {
	Query        map[string]any
	Headers      map[string]string
	Microversion *string
	Control      resource.ListControl
	JMESPath     *string
}
type RecordListOption func(*RecordListOpts) error

func cloneOptions(value RecordListOpts) RecordListOpts {
	value.Query = maps.Clone(value.Query)
	if value.Query == nil {
		value.Query = make(map[string]any)
	}
	for k, v := range value.Query {
		if raw, ok := v.(json.RawMessage); ok {
			value.Query[k] = json.RawMessage(bytes.Clone(raw))
		}
	}
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	if value.Microversion != nil {
		v := *value.Microversion
		value.Microversion = &v
	}
	if value.JMESPath != nil {
		v := *value.JMESPath
		value.JMESPath = &v
	}
	return value
}
func freezeQuery(value *RecordListOpts) error {
	for key := range value.Query {
		switch key {
		case "cls", "self", "session", "base_path", "list_base_path", "microversion", "headers", "__conflicting_attrs", "max_items", "paginated", "allow_unknown_params":
			return invalid("list control %q requires a typed option", key)
		}
	}
	for _, key := range []string{"resource", "filters", "id", "name", "jmespath_filters"} {
		if v, exists := value.Query[key]; exists {
			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			if !utf8.Valid(raw) {
				return invalid("query field %q must be UTF-8", key)
			}
			value.Query[key] = json.RawMessage(bytes.Clone(raw))
		}
	}
	return nil
}
func WithRecordListOptions(value RecordListOpts) RecordListOption {
	owned := cloneOptions(value)
	err := freezeQuery(&owned)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		*c = cloneOptions(owned)
		return nil
	}
}
func WithRecordListQuery(value map[string]any) RecordListOption {
	owned := RecordListOpts{Query: maps.Clone(value)}
	err := freezeQuery(&owned)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		c.Query = cloneOptions(owned).Query
		return nil
	}
}
func WithRecordListResource(value string) RecordListOption {
	return func(c *RecordListOpts) error { c.Query["resource"] = value; return nil }
}
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordListMicroversion(value string) RecordListOption {
	return func(c *RecordListOpts) error { c.Microversion = &value; return nil }
}
func WithRecordListControl(value resource.ListControl) RecordListOption {
	return func(c *RecordListOpts) error { c.Control = value; return nil }
}

// WithRecordListJMESPath selects the Proxy expression. ProxyListRecords keeps
// the opaque Source input; ListResult explicitly evaluates collected JSON rows.
// As in the pinned Proxy, an empty expression means no transformation.
func WithRecordListJMESPath(value string) RecordListOption {
	return func(c *RecordListOpts) error {
		if value == "" {
			c.JMESPath = nil
		} else {
			c.JMESPath = &value
		}
		return nil
	}
}
func copyCallbacks(values []RecordListOption) []RecordListOption { return slices.Clone(values) }

// WithRecordListProxyJMESValue retains JSON falsey/non-string kwargs as Source
// Proxy._list does, rather than coercing a bool/list/object into an expression.
func WithRecordListProxyJMESValue(value any) RecordListOption {
	raw, err := json.Marshal(value)
	owned := json.RawMessage(bytes.Clone(raw))
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		c.Query["jmespath_filters"] = json.RawMessage(bytes.Clone(owned))
		return nil
	}
}
