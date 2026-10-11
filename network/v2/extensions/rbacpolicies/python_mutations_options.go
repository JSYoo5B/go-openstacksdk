package rbacpolicies

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// RecordCreateOpts owns provided Source attributes. Missing attributes do not
// become dirty request fields. Headers and Microversion are Go conveniences.
type RecordCreateOpts struct {
	Attrs        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type RecordCreateOption func(*RecordCreateOpts) error

func rbacCreateMutationPolicy(v RecordCreateOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Attrs, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyRBACCreateMutation(v RecordCreateOpts) RecordCreateOpts {
	owned := neutronfind.CloneOptions(rbacCreateMutationPolicy(v))
	v.Attrs, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithRecordCreateOpts(v RecordCreateOpts) RecordCreateOption {
	owned := copyRBACCreateMutation(v)
	return func(c *RecordCreateOpts) error { *c = copyRBACCreateMutation(owned); return nil }
}
func WithRecordCreateField(key string, value any) RecordCreateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordCreateOpts) error {
		if err != nil {
			return errors.Join(rbacRecordInvalid("create attribute %q is not JSON", key), err)
		}
		if c.Attrs == nil {
			c.Attrs = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Attrs[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordCreateHeaders(v map[string]string) RecordCreateOption {
	owned := maps.Clone(v)
	return func(c *RecordCreateOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordCreateMicroversion(v string) RecordCreateOption {
	return func(c *RecordCreateOpts) error { x := v; c.Microversion = &x; return nil }
}

// RecordUpdateOpts owns only changed attributes for an immutable literal-ID
// update. No dirty attributes produce the Source no-op, even with safe
// transport header/version options.
type RecordUpdateOpts struct {
	Attrs        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type RecordUpdateOption func(*RecordUpdateOpts) error

func rbacUpdateMutationPolicy(v RecordUpdateOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Attrs, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyRBACUpdateMutation(v RecordUpdateOpts) RecordUpdateOpts {
	owned := neutronfind.CloneOptions(rbacUpdateMutationPolicy(v))
	v.Attrs, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithRecordUpdateOpts(v RecordUpdateOpts) RecordUpdateOption {
	owned := copyRBACUpdateMutation(v)
	return func(c *RecordUpdateOpts) error { *c = copyRBACUpdateMutation(owned); return nil }
}
func WithRecordUpdateField(key string, value any) RecordUpdateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordUpdateOpts) error {
		if err != nil {
			return errors.Join(rbacRecordInvalid("update attribute %q is not JSON", key), err)
		}
		if c.Attrs == nil {
			c.Attrs = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, key) {
			c.order = append(c.order, key)
		}
		c.Attrs[key] = bytes.Clone(owned)
		return nil
	}
}
func WithRecordUpdateHeaders(v map[string]string) RecordUpdateOption {
	owned := maps.Clone(v)
	return func(c *RecordUpdateOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordUpdateMicroversion(v string) RecordUpdateOption {
	return func(c *RecordUpdateOpts) error { x := v; c.Microversion = &x; return nil }
}

// RecordDeleteOpts defaults IgnoreMissing to true. Only a clean actual404 can
// be ignored; transport, read, Close, Source and cancellation failures persist.
type RecordDeleteOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}
type RecordDeleteOption func(*RecordDeleteOpts) error

func rbacDeleteMutationPolicy(v RecordDeleteOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Headers: v.Headers, Microversion: v.Microversion}
}
func copyRBACDeleteMutation(v RecordDeleteOpts) RecordDeleteOpts {
	owned := neutronfind.CloneOptions(rbacDeleteMutationPolicy(v))
	v.IgnoreMissing, v.Headers, v.Microversion = owned.IgnoreMissing, owned.Headers, owned.Microversion
	return v
}
func WithRecordDeleteOpts(v RecordDeleteOpts) RecordDeleteOption {
	owned := copyRBACDeleteMutation(v)
	return func(c *RecordDeleteOpts) error { *c = copyRBACDeleteMutation(owned); return nil }
}
func WithRecordDeleteIgnoreMissing(v bool) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordDeleteHeaders(v map[string]string) RecordDeleteOption {
	owned := maps.Clone(v)
	return func(c *RecordDeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordDeleteMicroversion(v string) RecordDeleteOption {
	return func(c *RecordDeleteOpts) error { x := v; c.Microversion = &x; return nil }
}
