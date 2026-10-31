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

// BandwidthLimitFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type BandwidthLimitFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

// BandwidthLimitFindOption is one option of an operation over
// BandwidthLimitFindOpts: a With* function of this package, request.Custom
// over *BandwidthLimitFindOpts, or a shared request.FindCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type BandwidthLimitFindOption = request.FindCall

func bandwidthLimitFindOptions(v BandwidthLimitFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyBandwidthLimitFind(v BandwidthLimitFindOpts) BandwidthLimitFindOpts {
	owned := neutronfind.CloneOptions(bandwidthLimitFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithBandwidthLimitFindOpts(v BandwidthLimitFindOpts) BandwidthLimitFindOption {
	owned := copyBandwidthLimitFind(v)
	return request.Custom(func(c *BandwidthLimitFindOpts) error { *c = copyBandwidthLimitFind(owned); return nil })
}

// bandwidthLimitFindQuery applies one marshaled request.Query value.
func bandwidthLimitFindQuery(key string, owned json.RawMessage, err error) func(*BandwidthLimitFindOpts) error {
	return func(c *BandwidthLimitFindOpts) error {
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

// BandwidthLimitListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type BandwidthLimitListOpts struct {
	BasePath     *string
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

// BandwidthLimitListOption is one option of an operation over
// BandwidthLimitListOpts: a With* function of this package, request.Custom
// over *BandwidthLimitListOpts, or a shared request.ListBasePathCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type BandwidthLimitListOption = request.ListBasePathCall

func bandwidthLimitListOptions(v BandwidthLimitListOpts) neutronfind.Options {
	return neutronfind.Options{BasePath: v.BasePath, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyBandwidthLimitList(v BandwidthLimitListOpts) BandwidthLimitListOpts {
	owned := neutronfind.CloneOptions(bandwidthLimitListOptions(v))
	v.BasePath, v.Microversion, v.Query, v.Headers, v.order = owned.BasePath, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithBandwidthLimitListOpts(v BandwidthLimitListOpts) BandwidthLimitListOption {
	owned := copyBandwidthLimitList(v)
	return request.Custom(func(c *BandwidthLimitListOpts) error { *c = copyBandwidthLimitList(owned); return nil })
}
func bandwidthLimitListIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "qos_policy_id", "qos_policy", "qos_rule", "qos_rule_id", "ignore_missing", "name_or_id", "cls", "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range bandwidthLimitRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range bandwidthLimitRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

// bandwidthLimitListQuery applies one marshaled request.Query value.
func bandwidthLimitListQuery(key string, owned json.RawMessage, err error) func(*BandwidthLimitListOpts) error {
	if bandwidthLimitListIgnores(key) {
		return func(*BandwidthLimitListOpts) error { return nil }
	}
	return func(c *BandwidthLimitListOpts) error {
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

// BandwidthLimitGetOpts adds operation-owned safe headers and microversion.
// The pinned proxy exposes no member query kwargs; no Find options are reused.
type BandwidthLimitGetOpts struct {
	Microversion *string
	Headers      map[string]string
}

// BandwidthLimitGetOption is one option of an operation over
// BandwidthLimitGetOpts: a With* function of this package, request.Custom over
// *BandwidthLimitGetOpts, or a shared request.GetCall value. A nil option
// fails with resource.ErrInvalidOption before HTTP.
type BandwidthLimitGetOption = request.GetCall

func bandwidthLimitGetOptions(v BandwidthLimitGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyBandwidthLimitGet(v BandwidthLimitGetOpts) BandwidthLimitGetOpts {
	owned := neutronfind.CloneOptions(bandwidthLimitGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	return v
}
func WithBandwidthLimitGetOpts(v BandwidthLimitGetOpts) BandwidthLimitGetOption {
	owned := copyBandwidthLimitGet(v)
	return request.Custom(func(c *BandwidthLimitGetOpts) error { *c = copyBandwidthLimitGet(owned); return nil })
}

// bandwidthLimitFindCalls translates the options of one call into BandwidthLimitFindOpts callbacks.
func bandwidthLimitFindCalls(calls []request.FindCall) []func(*BandwidthLimitFindOpts) error {
	return qosCalls(calls, func(s *BandwidthLimitFindOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, bandwidthLimitFindQuery, false)
}

// bandwidthLimitListCalls translates the options of one call into BandwidthLimitListOpts callbacks.
func bandwidthLimitListCalls(calls []request.ListBasePathCall) []func(*BandwidthLimitListOpts) error {
	return qosCalls(calls, func(s *BandwidthLimitListOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, basePath: &s.BasePath}
	}, bandwidthLimitListQuery, true)
}

// bandwidthLimitGetCalls translates the options of one call into BandwidthLimitGetOpts callbacks.
func bandwidthLimitGetCalls(calls []request.GetCall) []func(*BandwidthLimitGetOpts) error {
	return qosCalls(calls, func(s *BandwidthLimitGetOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, nil, false)
}
