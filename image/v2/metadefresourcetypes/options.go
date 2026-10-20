package metadefresourcetypes

import (
	"maps"
	"net/http"
	"strings"
)

// CreateOpts configures the bounded create operation.
type CreateOpts struct {
	Headers          map[string]string
	Prefix           *string
	PropertiesTarget *string
}
type CreateOption func(*CreateOpts) error

func copyCreate(value CreateOpts) CreateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Prefix = copyPointer(value.Prefix)
	value.PropertiesTarget = copyPointer(value.PropertiesTarget)
	return value
}

// WithCreateOpts snapshots and replaces the complete options value.
func WithCreateOpts(value CreateOpts) CreateOption {
	owned := copyCreate(value)
	return func(config *CreateOpts) error { *config = copyCreate(owned); return nil }
}
func WithCreateHeader(key, value string) CreateOption {
	return func(config *CreateOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithCreateHeaders(value map[string]string) CreateOption {
	owned := copyHeaders(value)
	return func(config *CreateOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithCreatePrefix(value string) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.Prefix = &owned; return nil }
}
func WithCreatePropertiesTarget(value string) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.PropertiesTarget = &owned; return nil }
}

// DeleteOpts configures the bounded delete operation.
type DeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type DeleteOption func(*DeleteOpts) error

func copyDelete(value DeleteOpts) DeleteOpts {
	value.Headers = copyHeaders(value.Headers)
	value.IgnoreMissing = copyPointer(value.IgnoreMissing)
	return value
}

// WithDeleteOpts snapshots and replaces the complete options value.
func WithDeleteOpts(value DeleteOpts) DeleteOption {
	owned := copyDelete(value)
	return func(config *DeleteOpts) error { *config = copyDelete(owned); return nil }
}
func WithDeleteHeader(key, value string) DeleteOption {
	return func(config *DeleteOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithDeleteHeaders(value map[string]string) DeleteOption {
	owned := copyHeaders(value)
	return func(config *DeleteOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(config *DeleteOpts) error { owned := value; config.IgnoreMissing = &owned; return nil }
}

// ListOpts configures the bounded list operation.
type ListOpts struct {
	Headers  map[string]string
	MaxItems int
}
type ListOption func(*ListOpts) error

func copyList(value ListOpts) ListOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithListOpts snapshots and replaces the complete options value.
func WithListOpts(value ListOpts) ListOption {
	owned := copyList(value)
	return func(config *ListOpts) error { *config = copyList(owned); return nil }
}
func WithListHeader(key, value string) ListOption {
	return func(config *ListOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithListHeaders(value map[string]string) ListOption {
	owned := copyHeaders(value)
	return func(config *ListOpts) error {
		headers, err := validateHeaders(owned, false, "")
		if err != nil {
			return err
		}
		for key, value := range headers {
			config.Headers = setHeader(config.Headers, key, value)
		}
		return nil
	}
}
func WithListMaxItems(value int) ListOption {
	return func(config *ListOpts) error { config.MaxItems = value; return nil }
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
	value, err := apply(options, copyCreate)
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
	value, err := apply(options, copyDelete)
	if err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareList(options []ListOption) (ListOpts, error) {
	value, err := apply(options, copyList)
	if err != nil {
		return value, err
	}
	if value.MaxItems < 0 {
		return value, invalid("max items must be nonnegative")
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
