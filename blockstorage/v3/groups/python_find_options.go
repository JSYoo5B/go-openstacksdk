package groups

import (
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordFindOpts keeps find_group's IgnoreMissing/Details defaults true.
// Microversion is a GET-only inherited Resource.find override; fallback uses
// the original selected version. Headers are an explicit ordinary Go extension.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Details       *bool
	Microversion  *string
	Headers       map[string]string
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// WithRecordFindDetails, request.Custom over *RecordFindOpts,
// request.IgnoreMissing, request.Microversion or request.Headers (replaces
// the headers). Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordFindOption = request.FindCall

var findCalls = calls.Spec[RecordFindOpts]{
	Headers:       calls.Value(func(o *RecordFindOpts) *map[string]string { return &o.Headers }),
	Microversion:  calls.Pointer(func(o *RecordFindOpts) **string { return &o.Microversion }),
	IgnoreMissing: calls.Pointer(func(o *RecordFindOpts) **bool { return &o.IgnoreMissing }),
}

func copyFindOpts(v RecordFindOpts) RecordFindOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyFindOpts(v)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil })
}
func WithRecordFindDetails(v bool) RecordFindOption {
	return request.Custom(func(c *RecordFindOpts) error { x := v; c.Details = &x; return nil })
}
