package cinderowned

import (
	"bytes"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// Handlers of the shared request calls over the owned option states. Each
// package composes them into a calls.Spec for its operations; a control a
// package leaves out fails with request.UnsupportedCall before HTTP.

// ReadHeadersCall handles request.Headers over ReadOpts.
var ReadHeadersCall = calls.Value(func(o *ReadOpts) *map[string]string { return &o.Headers })

// WriteHeadersCall handles request.Headers over WriteOpts.
var WriteHeadersCall = calls.Value(func(o *WriteOpts) *map[string]string { return &o.Headers })

// DeleteHeadersCall handles request.Headers over DeleteOpts.
var DeleteHeadersCall = calls.Value(func(o *DeleteOpts) *map[string]string { return &o.Headers })

// DeleteIgnoreMissingCall handles request.IgnoreMissing over DeleteOpts.
var DeleteIgnoreMissingCall = calls.Pointer(func(o *DeleteOpts) **bool { return &o.IgnoreMissing })

// FindIgnoreMissingCall handles request.IgnoreMissing over FindOpts.
var FindIgnoreMissingCall = calls.Pointer(func(o *FindOpts) **bool { return &o.IgnoreMissing })

// WriteFieldCall handles request.Field as one Python attribute. Unknown names
// are ignored together with their marshal error, as Resource.new drops them.
func (s *Spec) WriteFieldCall(o *WriteOpts, entry request.CallEntry) error {
	if _, known := s.Attrs[entry.Key]; !known {
		return nil
	}
	return setEntry(&o.Fields, entry)
}

// ListQueryCall handles request.Query as one Python **query name. Names that
// are neither server query parameters nor local Body filters are ignored
// together with their marshal error.
func ListQueryCall[S any](s *Spec, inner func(*S) *ListOpts) func(*S, request.CallEntry) error {
	return func(v *S, entry request.CallEntry) error {
		_, query := s.Query[entry.Key]
		_, filter := s.Filters[entry.Key]
		if !query && !filter {
			return nil
		}
		return setEntry(&inner(v).Query, entry)
	}
}

// ListHeadersCall handles request.Headers over the ListOpts inside S.
func ListHeadersCall[S any](inner func(*S) *ListOpts) func(*S, map[string]string) error {
	return func(v *S, values map[string]string) error { inner(v).Headers = values; return nil }
}

// ListPaging handles request.Limit, Marker, MaxItems and Paginated over the
// ListOpts inside S.
func ListPaging[S any](inner func(*S) *ListOpts) calls.Paging[S] {
	return calls.Paging[S]{
		Limit:     func(v *S, n int) error { inner(v).Limit = &n; return nil },
		Marker:    func(v *S, marker string) error { inner(v).Marker = marker; return nil },
		MaxItems:  func(v *S, n int) error { inner(v).MaxItems = n; return nil },
		Paginated: func(v *S, paginated bool) error { inner(v).Paginated = &paginated; return nil },
	}
}

// Self returns its argument, for ListOpts states without an outer struct.
func Self[S any](v *S) *S { return v }

// setEntry stores the call's JSON value under its key; a marshal failure
// surfaces when the call is applied.
func setEntry(target *map[string]json.RawMessage, entry request.CallEntry) error {
	if entry.JSONErr != nil {
		return entry.JSONErr
	}
	if *target == nil {
		*target = make(map[string]json.RawMessage)
	}
	(*target)[entry.Key] = bytes.Clone(entry.JSON)
	return nil
}
