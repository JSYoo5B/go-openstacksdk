package ownedrecords

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
)

// GetOpts is the get_* fields selector plus Go transport settings.
type GetOpts struct {
	Fields       []string
	Microversion *string
	Headers      map[string]string
}

// ListOpts owns Python list **query as JSON. Declared server query names and
// Body filters are kept; max_items and paginated control paging.
type ListOpts struct {
	Query        map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
	order        []string
}

// FindOpts owns Python find **query as JSON. Nil IgnoreMissing means true.
type FindOpts struct {
	IgnoreMissing *bool
	Query         map[string]json.RawMessage
	Microversion  *string
	Headers       map[string]string
	order         []string
}

// WriteOpts owns Python create/update **attrs as JSON.
type WriteOpts struct {
	Attrs        map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
	order        []string
}

// DeleteOpts selects ignore_missing. Nil IgnoreMissing means true.
type DeleteOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Headers       map[string]string
}

// ActionOpts carries Go transport settings for resource actions.
type ActionOpts struct {
	Microversion *string
	Headers      map[string]string
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
func listPolicy(v ListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func CopyList(v ListOpts) ListOpts {
	owned := neutronfind.CloneOptions(listPolicy(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func findPolicy(v FindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func CopyFind(v FindOpts) FindOpts {
	owned := neutronfind.CloneOptions(findPolicy(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func writePolicy(v WriteOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Attrs, Headers: v.Headers, Order: v.order}
}
func CopyWrite(v WriteOpts) WriteOpts {
	owned := neutronfind.CloneOptions(writePolicy(v))
	v.Microversion, v.Attrs, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func CopyDelete(v DeleteOpts) DeleteOpts {
	owned := neutronfind.CloneOptions(neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Headers: v.Headers})
	v.IgnoreMissing, v.Microversion, v.Headers = owned.IgnoreMissing, owned.Microversion, owned.Headers
	return v
}
func CopyAction(v ActionOpts) ActionOpts {
	v.Headers, v.Microversion = maps.Clone(v.Headers), copyVersion(v.Microversion)
	return v
}

// setRaw snapshots value at option construction; a marshal error surfaces
// when the option is applied, before HTTP.
func setRaw(label, key string, value any) func(*map[string]json.RawMessage, *[]string) error {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(target *map[string]json.RawMessage, order *[]string) error {
		if err != nil {
			return errors.Join(Invalid("%s %q is not JSON", label, key), err)
		}
		if *target == nil {
			*target = make(map[string]json.RawMessage)
		}
		if !slices.Contains(*order, key) {
			*order = append(*order, key)
		}
		(*target)[key] = bytes.Clone(owned)
		return nil
	}
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

func ListSnapshot(v ListOpts) func(*ListOpts) error {
	owned := CopyList(v)
	return func(c *ListOpts) error { *c = CopyList(owned); return nil }
}

// listIgnores reports a list kwarg that Resource.list discards before its
// value is used: neither a declared query name, a Body attribute nor a
// paging control. Reserved constructor controls are kept so they fail.
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

// ListQuery records one Python **query name. Unknown names are ignored before
// marshaling, as Resource.list drops them.
func (s *Spec) ListQuery(key string, value any) func(*ListOpts) error {
	if s.listIgnores(key) {
		return func(*ListOpts) error { return nil }
	}
	set := setRaw("list query", key, value)
	return func(c *ListOpts) error { return set(&c.Query, &c.order) }
}
func ListHeaders(values map[string]string) func(*ListOpts) error {
	owned := maps.Clone(values)
	return func(c *ListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func ListMicroversion(v string) func(*ListOpts) error {
	return func(c *ListOpts) error { x := v; c.Microversion = &x; return nil }
}

func FindSnapshot(v FindOpts) func(*FindOpts) error {
	owned := CopyFind(v)
	return func(c *FindOpts) error { *c = CopyFind(owned); return nil }
}
func FindQuery(key string, value any) func(*FindOpts) error {
	set := setRaw("find query", key, value)
	return func(c *FindOpts) error { return set(&c.Query, &c.order) }
}
func FindIgnoreMissing(v bool) func(*FindOpts) error {
	return func(c *FindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func FindHeaders(values map[string]string) func(*FindOpts) error {
	owned := maps.Clone(values)
	return func(c *FindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func FindMicroversion(v string) func(*FindOpts) error {
	return func(c *FindOpts) error { x := v; c.Microversion = &x; return nil }
}

func WriteSnapshot(v WriteOpts) func(*WriteOpts) error {
	owned := CopyWrite(v)
	return func(c *WriteOpts) error { *c = CopyWrite(owned); return nil }
}
func WriteField(key string, value any) func(*WriteOpts) error {
	set := setRaw("attribute", key, value)
	return func(c *WriteOpts) error { return set(&c.Attrs, &c.order) }
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

func ActionSnapshot(v ActionOpts) func(*ActionOpts) error {
	owned := CopyAction(v)
	return func(c *ActionOpts) error { *c = CopyAction(owned); return nil }
}
func ActionHeaders(values map[string]string) func(*ActionOpts) error {
	owned := maps.Clone(values)
	return func(c *ActionOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func ActionMicroversion(v string) func(*ActionOpts) error {
	return func(c *ActionOpts) error { x := v; c.Microversion = &x; return nil }
}
