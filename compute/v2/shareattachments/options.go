// Package shareattachments provides Nova server share mappings with SDK-owned
// options, response projection and transport policy.
package shareattachments

import (
	"bytes"
	"encoding/json"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"maps"
)

type ReadOpts struct{ Microversion *string }
type ReadOption = request.Option[ReadOpts]
type CreateOpts struct {
	Attributes   map[string]json.RawMessage
	Microversion *string
}
type CreateOption = request.Option[CreateOpts]
type DeleteOpts struct {
	IgnoreMissing *bool
	Microversion  *string
}
type DeleteOption = request.Option[DeleteOpts]
type ListOpts struct {
	Limit        int
	Marker       string
	MaxItems     int
	Paginated    *bool
	Microversion *string
	Filters      []resource.ListOption
}
type ListOption = request.Option[ListOpts]

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
func WithReadMicroversion(value string) ReadOption {
	return func(c *request.Config[ReadOpts]) error { c.Options.Microversion = pointer(value); return nil }
}
func WithReadHeader(key, value string) ReadOption { return request.WithHeader[ReadOpts](key, value) }
func WithCreateOptions(value CreateOpts) CreateOption {
	owned := copyCreate(value)
	return func(c *request.Config[CreateOpts]) error { c.Options = copyCreate(owned); return nil }
}

// WithCreateAttribute retains omission and explicit JSON null. Unknown ordinary
// attributes are discarded before invoking their JSON marshaler.
func WithCreateAttribute(key string, value any) CreateOption {
	return func(c *request.Config[CreateOpts]) error {
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
	}
}
func WithCreateMicroversion(value string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.Microversion = pointer(value); return nil }
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error { c.Options.IgnoreMissing = pointer(value); return nil }
}
func WithDeleteMicroversion(value string) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error { c.Options.Microversion = pointer(value); return nil }
}
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}
func WithListOptions(value ListOpts) ListOption {
	owned := copyList(value)
	return func(c *request.Config[ListOpts]) error { c.Options = copyList(owned); return nil }
}
func WithListLimit(value int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Limit = value; return nil }
}
func WithListMarker(value string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Marker = value; return nil }
}
func WithListMaxItems(value int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.MaxItems = value; return nil }
}
func WithListPaginated(value bool) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Paginated = pointer(value); return nil }
}
func WithListMicroversion(value string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Microversion = pointer(value); return nil }
}
func WithListHeader(key, value string) ListOption { return request.WithHeader[ListOpts](key, value) }

// WithListFilter classifies attributes before serialization so ignored Source
// kwargs and reserved controls never invoke a caller marshaler.
func WithListFilter(key string, value any) ListOption {
	return func(c *request.Config[ListOpts]) error {
		if control(key) {
			return unsupportedControl(key)
		}
		if !knownField(key) && key != "limit" && key != "marker" {
			return nil
		}
		c.Options.Filters = append(c.Options.Filters, resource.WithFilter(key, value))
		return nil
	}
}
func WithListFilters(value map[string]any) ListOption {
	owned := maps.Clone(value)
	return func(c *request.Config[ListOpts]) error {
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
	}
}
