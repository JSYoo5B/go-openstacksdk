package groupsnapshots

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
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

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// WithRecordFindDetails, WithRecordFindPaginated, WithRecordFindMaxItems,
// WithRecordFindLocation, request.Custom over *RecordFindOpts, request.Query
// (a GET query field as JSON, including unknown keys; reserved constructor
// and control keys fail), request.IgnoreMissing, request.Microversion or
// request.Headers (replaces the headers). Other shared calls fail with
// resource.ErrInvalidOption before HTTP. Find paging stays
// operation-specific: no shared category admits IgnoreMissing with paging.
type RecordFindOption = request.FindCall

var findCalls = calls.Spec[RecordFindOpts]{
	Query: calls.JSONQuery(func(o *RecordFindOpts, key string, raw json.RawMessage, err error) error {
		if keyErr := findQueryKey(key); keyErr != nil {
			return keyErr
		}
		if err != nil {
			return err
		}
		if o.Query == nil {
			o.Query = make(map[string]json.RawMessage)
		}
		o.Query[key] = bytes.Clone(raw)
		return nil
	}),
	Headers:       func(o *RecordFindOpts, v map[string]string) error { o.Headers = recordHeaders(v); return nil },
	Microversion:  calls.Pointer(func(o *RecordFindOpts) **string { return &o.Microversion }),
	IgnoreMissing: calls.Pointer(func(o *RecordFindOpts) **bool { return &o.IgnoreMissing }),
}

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
	return request.Custom(func(c *RecordFindOpts) error { *c = copyFind(owned); return nil })
}
func WithRecordFindDetails(v bool) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { x := v; c.Details = &x; return nil })
}
func WithRecordFindPaginated(v bool) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { x := v; c.Paginated = &x; return nil })
}
func WithRecordFindMaxItems(v int) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { c.MaxItems = v; return nil })
}
func WithRecordFindLocation(v resource.CloudLocation) RecordFindOption {
	owned := v.Clone()
	return request.Custom(func(c *RecordFindOpts) error { c.Location = cloneLocation(&owned); return nil })
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
