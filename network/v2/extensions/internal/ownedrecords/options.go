package ownedrecords

import (
	"encoding/json"
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

func GetSnapshot(v GetOpts) func(*GetOpts) error {
	owned := CopyGet(v)
	return func(c *GetOpts) error { *c = CopyGet(owned); return nil }
}
func GetFields(values ...string) func(*GetOpts) error {
	owned := slices.Clone(values)
	return func(c *GetOpts) error { c.Fields = slices.Clone(owned); return nil }
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

func FindSnapshot(v FindOpts) func(*FindOpts) error {
	owned := CopyFind(v)
	return func(c *FindOpts) error { *c = CopyFind(owned); return nil }
}

func WriteSnapshot(v WriteOpts) func(*WriteOpts) error {
	owned := CopyWrite(v)
	return func(c *WriteOpts) error { *c = CopyWrite(owned); return nil }
}

func DeleteSnapshot(v DeleteOpts) func(*DeleteOpts) error {
	owned := CopyDelete(v)
	return func(c *DeleteOpts) error { *c = CopyDelete(owned); return nil }
}

func ActionSnapshot(v ActionOpts) func(*ActionOpts) error {
	owned := CopyAction(v)
	return func(c *ActionOpts) error { *c = CopyAction(owned); return nil }
}
