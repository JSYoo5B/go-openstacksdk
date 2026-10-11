package groupsnapshots

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// RecordFindOpts owns named Source defaults and explicit Go read extensions.
// Microversion overrides the member GET only; fallback reuses the selected
// service version, or independently discovers the Source 3.29 ceiling.
// Query exposes a bounded Resource.find parameter profile; the named Python
// proxy itself accepts only identity, ignore_missing and details.
type RecordFindOpts struct {
	IgnoreMissing, Details, Paginated *bool
	Headers                           map[string]string
	Microversion                      *string
	Location                          *resource.CloudLocation
	Query                             map[string]json.RawMessage
	MaxItems                          int
}
type RecordFindOption func(*RecordFindOpts) error

func copyFind(v RecordFindOpts) RecordFindOpts {
	v.IgnoreMissing = cloneBool(v.IgnoreMissing)
	v.Details = cloneBool(v.Details)
	v.Paginated = cloneBool(v.Paginated)
	v.Headers = recordHeaders(v.Headers)
	v.Microversion = cloneString(v.Microversion)
	v.Location = cloneLocation(v.Location)
	v.Query = recordClone(v.Query)
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyFind(v)
	return func(c *RecordFindOpts) error { *c = copyFind(owned); return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindDetails(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Details = &x; return nil }
}
func WithRecordFindPaginated(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Paginated = &x; return nil }
}
func WithRecordFindMaxItems(v int) RecordFindOption {
	return func(c *RecordFindOpts) error { c.MaxItems = v; return nil }
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := recordHeaders(v)
	return func(c *RecordFindOpts) error { c.Headers = recordHeaders(owned); return nil }
}
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindLocation(v resource.CloudLocation) RecordFindOption {
	owned := v.Clone()
	return func(c *RecordFindOpts) error { c.Location = cloneLocation(&owned); return nil }
}
func findQueryKey(key string) error {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return recordInvalid("find query key requires nonblank UTF-8")
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return recordInvalid("find query key contains controls")
		}
	}
	switch key {
	case "id", "cls", "self", "connection", "_synchronized":
		return recordInvalid("group snapshot find constructor collides with %q", key)
	case "session", "name_or_id", "ignore_missing", "details", "microversion", "base_path", "list_base_path", "requires_id", "error_message", "skip_cache", "resource_response_key", "headers", "location", "paginated", "max_items", "__conflicting_attrs":
		return recordInvalid("group snapshot find control %q requires an owned option or is outside the fixed route profile", key)
	}
	return nil
}

// WithRecordFindQuery snapshots JSON when created, including unknown GET query
// fields. Reserved constructor/control keys fail without invoking a Marshaler.
func WithRecordFindQuery(key string, value any) RecordFindOption {
	if err := findQueryKey(key); err != nil {
		return func(*RecordFindOpts) error { return err }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		return nil
	}
}
