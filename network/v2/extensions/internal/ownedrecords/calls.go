package ownedrecords

import (
	"bytes"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// setRaw records one JSON value marshaled when the call was constructed; a
// marshal error surfaces when the call is applied, before HTTP.
func setRaw(label string, target *map[string]json.RawMessage, order *[]string, key string, raw json.RawMessage, err error) error {
	if err != nil {
		return errors.Join(Invalid("%s %q is not JSON", label, key), err)
	}
	if *target == nil {
		*target = make(map[string]json.RawMessage)
	}
	if !slices.Contains(*order, key) {
		*order = append(*order, key)
	}
	(*target)[key] = bytes.Clone(raw)
	return nil
}

// GetCalls interprets the shared calls of a get: request.Headers replaces
// the per-request headers and request.Microversion selects the
// microversion. request.Header is unsupported.
var GetCalls = calls.Spec[GetOpts]{
	Headers:      calls.Value(func(c *GetOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *GetOpts) **string { return &c.Microversion }),
}

// WriteCalls interprets the shared calls of a create or update:
// request.Field records one Python attribute, request.Headers replaces the
// headers and request.Microversion selects the microversion.
var WriteCalls = calls.Spec[WriteOpts]{
	Field: func(c *WriteOpts, entry request.CallEntry) error {
		return setRaw("attribute", &c.Attrs, &c.order, entry.Key, entry.JSON, entry.JSONErr)
	},
	Headers:      calls.Value(func(c *WriteOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *WriteOpts) **string { return &c.Microversion }),
}

// DeleteCalls interprets request.IgnoreMissing, request.Headers (replaces
// the headers) and request.Microversion of a delete.
var DeleteCalls = calls.Spec[DeleteOpts]{
	IgnoreMissing: calls.Pointer(func(c *DeleteOpts) **bool { return &c.IgnoreMissing }),
	Headers:       calls.Value(func(c *DeleteOpts) *map[string]string { return &c.Headers }),
	Microversion:  calls.Pointer(func(c *DeleteOpts) **string { return &c.Microversion }),
}

// FindCalls interprets the shared calls of a find: request.IgnoreMissing,
// request.Query (one Python **query value as JSON), request.Headers
// (replaces the headers) and request.Microversion.
var FindCalls = calls.Spec[FindOpts]{
	IgnoreMissing: calls.Pointer(func(c *FindOpts) **bool { return &c.IgnoreMissing }),
	Query: calls.JSONQuery(func(c *FindOpts, key string, raw json.RawMessage, err error) error {
		return setRaw("find query", &c.Query, &c.order, key, raw, err)
	}),
	Headers:      calls.Value(func(c *FindOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *FindOpts) **string { return &c.Microversion }),
}

// ActionCalls interprets request.Headers (replaces the headers) and
// request.Microversion of a resource action. request.Header is unsupported.
var ActionCalls = calls.Spec[ActionOpts]{
	Headers:      calls.Value(func(c *ActionOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *ActionOpts) **string { return &c.Microversion }),
}

// ListCalls interprets the shared calls of a list: request.Query records one
// Python **query name as JSON (unknown names are dropped together with their
// marshal error, as Resource.list drops them), request.Limit,
// request.Marker, request.MaxItems and request.Paginated are the limit,
// marker, max_items and paginated query names, request.Headers replaces the
// headers and request.Microversion selects the microversion.
func (s *Spec) ListCalls() calls.Spec[ListOpts] {
	query := func(c *ListOpts, key string, raw json.RawMessage, err error) error {
		if s.listIgnores(key) {
			return nil
		}
		return setRaw("list query", &c.Query, &c.order, key, raw, err)
	}
	return calls.Spec[ListOpts]{
		Query:        calls.JSONQuery(query),
		Paging:       calls.JSONPaging(query),
		Headers:      calls.Value(func(c *ListOpts) *map[string]string { return &c.Headers }),
		Microversion: calls.Pointer(func(c *ListOpts) **string { return &c.Microversion }),
	}
}
