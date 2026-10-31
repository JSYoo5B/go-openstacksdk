// Package shareattachments provides Nova server share mappings with SDK-owned
// options, response projection and transport policy.
package shareattachments

import (
	"bytes"
	"encoding/json"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type ReadOpts struct{ Microversion *string }

// ReadOption is one option of GetRecord:
// request.Microversion, request.Header or request.Custom over
// *request.Config[ReadOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ReadOption = request.GetCall
type CreateOpts struct {
	Attributes   map[string]json.RawMessage
	Microversion *string
}

// CreateOption is one option of CreateRecord: WithCreateOptions,
// WithCreateAttribute, request.Microversion, request.Header or request.Custom
// over *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall
type DeleteOpts struct {
	IgnoreMissing *bool
	Microversion  *string
}

// DeleteOption is one option of DeleteRecord:
// request.IgnoreMissing, request.Microversion, request.Header or
// request.Custom over *request.Config[DeleteOpts]. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall
type ListOpts struct {
	Limit        int
	Marker       string
	MaxItems     int
	Paginated    *bool
	Microversion *string
	Filters      []resource.ListOption
}

// ListOption is one option of ListRecords: WithListOptions,
// WithListFilter, WithListFilters, request.Limit, request.Marker,
// request.MaxItems, request.Paginated, request.Microversion, request.Header
// or request.Custom over *request.Config[ListOpts]. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var (
	readCalls = calls.Spec[request.Config[ReadOpts]]{
		Header:       calls.ConfigHeader[ReadOpts],
		Microversion: calls.Pointer(func(c *request.Config[ReadOpts]) **string { return &c.Options.Microversion }),
	}
	createCalls = calls.Spec[request.Config[CreateOpts]]{
		Header:       calls.ConfigHeader[CreateOpts],
		Microversion: calls.Pointer(func(c *request.Config[CreateOpts]) **string { return &c.Options.Microversion }),
	}
	deleteCalls = calls.Spec[request.Config[DeleteOpts]]{
		Header:        calls.ConfigHeader[DeleteOpts],
		Microversion:  calls.Pointer(func(c *request.Config[DeleteOpts]) **string { return &c.Options.Microversion }),
		IgnoreMissing: calls.Pointer(func(c *request.Config[DeleteOpts]) **bool { return &c.Options.IgnoreMissing }),
	}
	listCalls = calls.Spec[request.Config[ListOpts]]{
		Header:       calls.ConfigHeader[ListOpts],
		Microversion: calls.Pointer(func(c *request.Config[ListOpts]) **string { return &c.Options.Microversion }),
		Paging: calls.Paging[request.Config[ListOpts]]{
			Limit:     calls.Value(func(c *request.Config[ListOpts]) *int { return &c.Options.Limit }),
			Marker:    calls.Value(func(c *request.Config[ListOpts]) *string { return &c.Options.Marker }),
			MaxItems:  calls.Value(func(c *request.Config[ListOpts]) *int { return &c.Options.MaxItems }),
			Paginated: calls.Pointer(func(c *request.Config[ListOpts]) **bool { return &c.Options.Paginated }),
		},
	}
)

func pointer[T any](value T) *T { return &value }
func copyPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	return pointer(*value)
}
func copyFields(value map[string]json.RawMessage) map[string]json.RawMessage {
	out := maps.Clone(value)
	for key, raw := range out {
		out[key] = bytes.Clone(raw)
	}
	return out
}
func copyCreate(value CreateOpts) CreateOpts {
	value.Attributes = copyFields(value.Attributes)
	value.Microversion = copyPointer(value.Microversion)
	return value
}
func copyList(value ListOpts) ListOpts {
	value.Microversion = copyPointer(value.Microversion)
	value.Paginated = copyPointer(value.Paginated)
	value.Filters = append([]resource.ListOption(nil), value.Filters...)
	return value
}
func WithCreateOptions(value CreateOpts) CreateOption {
	owned := copyCreate(value)
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options = copyCreate(owned); return nil })
}

// WithCreateAttribute retains omission and explicit JSON null. Unknown ordinary
// attributes are discarded before invoking their JSON marshaler.
func WithCreateAttribute(key string, value any) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error {
		if control(key) {
			return unsupportedControl(key)
		}
		if !knownField(key) {
			return nil
		}
		if key == "share_id" {
			return invalid("share_id is supplied by the share argument")
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if c.Options.Attributes == nil {
			c.Options.Attributes = make(map[string]json.RawMessage)
		}
		c.Options.Attributes[key] = raw
		return nil
	})
}
func WithListOptions(value ListOpts) ListOption {
	owned := copyList(value)
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options = copyList(owned); return nil })
}

// WithListFilter classifies attributes before serialization so ignored Source
// kwargs and reserved controls never invoke a caller marshaler.
func WithListFilter(key string, value any) ListOption {
	return request.Custom(func(c *request.Config[ListOpts]) error {
		if control(key) {
			return unsupportedControl(key)
		}
		if !knownField(key) && key != "limit" && key != "marker" {
			return nil
		}
		c.Options.Filters = append(c.Options.Filters, resource.WithFilter(key, value))
		return nil
	})
}
func WithListFilters(value map[string]any) ListOption {
	owned := maps.Clone(value)
	return request.Custom(func(c *request.Config[ListOpts]) error {
		accepted := make(map[string]any)
		for key, value := range owned {
			if control(key) {
				return unsupportedControl(key)
			}
			if knownField(key) || key == "limit" || key == "marker" {
				accepted[key] = value
			}
		}
		c.Options.Filters = append(c.Options.Filters, resource.WithFilters(accepted))
		return nil
	})
}
