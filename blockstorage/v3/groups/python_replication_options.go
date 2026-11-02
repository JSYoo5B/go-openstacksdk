package groups

import (
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordReplicationOpts owns the enable/disable read-independent action policy.
// PinnedSourceWire reproduces the fixed SDK's null action value; the default
// empty object is the Cinder27 schema-compatible Go correction.
type RecordReplicationOpts struct {
	Microversion     *string
	Headers          map[string]string
	PinnedSourceWire bool
}

// RecordReplicationOption is one option of EnableReplicationRecord and
// DisableReplicationRecord: WithRecordReplicationOpts,
// WithRecordReplicationPinnedSourceWire, request.Custom over
// *RecordReplicationOpts, request.Microversion or request.Headers (replaces
// the headers). Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type RecordReplicationOption = request.WriteCall

// RecordFailoverOpts defaults to false/null parameters and the stock action
// key. PinnedSourceWire explicitly reproduces the pinned wrong action key.
type RecordFailoverOpts struct {
	AllowedAttachedVolume *bool
	SecondaryBackendID    *string
	Microversion          *string
	Headers               map[string]string
	PinnedSourceWire      bool
}

// RecordFailoverOption is one option of FailoverReplicationRecord:
// WithRecordFailoverOpts, WithRecordFailover<Argument>, request.Custom over
// *RecordFailoverOpts, request.Microversion or request.Headers (replaces the
// headers). Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type RecordFailoverOption = request.WriteCall

var (
	replicationCalls = calls.Spec[RecordReplicationOpts]{
		Headers: func(o *RecordReplicationOpts, v map[string]string) error {
			o.Headers = replicationHeaders(v)
			return nil
		},
		Microversion: calls.Pointer(func(o *RecordReplicationOpts) **string { return &o.Microversion }),
	}
	failoverCalls = calls.Spec[RecordFailoverOpts]{
		Headers:      func(o *RecordFailoverOpts, v map[string]string) error { o.Headers = replicationHeaders(v); return nil },
		Microversion: calls.Pointer(func(o *RecordFailoverOpts) **string { return &o.Microversion }),
	}
)

func replicationHeaders(v map[string]string) map[string]string {
	owned := maps.Clone(v)
	if owned == nil {
		owned = make(map[string]string)
	}
	return owned
}
func copyReplication(v RecordReplicationOpts) RecordReplicationOpts {
	v.Headers = replicationHeaders(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func copyFailover(v RecordFailoverOpts) RecordFailoverOpts {
	v.Headers = replicationHeaders(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	if v.AllowedAttachedVolume != nil {
		x := *v.AllowedAttachedVolume
		v.AllowedAttachedVolume = &x
	}
	if v.SecondaryBackendID != nil {
		x := *v.SecondaryBackendID
		v.SecondaryBackendID = &x
	}
	return v
}
func WithRecordReplicationOpts(v RecordReplicationOpts) RecordReplicationOption {
	owned := copyReplication(v)
	return request.Custom(func(c *RecordReplicationOpts) error { *c = copyReplication(owned); return nil })
}
func WithRecordReplicationPinnedSourceWire(v bool) RecordReplicationOption {
	return request.Custom(func(c *RecordReplicationOpts) error { c.PinnedSourceWire = v; return nil })
}
func WithRecordFailoverOpts(v RecordFailoverOpts) RecordFailoverOption {
	owned := copyFailover(v)
	return request.Custom(func(c *RecordFailoverOpts) error { *c = copyFailover(owned); return nil })
}
func WithRecordFailoverAllowedAttachedVolume(v bool) RecordFailoverOption {
	return request.Custom(func(c *RecordFailoverOpts) error { x := v; c.AllowedAttachedVolume = &x; return nil })
}
func WithRecordFailoverSecondaryBackendID(v string) RecordFailoverOption {
	return request.Custom(func(c *RecordFailoverOpts) error { x := v; c.SecondaryBackendID = &x; return nil })
}
func WithRecordFailoverPinnedSourceWire(v bool) RecordFailoverOption {
	return request.Custom(func(c *RecordFailoverOpts) error { c.PinnedSourceWire = v; return nil })
}
