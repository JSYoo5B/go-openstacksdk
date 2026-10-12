package conntrackhelpers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// CreateOpts owns the provided Source attributes. Absent attributes are not
// sent. Headers and Microversion are explicit Go transport conveniences.
type CreateOpts struct {
	Fields       map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type CreateOption func(*CreateOpts) error

func createPolicy(v CreateOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Fields, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyCreate(v CreateOpts) CreateOpts {
	owned := neutronfind.CloneOptions(createPolicy(v))
	v.Fields, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithCreateOpts(v CreateOpts) CreateOption {
	owned := copyCreate(v)
	return func(c *CreateOpts) error { *c = copyCreate(owned); return nil }
}

// WithCreateField sets one Source attribute. ConntrackHelper does not allow
// unknown Body attributes, so undeclared names are discarded before HTTP.
func WithCreateField(key string, value any) CreateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *CreateOpts) error {
		if err != nil {
			return errors.Join(invalid("create attribute %q is not JSON", key), err)
		}
		c.Fields, c.order = setField(c.Fields, c.order, key, owned)
		return nil
	}
}
func WithCreateHeaders(v map[string]string) CreateOption {
	owned := maps.Clone(v)
	return func(c *CreateOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithCreateMicroversion(v string) CreateOption {
	return func(c *CreateOpts) error { x := v; c.Microversion = &x; return nil }
}

// UpdateOpts owns only changed attributes. No changed attribute is the Source
// commit no-op and returns the ID view without HTTP.
type UpdateOpts struct {
	Fields       map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type UpdateOption func(*UpdateOpts) error

func updatePolicy(v UpdateOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Fields, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyUpdate(v UpdateOpts) UpdateOpts {
	owned := neutronfind.CloneOptions(updatePolicy(v))
	v.Fields, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithUpdateOpts(v UpdateOpts) UpdateOption {
	owned := copyUpdate(v)
	return func(c *UpdateOpts) error { *c = copyUpdate(owned); return nil }
}
func WithUpdateField(key string, value any) UpdateOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *UpdateOpts) error {
		if err != nil {
			return errors.Join(invalid("update attribute %q is not JSON", key), err)
		}
		c.Fields, c.order = setField(c.Fields, c.order, key, owned)
		return nil
	}
}
func WithUpdateHeaders(v map[string]string) UpdateOption {
	owned := maps.Clone(v)
	return func(c *UpdateOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithUpdateMicroversion(v string) UpdateOption {
	return func(c *UpdateOpts) error { x := v; c.Microversion = &x; return nil }
}

// DeleteOpts defaults IgnoreMissing to true, as delete_conntrack_helper.
type DeleteOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}
type DeleteOption func(*DeleteOpts) error

func deletePolicy(v DeleteOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Headers: v.Headers, Microversion: v.Microversion}
}
func copyDelete(v DeleteOpts) DeleteOpts {
	owned := neutronfind.CloneOptions(deletePolicy(v))
	v.IgnoreMissing, v.Headers, v.Microversion = owned.IgnoreMissing, owned.Headers, owned.Microversion
	return v
}
func WithDeleteIgnoreMissing(v bool) DeleteOption {
	return func(c *DeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithDeleteHeaders(v map[string]string) DeleteOption {
	owned := maps.Clone(v)
	return func(c *DeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithDeleteMicroversion(v string) DeleteOption {
	return func(c *DeleteOpts) error { x := v; c.Microversion = &x; return nil }
}

// GetOpts holds explicit Go transport settings. get_conntrack_helper itself
// has no fields or query arguments.
type GetOpts struct {
	Headers      map[string]string
	Microversion *string
}
type GetOption func(*GetOpts) error

func copyGet(v GetOpts) GetOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithGetHeaders(v map[string]string) GetOption {
	owned := maps.Clone(v)
	return func(c *GetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithGetMicroversion(v string) GetOption {
	return func(c *GetOpts) error { x := v; c.Microversion = &x; return nil }
}

// ListOpts owns list kwargs: declared server query fields, other declared Body
// attributes as local filters, and paginated/max_items controls.
type ListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}
type ListOption func(*ListOpts) error

func listPolicy(v ListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyList(v ListOpts) ListOpts {
	owned := neutronfind.CloneOptions(listPolicy(v))
	v.Query, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithListOpts(v ListOpts) ListOption {
	owned := copyList(v)
	return func(c *ListOpts) error { *c = copyList(owned); return nil }
}

// listIgnores reports the unknown kwargs that Resource.list discards before
// evaluating their value. Controls and invalid keys are kept to fail later.
func listIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "router_id", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "cls", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range profile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range profile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}
func WithListQuery(key string, value any) ListOption {
	if listIgnores(key) {
		return func(*ListOpts) error { return nil }
	}
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *ListOpts) error {
		if err != nil {
			return errors.Join(invalid("list query %q is not JSON", key), err)
		}
		c.Query, c.order = setField(c.Query, c.order, key, owned)
		return nil
	}
}
func WithListHeaders(v map[string]string) ListOption {
	owned := maps.Clone(v)
	return func(c *ListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithListMicroversion(v string) ListOption {
	return func(c *ListOpts) error { x := v; c.Microversion = &x; return nil }
}
func WithListLimit(v int) ListOption      { return WithListQuery("limit", v) }
func WithListMarker(v string) ListOption  { return WithListQuery("marker", v) }
func WithListMaxItems(v int) ListOption   { return WithListQuery("max_items", v) }
func WithListPaginated(v bool) ListOption { return WithListQuery("paginated", v) }

func setField(values map[string]json.RawMessage, order []string, key string, raw json.RawMessage) (map[string]json.RawMessage, []string) {
	if values == nil {
		values = make(map[string]json.RawMessage)
	}
	if !slices.Contains(order, key) {
		order = append(order, key)
	}
	values[key] = bytes.Clone(raw)
	return values, order
}

func applyOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, invalid("nil option")
		}
		next := clone(policy)
		err := apply(&next)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, nil
}
