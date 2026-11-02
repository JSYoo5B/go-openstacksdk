package metadefresourcetypes

import (
	"maps"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/internal/glancecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts configures the bounded create operation.
type CreateOpts struct {
	Headers          map[string]string
	Prefix           *string
	PropertiesTarget *string
}

// CreateOption is one option of the operation: WithCreateOpts,
// WithCreateHeaders, WithCreatePrefix, WithCreatePropertiesTarget,
// request.Header, or request.Custom over *CreateOpts. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func copyCreate(value CreateOpts) CreateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Prefix = copyPointer(value.Prefix)
	value.PropertiesTarget = copyPointer(value.PropertiesTarget)
	return value
}

// WithCreateOpts snapshots and replaces the complete options value.
func WithCreateOpts(value CreateOpts) CreateOption {
	owned := copyCreate(value)
	return request.Custom(func(config *CreateOpts) error { *config = copyCreate(owned); return nil })
}
func WithCreateHeaders(value map[string]string) CreateOption {
	owned := copyHeaders(value)
	return request.Custom(func(config *CreateOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	})
}
func WithCreatePrefix(value string) CreateOption {
	return request.Custom(func(config *CreateOpts) error { owned := value; config.Prefix = &owned; return nil })
}
func WithCreatePropertiesTarget(value string) CreateOption {
	return request.Custom(func(config *CreateOpts) error { owned := value; config.PropertiesTarget = &owned; return nil })
}

// DeleteOpts configures the bounded delete operation.
type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// DeleteOption is one option of the operation: WithDeleteOpts,
// WithDeleteHeaders, request.Header, request.IgnoreMissing, or request.Custom
// over *DeleteOpts. Other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type DeleteOption = request.DeleteCall

func copyDelete(value DeleteOpts) DeleteOpts {
	value.Headers = copyHeaders(value.Headers)
	value.IgnoreMissing = copyPointer(value.IgnoreMissing)
	return value
}

// WithDeleteOpts snapshots and replaces the complete options value.
func WithDeleteOpts(value DeleteOpts) DeleteOption {
	owned := copyDelete(value)
	return request.Custom(func(config *DeleteOpts) error { *config = copyDelete(owned); return nil })
}
func WithDeleteHeaders(value map[string]string) DeleteOption {
	owned := copyHeaders(value)
	return request.Custom(func(config *DeleteOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	})
}

// ListOpts configures the bounded list operation.
type ListOpts struct {
	Headers  map[string]string
	MaxItems int
}

// ListOption is one option of the operation: WithListOpts, WithListHeaders,
// request.Header, request.MaxItems, or request.Custom over *ListOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func copyList(value ListOpts) ListOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithListOpts snapshots and replaces the complete options value.
func WithListOpts(value ListOpts) ListOption {
	owned := copyList(value)
	return request.Custom(func(config *ListOpts) error { *config = copyList(owned); return nil })
}
func WithListHeaders(value map[string]string) ListOption {
	owned := copyHeaders(value)
	return request.Custom(func(config *ListOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	})
}

func copyPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func copyHeaders(value map[string]string) map[string]string {
	owned := maps.Clone(value)
	if owned == nil {
		owned = make(map[string]string)
	}
	return owned
}
func setHeader(headers map[string]string, key, value string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	for old := range headers {
		if strings.EqualFold(old, key) {
			delete(headers, old)
		}
	}
	headers[http.CanonicalHeaderKey(key)] = value
	return headers
}
func apply[T any, O ~func(*T) error](options []O, copyValue func(T) T) (T, error) {
	var zero T
	config := copyValue(zero)
	for _, option := range options {
		if option == nil {
			return config, invalid("nil resource type option")
		}
		working := copyValue(config)
		if err := option(&working); err != nil {
			return config, err
		}
		config = copyValue(working)
	}
	return config, nil
}
func prepareCreate(options []CreateOption) (CreateOpts, error) {
	value, err := apply(calls.Translate(options, createCalls), copyCreate)
	if err != nil {
		return value, err
	}
	for _, field := range []*string{value.Prefix, value.PropertiesTarget} {
		if field != nil {
			if err = text(*field, 80); err != nil {
				return value, err
			}
		}
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareDelete(options []DeleteOption) (DeleteOpts, error) {
	value, err := apply(calls.Translate(options, deleteCalls), copyDelete)
	if err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareList(options []ListOption) (ListOpts, error) {
	value, err := apply(calls.Translate(options, listCalls), copyList)
	if err != nil {
		return value, err
	}
	if value.MaxItems < 0 {
		return value, invalid("max items must be nonnegative")
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}

var (
	createCalls = glancecalls.HeaderSpec(func(o *CreateOpts) *map[string]string { return &o.Headers })
	deleteCalls = calls.Spec[DeleteOpts]{
		Header:        glancecalls.SetHeader(func(o *DeleteOpts) *map[string]string { return &o.Headers }),
		IgnoreMissing: calls.Pointer(func(o *DeleteOpts) **bool { return &o.IgnoreMissing }),
	}
	listCalls = calls.Spec[ListOpts]{
		Header: glancecalls.SetHeader(func(o *ListOpts) *map[string]string { return &o.Headers }),
		Paging: calls.Paging[ListOpts]{
			MaxItems: calls.Value(func(o *ListOpts) *int { return &o.MaxItems }),
		},
	}
)
