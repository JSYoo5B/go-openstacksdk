package groups

import "maps"

// RecordReplicationOpts owns the enable/disable read-independent action policy.
// PinnedSourceWire reproduces the fixed SDK's null action value; the default
// empty object is the Cinder27 schema-compatible Go correction.
type RecordReplicationOpts struct {
	Microversion     *string
	Headers          map[string]string
	PinnedSourceWire bool
}
type RecordReplicationOption func(*RecordReplicationOpts) error

// RecordFailoverOpts defaults to false/null parameters and the stock action
// key. PinnedSourceWire explicitly reproduces the pinned wrong action key.
type RecordFailoverOpts struct {
	AllowedAttachedVolume *bool
	SecondaryBackendID    *string
	Microversion          *string
	Headers               map[string]string
	PinnedSourceWire      bool
}
type RecordFailoverOption func(*RecordFailoverOpts) error

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
	return func(c *RecordReplicationOpts) error { *c = copyReplication(owned); return nil }
}
func WithRecordReplicationMicroversion(v string) RecordReplicationOption {
	return func(c *RecordReplicationOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordReplicationHeaders(v map[string]string) RecordReplicationOption {
	owned := replicationHeaders(v)
	return func(c *RecordReplicationOpts) error { c.Headers = replicationHeaders(owned); return nil }
}
func WithRecordReplicationPinnedSourceWire(v bool) RecordReplicationOption {
	return func(c *RecordReplicationOpts) error { c.PinnedSourceWire = v; return nil }
}
func WithRecordFailoverOpts(v RecordFailoverOpts) RecordFailoverOption {
	owned := copyFailover(v)
	return func(c *RecordFailoverOpts) error { *c = copyFailover(owned); return nil }
}
func WithRecordFailoverAllowedAttachedVolume(v bool) RecordFailoverOption {
	return func(c *RecordFailoverOpts) error { x := v; c.AllowedAttachedVolume = &x; return nil }
}
func WithRecordFailoverSecondaryBackendID(v string) RecordFailoverOption {
	return func(c *RecordFailoverOpts) error { x := v; c.SecondaryBackendID = &x; return nil }
}
func WithRecordFailoverMicroversion(v string) RecordFailoverOption {
	return func(c *RecordFailoverOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithRecordFailoverHeaders(v map[string]string) RecordFailoverOption {
	owned := replicationHeaders(v)
	return func(c *RecordFailoverOpts) error { c.Headers = replicationHeaders(owned); return nil }
}
func WithRecordFailoverPinnedSourceWire(v bool) RecordFailoverOption {
	return func(c *RecordFailoverOpts) error { c.PinnedSourceWire = v; return nil }
}
