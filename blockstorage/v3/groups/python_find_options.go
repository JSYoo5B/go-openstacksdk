package groups

import "maps"

// RecordFindOpts keeps find_group's IgnoreMissing/Details defaults true.
// Microversion is a GET-only inherited Resource.find override; fallback uses
// the original selected version. Headers are an explicit ordinary Go extension.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Details       *bool
	Microversion  *string
	Headers       map[string]string
}
type RecordFindOption func(*RecordFindOpts) error

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
	return func(c *RecordFindOpts) error { *c = copyFindOpts(owned); return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindDetails(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Details = &x; return nil }
}
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
