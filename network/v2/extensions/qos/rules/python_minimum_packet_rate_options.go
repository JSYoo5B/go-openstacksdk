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

// MinimumPacketRateFindOpts owns Source kwargs as JSON. Omitted IgnoreMissing is true.
// Query is member query/constructor input; fallback maps aliases separately.
type MinimumPacketRateFindOpts struct {
	IgnoreMissing *bool
	Microversion  *string
	Query         map[string]json.RawMessage
	Headers       map[string]string
	order         []string
}

// MinimumPacketRateFindOption is one option of an operation over
// MinimumPacketRateFindOpts: a With* function of this package, request.Custom
// over *MinimumPacketRateFindOpts, or a shared request.FindCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type MinimumPacketRateFindOption = request.FindCall

func minimumPacketRateFindOptions(v MinimumPacketRateFindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumPacketRateFind(v MinimumPacketRateFindOpts) MinimumPacketRateFindOpts {
	owned := neutronfind.CloneOptions(minimumPacketRateFindOptions(v))
	v.IgnoreMissing, v.Microversion, v.Query, v.Headers, v.order = owned.IgnoreMissing, owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithMinimumPacketRateFindOpts(v MinimumPacketRateFindOpts) MinimumPacketRateFindOption {
	owned := copyMinimumPacketRateFind(v)
	return request.Custom(func(c *MinimumPacketRateFindOpts) error { *c = copyMinimumPacketRateFind(owned); return nil })
}

// minimumPacketRateFindQuery applies one marshaled request.Query value.
func minimumPacketRateFindQuery(key string, owned json.RawMessage, err error) func(*MinimumPacketRateFindOpts) error {
	return func(c *MinimumPacketRateFindOpts) error {
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

// MinimumPacketRateListOpts owns list kwargs. Query includes declared server fields,
// nonquery Body filters and paginated/max_items controls. Unknowns are ignored.
type MinimumPacketRateListOpts struct {
	Microversion *string
	Query        map[string]json.RawMessage
	Headers      map[string]string
	order        []string
}

// MinimumPacketRateListOption is one option of an operation over
// MinimumPacketRateListOpts: a With* function of this package, request.Custom
// over *MinimumPacketRateListOpts, or a shared request.ListCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type MinimumPacketRateListOption = request.ListCall

func minimumPacketRateListOptions(v MinimumPacketRateListOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Query: v.Query, Headers: v.Headers, Order: v.order}
}
func copyMinimumPacketRateList(v MinimumPacketRateListOpts) MinimumPacketRateListOpts {
	owned := neutronfind.CloneOptions(minimumPacketRateListOptions(v))
	v.Microversion, v.Query, v.Headers, v.order = owned.Microversion, owned.Query, owned.Headers, owned.Order
	return v
}
func WithMinimumPacketRateListOpts(v MinimumPacketRateListOpts) MinimumPacketRateListOption {
	owned := copyMinimumPacketRateList(v)
	return request.Custom(func(c *MinimumPacketRateListOpts) error { *c = copyMinimumPacketRateList(owned); return nil })
}
func minimumPacketRateListIgnores(key string) bool {
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
	for _, pair := range minimumPacketRateRecordProfile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range minimumPacketRateRecordProfile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

// minimumPacketRateListQuery applies one marshaled request.Query value.
func minimumPacketRateListQuery(key string, owned json.RawMessage, err error) func(*MinimumPacketRateListOpts) error {
	if minimumPacketRateListIgnores(key) {
		return func(*MinimumPacketRateListOpts) error { return nil }
	}
	return func(c *MinimumPacketRateListOpts) error {
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

// MinimumPacketRateGetOpts adds operation-owned safe headers and microversion.
// The pinned proxy exposes no member query kwargs; no Find options are reused.
type MinimumPacketRateGetOpts struct {
	Microversion *string
	Headers      map[string]string
}

// MinimumPacketRateGetOption is one option of an operation over
// MinimumPacketRateGetOpts: a With* function of this package, request.Custom
// over *MinimumPacketRateGetOpts, or a shared request.GetCall value. A nil
// option fails with resource.ErrInvalidOption before HTTP.
type MinimumPacketRateGetOption = request.GetCall

func minimumPacketRateGetOptions(v MinimumPacketRateGetOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
}
func copyMinimumPacketRateGet(v MinimumPacketRateGetOpts) MinimumPacketRateGetOpts {
	owned := neutronfind.CloneOptions(minimumPacketRateGetOptions(v))
	v.Microversion, v.Headers = owned.Microversion, owned.Headers
	return v
}
func WithMinimumPacketRateGetOpts(v MinimumPacketRateGetOpts) MinimumPacketRateGetOption {
	owned := copyMinimumPacketRateGet(v)
	return request.Custom(func(c *MinimumPacketRateGetOpts) error { *c = copyMinimumPacketRateGet(owned); return nil })
}

// minimumPacketRateFindCalls translates the options of one call into MinimumPacketRateFindOpts callbacks.
func minimumPacketRateFindCalls(calls []request.FindCall) []func(*MinimumPacketRateFindOpts) error {
	return qosCalls(calls, func(s *MinimumPacketRateFindOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion, ignore: &s.IgnoreMissing}
	}, minimumPacketRateFindQuery, false)
}

// minimumPacketRateListCalls translates the options of one call into MinimumPacketRateListOpts callbacks.
func minimumPacketRateListCalls(calls []request.ListCall) []func(*MinimumPacketRateListOpts) error {
	return qosCalls(calls, func(s *MinimumPacketRateListOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, minimumPacketRateListQuery, true)
}

// minimumPacketRateGetCalls translates the options of one call into MinimumPacketRateGetOpts callbacks.
func minimumPacketRateGetCalls(calls []request.GetCall) []func(*MinimumPacketRateGetOpts) error {
	return qosCalls(calls, func(s *MinimumPacketRateGetOpts) qosCallFields {
		return qosCallFields{headers: &s.Headers, microversion: &s.Microversion}
	}, nil, false)
}
