package addressscopes

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
)

// RecordReadOpts owns literal-ID fetch options. Seed is an immutable constructor
// Body snapshot; Fields uses Python canonical names and is translated for GET.
type RecordReadOpts struct {
	Fields       []string
	Seed         map[string]json.RawMessage
	Microversion *string
	Headers      map[string]string
}
type RecordReadOption func(*RecordReadOpts) error

// RecordListOpts owns Source kwargs; Query retains null, booleans and arrays.
// Omitted paginated means true; max_items counts raw rows before local filters.
type RecordListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	queryOrder   []string
}
type RecordListOption func(*RecordListOpts) error

type RecordFindOpts struct {
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	IgnoreMissing *bool
	queryOrder    []string
}
type RecordFindOption func(*RecordFindOpts) error

func scopeRecordOrder(query map[string]json.RawMessage, order []string) []string {
	owned := make([]string, 0, len(query))
	for _, key := range order {
		if _, exists := query[key]; exists && !slices.Contains(owned, key) {
			owned = append(owned, key)
		}
	}
	missing := make([]string, 0)
	for key := range query {
		if !slices.Contains(owned, key) {
			missing = append(missing, key)
		}
	}
	slices.Sort(missing)
	return append(owned, missing...)
}
func scopeRecordVersion(value *string) *string {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func copyScopeRead(value RecordReadOpts) RecordReadOpts {
	value.Fields = slices.Clone(value.Fields)
	value.Seed = scopeRecordFields(value.Seed)
	value.Headers = maps.Clone(value.Headers)
	value.Microversion = scopeRecordVersion(value.Microversion)
	return value
}
func copyScopeList(value RecordListOpts) RecordListOpts {
	value.Query = scopeRecordFields(value.Query)
	value.queryOrder = scopeRecordOrder(value.Query, value.queryOrder)
	value.Headers = maps.Clone(value.Headers)
	value.Microversion = scopeRecordVersion(value.Microversion)
	return value
}
func copyScopeFind(value RecordFindOpts) RecordFindOpts {
	value.Query = scopeRecordFields(value.Query)
	value.queryOrder = scopeRecordOrder(value.Query, value.queryOrder)
	value.Headers = maps.Clone(value.Headers)
	value.Microversion = scopeRecordVersion(value.Microversion)
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	return value
}
func WithRecordReadOpts(value RecordReadOpts) RecordReadOption {
	owned := copyScopeRead(value)
	return func(c *RecordReadOpts) error { *c = copyScopeRead(owned); return nil }
}
func WithRecordReadFields(fields ...string) RecordReadOption {
	owned := slices.Clone(fields)
	return func(c *RecordReadOpts) error { c.Fields = slices.Clone(owned); return nil }
}
func WithRecordReadSeed(value map[string]json.RawMessage) RecordReadOption {
	owned := scopeRecordFields(value)
	return func(c *RecordReadOpts) error { c.Seed = scopeRecordFields(owned); return nil }
}
func WithRecordReadHeaders(value map[string]string) RecordReadOption {
	owned := maps.Clone(value)
	return func(c *RecordReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordReadMicroversion(value string) RecordReadOption {
	return func(c *RecordReadOpts) error { owned := value; c.Microversion = &owned; return nil }
}
func WithRecordListOpts(value RecordListOpts) RecordListOption {
	owned := copyScopeList(value)
	return func(c *RecordListOpts) error { *c = copyScopeList(owned); return nil }
}
func WithRecordListQuery(key string, value any) RecordListOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordListOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func WithRecordListHeaders(value map[string]string) RecordListOption {
	owned := maps.Clone(value)
	return func(c *RecordListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordListMicroversion(value string) RecordListOption {
	return func(c *RecordListOpts) error { owned := value; c.Microversion = &owned; return nil }
}
func WithRecordFindOpts(value RecordFindOpts) RecordFindOption {
	owned := copyScopeFind(value)
	return func(c *RecordFindOpts) error { *c = copyScopeFind(owned); return nil }
}
func WithRecordFindQuery(key string, value any) RecordFindOption {
	raw, err := json.Marshal(value)
	owned := bytes.Clone(raw)
	return func(c *RecordFindOpts) error {
		if err != nil {
			return err
		}
		if c.Query == nil {
			c.Query = make(map[string]json.RawMessage)
		}
		c.Query[key] = bytes.Clone(owned)
		if !slices.Contains(c.queryOrder, key) {
			c.queryOrder = append(c.queryOrder, key)
		}
		return nil
	}
}
func WithRecordFindHeaders(value map[string]string) RecordFindOption {
	owned := maps.Clone(value)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordFindMicroversion(value string) RecordFindOption {
	return func(c *RecordFindOpts) error { owned := value; c.Microversion = &owned; return nil }
}
func WithRecordFindIgnoreMissing(value bool) RecordFindOption {
	return func(c *RecordFindOpts) error { owned := value; c.IgnoreMissing = &owned; return nil }
}
