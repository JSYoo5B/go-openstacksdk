package rules

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// DSCPMarkingFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type DSCPMarkingFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

// DSCPMarkingFindOption is one option of an operation over
// DSCPMarkingFindOpts: a With* function of this package, request.Custom over
// *DSCPMarkingFindOpts, or a shared request.FindCall value. A nil option fails
// with resource.ErrInvalidOption before HTTP.
type DSCPMarkingFindOption = request.FindCall

func dscpMarkingFindOptions(v DSCPMarkingFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingFind(v DSCPMarkingFindOpts) DSCPMarkingFindOpts {
	owned := neutronfind.CloneOptions(dscpMarkingFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithDSCPMarkingFindOpts(v DSCPMarkingFindOpts) DSCPMarkingFindOption {
	owned := copyDSCPMarkingFind(v)
	return request.Custom(func(c *DSCPMarkingFindOpts) error { *c = copyDSCPMarkingFind(owned); return nil })
}

// dscpMarkingFindQuery applies one marshaled request.Query value.
func dscpMarkingFindQuery(key string, owned json.RawMessage, err error) func(*DSCPMarkingFindOpts) error {
	return func(c *DSCPMarkingFindOpts) error {
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
}

// DSCPMarkingListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type DSCPMarkingListOpts struct {
	BasePath     *string
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

// DSCPMarkingListOption is one option of an operation over
// DSCPMarkingListOpts: a With* function of this package, request.Custom over
// *DSCPMarkingListOpts, or a shared request.ListBasePathCall value. A nil option fails
// with resource.ErrInvalidOption before HTTP.
type DSCPMarkingListOption = request.ListBasePathCall

func dscpMarkingListOptions(v DSCPMarkingListOpts) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingList(v DSCPMarkingListOpts) DSCPMarkingListOpts {
	owned := neutronfind.CloneOptions(dscpMarkingListOptions(v))
	v.BasePath = owned.BasePath
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithDSCPMarkingListOpts(v DSCPMarkingListOpts) DSCPMarkingListOption {
	owned := copyDSCPMarkingList(v)
	return request.Custom(func(c *DSCPMarkingListOpts) error { *c = copyDSCPMarkingList(owned); return nil })
}
func dscpMarkingListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "qos_policy", "qos_policy_id", "cls", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range dscpMarkingRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range dscpMarkingRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

// dscpMarkingListQuery applies one marshaled request.Query value.
func dscpMarkingListQuery(key string, owned json.RawMessage, err error) func(*DSCPMarkingListOpts) error {
	if dscpMarkingListIgnores(key) {
		return func(*DSCPMarkingListOpts) error { return nil }
	}
	return func(c *DSCPMarkingListOpts) error {
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
}

// DSCPMarkingGetOpts owns optional member transport extensions. Query does
// not seed response Body; the pinned named getter is query-free by default.
type DSCPMarkingGetOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

// DSCPMarkingGetOption is one option of an operation over DSCPMarkingGetOpts:
// a With* function of this package, request.Custom over *DSCPMarkingGetOpts,
// or a shared request.GetQueryCall value. A nil option fails with
// resource.ErrInvalidOption before HTTP.
type DSCPMarkingGetOption = request.GetQueryCall

func dscpMarkingGetOptions(v DSCPMarkingGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyDSCPMarkingGet(v DSCPMarkingGetOpts) DSCPMarkingGetOpts {
	owned := neutronfind.CloneOptions(dscpMarkingGetOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithDSCPMarkingGetOpts(v DSCPMarkingGetOpts) DSCPMarkingGetOption {
	owned := copyDSCPMarkingGet(v)
	return request.Custom(func(c *DSCPMarkingGetOpts) error { *c = copyDSCPMarkingGet(owned); return nil })
}

// dscpMarkingGetQuery applies one marshaled request.Query member value.
func dscpMarkingGetQuery(key string, owned json.RawMessage, err error) func(*DSCPMarkingGetOpts) error {
	return func(c *DSCPMarkingGetOpts) error {
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
}

// dscpMarkingFindCalls translates the options of one call into DSCPMarkingFindOpts callbacks.
func dscpMarkingFindCalls(calls []request.FindCall) []func(*DSCPMarkingFindOpts) error {
	return qosCalls(calls, func(s *DSCPMarkingFindOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, dscpMarkingFindQuery, false)
}

// dscpMarkingListCalls translates the options of one call into DSCPMarkingListOpts callbacks.
func dscpMarkingListCalls(calls []request.ListBasePathCall) []func(*DSCPMarkingListOpts) error {
	return qosCalls(calls, func(s *DSCPMarkingListOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, basePath: &s.BasePath}
	}, dscpMarkingListQuery, true)
}

// dscpMarkingGetCalls translates the options of one call into DSCPMarkingGetOpts callbacks.
func dscpMarkingGetCalls(calls []request.GetQueryCall) []func(*DSCPMarkingGetOpts) error {
	return qosCalls(calls, func(s *DSCPMarkingGetOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, dscpMarkingGetQuery, false)
}
