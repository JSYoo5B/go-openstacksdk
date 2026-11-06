package sfcrecord

import (
	"bytes"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ReadCalls interprets the shared calls of a get: request.Headers replaces
// the per-request headers. Other shared calls are unsupported.
var ReadCalls = calls.Spec[ReadOpts]{
	Headers: calls.Value(func(c *ReadOpts) *map[string]string { return &c.Headers }),
}

// WriteCalls interprets the shared calls of a create or update:
// request.Field records one Python attribute (unknown names are dropped
// together with their marshal error, as Resource.new drops them) and
// request.Headers replaces the per-request headers.
func (s *Spec) WriteCalls() calls.Spec[WriteOpts] {
	return calls.Spec[WriteOpts]{
		Field: func(c *WriteOpts, entry request.CallEntry) error {
			if _, known := s.Attrs[entry.Key]; !known {
				return nil
			}
			return setRaw(&c.Fields, entry.Key, entry.JSON, entry.JSONErr)
		},
		Headers: calls.Value(func(c *WriteOpts) *map[string]string { return &c.Headers }),
	}
}

// DeleteCalls interprets request.IgnoreMissing and request.Headers (replaces
// the per-request headers) of a delete.
var DeleteCalls = calls.Spec[DeleteOpts]{
	IgnoreMissing: calls.Pointer(func(c *DeleteOpts) **bool { return &c.IgnoreMissing }),
	Headers:       calls.Value(func(c *DeleteOpts) *map[string]string { return &c.Headers }),
}

// FindCalls interprets request.IgnoreMissing and request.Query (one Python
// find **query name as JSON; reserved names fail when Find runs) of a find.
var FindCalls = calls.Spec[FindOpts]{
	IgnoreMissing: calls.Pointer(func(c *FindOpts) **bool { return &c.IgnoreMissing }),
	Query: calls.JSONQuery(func(c *FindOpts, key string, raw json.RawMessage, err error) error {
		return setRaw(&c.Query, key, raw, err)
	}),
}

// ListCalls interprets the shared calls of a list: request.Query records one
// Python **query name as JSON (names that are neither server query
// parameters nor local Body filters are dropped together with their marshal
// error), request.Limit, request.Marker, request.MaxItems,
// request.Paginated and request.Headers (replaces the per-request headers).
func (s *Spec) ListCalls() calls.Spec[ListOpts] {
	return calls.Spec[ListOpts]{
		Query: calls.JSONQuery(func(c *ListOpts, key string, raw json.RawMessage, err error) error {
			_, query := s.Query[key]
			_, filter := s.Filters[key]
			if !query && !filter {
				return nil
			}
			return setRaw(&c.Query, key, raw, err)
		}),
		Headers: calls.Value(func(c *ListOpts) *map[string]string { return &c.Headers }),
		Paging: calls.Paging[ListOpts]{
			Limit:     calls.Pointer(func(c *ListOpts) **int { return &c.Limit }),
			Marker:    calls.Value(func(c *ListOpts) *string { return &c.Marker }),
			MaxItems:  calls.Value(func(c *ListOpts) *int { return &c.MaxItems }),
			Paginated: calls.Pointer(func(c *ListOpts) **bool { return &c.Paginated }),
		},
	}
}

func setRaw(target *map[string]json.RawMessage, key string, raw json.RawMessage, err error) error {
	owned := bytes.Clone(raw)
	return SetField(target, key, func() (json.RawMessage, error) { return owned, err })
}
