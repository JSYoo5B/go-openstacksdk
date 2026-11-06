package ruletypes

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordFindOpts has no Query: pinned find_qos_rule_type does not accept
// arbitrary kwargs. Omitted IgnoreMissing is true. Transport options are owned
// Go extensions; Microversion overrides only the member GET phase.
type RecordFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Headers       map[string]string
}

// RecordFindOption is one option of FindRecord: WithRecordFindOpts,
// request.IgnoreMissing, request.Microversion (member GET phase only),
// request.Headers (replaces the headers) or request.Custom over
// *RecordFindOpts. Other shared calls, including request.Query, fail with
// resource.ErrInvalidOption before HTTP.
type RecordFindOption = request.FindCall

var recordFindCalls = calls.Spec[RecordFindOpts]{
	IgnoreMissing: calls.Pointer(func(c *RecordFindOpts) **bool { return &c.IgnoreMissing }),
	Microversion:  calls.Pointer(func(c *RecordFindOpts) **string { return &c.Microversion }),
	Headers:       calls.Value(func(c *RecordFindOpts) *map[string]string { return &c.Headers }),
}

func ruleTypeFindOptions(v RecordFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Headers: v.Headers}
}
func copyRuleTypeFind(v RecordFindOpts) RecordFindOpts {
	owned := neutronfind.CloneOptions(ruleTypeFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Headers = owned.IgnoreMissing, owned.Microversion, owned.Headers
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyRuleTypeFind(v)
	return request.Custom(func(c *RecordFindOpts) error { *c = copyRuleTypeFind(owned); return nil })
}

// RecordListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type RecordListOpts struct {
	BasePath     *string
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

// RecordListOption is one option of ListRecords and AllRecords:
// WithRecordListOpts, request.Query (one Source kwarg as JSON; unknown names
// are discarded with their marshal error), request.Limit, request.Marker,
// request.MaxItems, request.Paginated (the limit, marker, max_items and
// paginated kwargs), request.BasePath (a safe service-relative list route;
// the raw request.Query("base_path", ...) lane keeps its rejection),
// request.Microversion, request.Headers (replaces the headers) or
// request.Custom over *RecordListOpts. request.Header fails with
// resource.ErrInvalidOption before HTTP.
type RecordListOption = request.ListBasePathCall

func ruleTypeListOptions(v RecordListOpts) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyRuleTypeList(v RecordListOpts) RecordListOpts {
	owned := neutronfind.CloneOptions(ruleTypeListOptions(v))
	v.BasePath, v.Microversion, v.Query, v.Headers, v.order = owned.BasePath, owned.Microversion, owned.Query, owned.Headers, owned.Order
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	return v
}
func WithRecordListOpts(v RecordListOpts) RecordListOption {
	owned := copyRuleTypeList(v)
	return request.Custom(func(c *RecordListOpts) error { *c = copyRuleTypeList(owned); return nil })
}
func ruleTypeListIgnores(key string) bool {
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
	for _, pair := range ruleTypeRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range ruleTypeRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

// ruleTypeListKeyword records one list kwarg. Source discards unknown kwargs
// before evaluating their transport value, so their marshal error is dropped.
func ruleTypeListKeyword(c *RecordListOpts, key string, owned json.RawMessage, err error) error {
	if ruleTypeListIgnores(key) {
		return nil
	}
	if err != nil {
		return err
	}
	if c.Query == nil {
		c.Query = make(map[string]json.RawMessage)
	}
	if !slices.Contains(c.order, key) {
		c.order = append(c.order, key)
	}
	c.Query[key] = bytes.Clone(owned)
	return nil
}

var recordListCalls = calls.Spec[RecordListOpts]{
	Query:        calls.JSONQuery(ruleTypeListKeyword),
	BasePath:     calls.Pointer(func(c *RecordListOpts) **string { return &c.BasePath }),
	Microversion: calls.Pointer(func(c *RecordListOpts) **string { return &c.Microversion }),
	Headers:      calls.Value(func(c *RecordListOpts) *map[string]string { return &c.Headers }),
	Paging:       calls.JSONPaging(ruleTypeListKeyword),
}

// RecordGetOpts is the pinned get_qos_rule_type fields selector plus explicit
// owned transport settings. It does not accept constructor/query kwargs.
type RecordGetOpts struct {
	Fields       []string
	Microversion *string
	Headers      map[string]string
}

// RecordGetOption is one option of GetRecord: WithRecordGetOpts,
// WithRecordGetFields, request.Microversion, request.Headers (replaces the
// headers) or request.Custom over *RecordGetOpts. request.Header fails with
// resource.ErrInvalidOption before HTTP.
type RecordGetOption = request.GetCall

var recordGetCalls = calls.Spec[RecordGetOpts]{
	Microversion: calls.Pointer(func(c *RecordGetOpts) **string { return &c.Microversion }),
	Headers:      calls.Value(func(c *RecordGetOpts) *map[string]string { return &c.Headers }),
}

func copyRuleTypeGet(v RecordGetOpts) RecordGetOpts {
	v.Fields = slices.Clone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Headers == nil {
		v.Headers = make(map[string]string)
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyRuleTypeGet(v)
	return request.Custom(func(c *RecordGetOpts) error { *c = copyRuleTypeGet(owned); return nil })
}
func WithRecordGetFields(values ...string) RecordGetOption {
	owned := slices.Clone(values)
	return request.Custom(func(c *RecordGetOpts) error { c.Fields = slices.Clone(owned); return nil })
}
func ruleTypeGetOptions(v RecordGetOpts) (neutronfind.Options, error) {
	prepared := neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
	if len(v.Fields) == 0 {
		return prepared, nil
	}
	fields := slices.Clone(v.Fields)
	for i, name := range fields {
		if !utf8.ValidString(name) {
			return prepared, ruleTypeRecordInvalid("fields must be UTF-8")
		}
		for _, field := range ruleTypeRecordProfile.Body {
			if name == field.Name {
				fields[i] = field.Wire
				break
			}
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return prepared, err
	}
	prepared.Query = map[string]json.RawMessage{"fields": raw}
	prepared.Order = []string{"fields"}
	return prepared, nil
}
