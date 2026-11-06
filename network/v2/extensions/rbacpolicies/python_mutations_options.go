package rbacpolicies

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordCreateOpts owns provided Source attributes. Missing attributes do not
// become dirty request fields. Headers and Microversion are Go conveniences.
type RecordCreateOpts struct {
	Attrs        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}

// RecordCreateOption is one option of CreateRecord: WithRecordCreateOpts,
// request.Field (one Source attribute as JSON; later values replace earlier
// ones), request.Headers (replaces the headers), request.Microversion or
// request.Custom over *RecordCreateOpts. request.Header fails with
// resource.ErrInvalidOption before HTTP.
type RecordCreateOption = request.WriteCall

var recordCreateCalls = calls.Spec[RecordCreateOpts]{
	Field: func(c *RecordCreateOpts, entry request.CallEntry) error {
		if entry.JSONErr != nil {
			return errors.Join(rbacRecordInvalid("create attribute %q is not JSON", entry.Key), entry.JSONErr)
		}
		if c.Attrs == nil {
			c.Attrs = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, entry.Key) {
			c.order = append(c.order, entry.Key)
		}
		c.Attrs[entry.Key] = entry.JSON
		return nil
	},
	Headers:      calls.Value(func(c *RecordCreateOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *RecordCreateOpts) **string { return &c.Microversion }),
}

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
	return request.Custom(func(c *RecordCreateOpts) error { *c = copyRBACCreateMutation(owned); return nil })
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

// RecordUpdateOption is one option of UpdateRecord: WithRecordUpdateOpts,
// request.Field (one Source attribute as JSON; later values replace earlier
// ones), request.Headers (replaces the headers), request.Microversion or
// request.Custom over *RecordUpdateOpts. request.Header fails with
// resource.ErrInvalidOption before HTTP.
type RecordUpdateOption = request.WriteCall

var recordUpdateCalls = calls.Spec[RecordUpdateOpts]{
	Field: func(c *RecordUpdateOpts, entry request.CallEntry) error {
		if entry.JSONErr != nil {
			return errors.Join(rbacRecordInvalid("update attribute %q is not JSON", entry.Key), entry.JSONErr)
		}
		if c.Attrs == nil {
			c.Attrs = make(map[string]json.RawMessage)
		}
		if !slices.Contains(c.order, entry.Key) {
			c.order = append(c.order, entry.Key)
		}
		c.Attrs[entry.Key] = entry.JSON
		return nil
	},
	Headers:      calls.Value(func(c *RecordUpdateOpts) *map[string]string { return &c.Headers }),
	Microversion: calls.Pointer(func(c *RecordUpdateOpts) **string { return &c.Microversion }),
}

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
	return request.Custom(func(c *RecordUpdateOpts) error { *c = copyRBACUpdateMutation(owned); return nil })
}

// RecordDeleteOpts defaults IgnoreMissing to true. Only a clean actual404 can
// be ignored; transport, read, Close, Source and cancellation failures persist.
type RecordDeleteOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}

// RecordDeleteOption is one option of DeleteRecord: WithRecordDeleteOpts,
// request.IgnoreMissing, request.Headers (replaces the headers),
// request.Microversion or request.Custom over *RecordDeleteOpts.
// request.Header fails with resource.ErrInvalidOption before HTTP.
type RecordDeleteOption = request.DeleteCall

var recordDeleteCalls = calls.Spec[RecordDeleteOpts]{
	IgnoreMissing: calls.Pointer(func(c *RecordDeleteOpts) **bool { return &c.IgnoreMissing }),
	Headers:       calls.Value(func(c *RecordDeleteOpts) *map[string]string { return &c.Headers }),
	Microversion:  calls.Pointer(func(c *RecordDeleteOpts) **string { return &c.Microversion }),
}

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
	return request.Custom(func(c *RecordDeleteOpts) error { *c = copyRBACDeleteMutation(owned); return nil })
}
