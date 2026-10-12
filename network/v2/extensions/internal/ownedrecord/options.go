package ownedrecord

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

// GetOpts is the pinned get_* fields selector plus per-request headers and
// microversion, which are Go extensions.
type GetOpts struct {
	Fields       []string
	Microversion *string
	Headers      map[string]string
}

// WriteOpts carries Python **attrs as raw JSON. Keys are Python attribute
// names or wire names; unknown keys stay in the body, as the pinned classes
// set _allow_unknown_attrs_in_body. Headers and Microversion are Go extensions.
type WriteOpts struct {
	Attrs        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}

// DeleteOpts selects ignore_missing (nil means the Python default true).
type DeleteOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}

// FindOpts carries ignore_missing (nil means true) and Python **query, which
// Resource.find forwards to the member GET and to the fallback list.
type FindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

// ListOpts carries Python **query: declared server query names, Body names
// filtered locally, and the limit, marker, max_items and paginated controls.
type ListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

func copyVersion(v *string) *string {
	if v == nil {
		return nil
	}
	x := *v
	return &x
}

func CopyGet(v GetOpts) GetOpts {
	v.Fields, v.Headers, v.Microversion = slices.Clone(v.Fields), maps.Clone(v.Headers), copyVersion(v.Microversion)
	return v
}
func CopyWrite(v WriteOpts) WriteOpts {
	owned := neutronfind.CloneOptions(neutronfind.Options{Query: v.Attrs, Headers: v.Headers, Microversion: v.Microversion, Order: v.order})
	v.Attrs, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func CopyDelete(v DeleteOpts) DeleteOpts {
	owned := neutronfind.CloneOptions(neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Headers: v.Headers, Microversion: v.Microversion})
	v.IgnoreMissing, v.Headers, v.Microversion = owned.IgnoreMissing, owned.Headers, owned.Microversion
	return v
}
func CopyFind(v FindOpts) FindOpts {
	owned := neutronfind.CloneOptions(findOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func CopyList(v ListOpts) ListOpts {
	owned := neutronfind.CloneOptions(listOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}

func findOptions(v FindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func listOptions(v ListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}

// marshal snapshots one option value; a marshal error surfaces when applied.
func marshal(value any) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	return bytes.Clone(raw), err
}

func setOrdered(target *map[string]json.RawMessage, order *[]string, key string, raw json.RawMessage) {
	if *target == nil {
		*target = make(map[string]json.RawMessage)
	}
	if !slices.Contains(*order, key) {
		*order = append(*order, key)
	}
	(*target)[key] = bytes.Clone(raw)
}

// Apply runs options in order on owned copies; a nil option fails before HTTP.
func Apply[T any, O ~func(*T) error](ctx context.Context, guard func(context.Context) error, value T, copyValue func(T) T, options []O) (T, error) {
	value = copyValue(value)
	for _, apply := range options {
		if err := guard(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, Invalid("nil option")
		}
		next := copyValue(value)
		err := apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return value, err
		}
		value = copyValue(next)
	}
	return value, guard(ctx)
}

func GetSnapshot(v GetOpts) func(*GetOpts) error {
	owned := CopyGet(v)
	return func(c *GetOpts) error { *c = CopyGet(owned); return nil }
}
func GetFields(values ...string) func(*GetOpts) error {
	owned := slices.Clone(values)
	return func(c *GetOpts) error { c.Fields = slices.Clone(owned); return nil }
}
func GetHeaders(values map[string]string) func(*GetOpts) error {
	owned := maps.Clone(values)
	return func(c *GetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func GetMicroversion(v string) func(*GetOpts) error {
	return func(c *GetOpts) error { x := v; c.Microversion = &x; return nil }
}

func WriteSnapshot(v WriteOpts) func(*WriteOpts) error {
	owned := CopyWrite(v)
	return func(c *WriteOpts) error { *c = CopyWrite(owned); return nil }
}

// WriteField records one Python attribute; later values replace earlier ones.
func WriteField(key string, value any) func(*WriteOpts) error {
	raw, err := marshal(value)
	return func(c *WriteOpts) error {
		if err != nil {
			return errors.Join(Invalid("attribute %q is not JSON", key), err)
		}
		setOrdered(&c.Attrs, &c.order, key, raw)
		return nil
	}
}
func WriteHeaders(values map[string]string) func(*WriteOpts) error {
	owned := maps.Clone(values)
	return func(c *WriteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WriteMicroversion(v string) func(*WriteOpts) error {
	return func(c *WriteOpts) error { x := v; c.Microversion = &x; return nil }
}

func DeleteSnapshot(v DeleteOpts) func(*DeleteOpts) error {
	owned := CopyDelete(v)
	return func(c *DeleteOpts) error { *c = CopyDelete(owned); return nil }
}
func DeleteIgnoreMissing(v bool) func(*DeleteOpts) error {
	return func(c *DeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func DeleteHeaders(values map[string]string) func(*DeleteOpts) error {
	owned := maps.Clone(values)
	return func(c *DeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func DeleteMicroversion(v string) func(*DeleteOpts) error {
	return func(c *DeleteOpts) error { x := v; c.Microversion = &x; return nil }
}

func FindSnapshot(v FindOpts) func(*FindOpts) error {
	owned := CopyFind(v)
	return func(c *FindOpts) error { *c = CopyFind(owned); return nil }
}
func FindIgnoreMissing(v bool) func(*FindOpts) error {
	return func(c *FindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// FindQuery records one Python **query value for find.
func FindQuery(key string, value any) func(*FindOpts) error {
	raw, err := marshal(value)
	return func(c *FindOpts) error {
		if err != nil {
			return errors.Join(Invalid("query %q is not JSON", key), err)
		}
		setOrdered(&c.Query, &c.order, key, raw)
		return nil
	}
}
func FindHeaders(values map[string]string) func(*FindOpts) error {
	owned := maps.Clone(values)
	return func(c *FindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func FindMicroversion(v string) func(*FindOpts) error {
	return func(c *FindOpts) error { x := v; c.Microversion = &x; return nil }
}

func ListSnapshot(v ListOpts) func(*ListOpts) error {
	owned := CopyList(v)
	return func(c *ListOpts) error { *c = CopyList(owned); return nil }
}

// ListQuery records one Python **query value. Names that are neither server
// query names, Body names, list controls nor bound proxy arguments are
// discarded before marshaling, as Resource.list discards them.
func (s *Spec) ListQuery(key string, value any) func(*ListOpts) error {
	if s.listIgnores(key) {
		return func(*ListOpts) error { return nil }
	}
	raw, err := marshal(value)
	return func(c *ListOpts) error {
		if err != nil {
			return errors.Join(Invalid("query %q is not JSON", key), err)
		}
		setOrdered(&c.Query, &c.order, key, raw)
		return nil
	}
}
func ListHeaders(values map[string]string) func(*ListOpts) error {
	owned := maps.Clone(values)
	return func(c *ListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func ListMicroversion(v string) func(*ListOpts) error {
	return func(c *ListOpts) error { x := v; c.Microversion = &x; return nil }
}

func (s *Spec) listIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "cls", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	if slices.Contains(s.ParentReserved(), key) {
		return false
	}
	for _, pair := range s.Profile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range s.Profile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

// ParentReserved names the bound parent proxy argument and URI attribute.
func (s *Spec) ParentReserved() []string {
	if s.Parent == "" {
		return nil
	}
	return []string{s.Parent, s.ParentArg}
}
