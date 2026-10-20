package metadefobjects

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// CreateOpts configures the bounded create operation.
type CreateOpts struct {
	Headers     map[string]string
	Description *string
	Properties  map[string]json.RawMessage
	Required    []string
}
type CreateOption func(*CreateOpts) error

func copyCreate(value CreateOpts) CreateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Description = copyPointer(value.Description)
	value.Properties = copyProperties(value.Properties)
	value.Required = slices.Clone(value.Required)
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
func WithCreateDescription(value string) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.Description = &owned; return nil }
}
func WithCreateProperties(value map[string]json.RawMessage) CreateOption {
	owned := copyProperties(value)
	return func(config *CreateOpts) error { config.Properties = copyProperties(owned); return nil }
}
func WithCreateRequired(value []string) CreateOption {
	owned := slices.Clone(value)
	return func(config *CreateOpts) error { config.Required = slices.Clone(owned); return nil }
}

// UpdateOpts configures the bounded update operation.
type UpdateOpts struct {
	Headers     map[string]string
	Name        *string
	Description *string
	Properties  map[string]json.RawMessage
	Required    []string
}
type UpdateOption func(*UpdateOpts) error

func copyUpdate(value UpdateOpts) UpdateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Name = copyPointer(value.Name)
	value.Description = copyPointer(value.Description)
	value.Properties = copyProperties(value.Properties)
	value.Required = slices.Clone(value.Required)
	return value
}

// WithUpdateOpts snapshots and replaces the complete options value.
func WithUpdateOpts(value UpdateOpts) UpdateOption {
	owned := copyUpdate(value)
	return func(config *UpdateOpts) error { *config = copyUpdate(owned); return nil }
}
func WithUpdateHeader(key, value string) UpdateOption {
	return func(config *UpdateOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithUpdateHeaders(value map[string]string) UpdateOption {
	owned := copyHeaders(value)
	return func(config *UpdateOpts) error {
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
func WithUpdateName(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Name = &owned; return nil }
}
func WithUpdateDescription(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Description = &owned; return nil }
}
func WithUpdateProperties(value map[string]json.RawMessage) UpdateOption {
	owned := copyProperties(value)
	return func(config *UpdateOpts) error { config.Properties = copyProperties(owned); return nil }
}
func WithUpdateRequired(value []string) UpdateOption {
	owned := slices.Clone(value)
	return func(config *UpdateOpts) error { config.Required = slices.Clone(owned); return nil }
}

// GetOpts configures the bounded get operation.
type GetOpts struct {
	Headers map[string]string
}
type GetOption func(*GetOpts) error

func copyGet(value GetOpts) GetOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithGetOpts snapshots and replaces the complete options value.
func WithGetOpts(value GetOpts) GetOption {
	owned := copyGet(value)
	return func(config *GetOpts) error { *config = copyGet(owned); return nil }
}
func WithGetHeader(key, value string) GetOption {
	return func(config *GetOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithGetHeaders(value map[string]string) GetOption {
	owned := copyHeaders(value)
	return func(config *GetOpts) error {
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

// DeleteAllOpts configures the bounded deleteall operation.
type DeleteAllOpts struct {
	Headers map[string]string
}
type DeleteAllOption func(*DeleteAllOpts) error

func copyDeleteAll(value DeleteAllOpts) DeleteAllOpts {
	value.Headers = copyHeaders(value.Headers)
	return value
}

// WithDeleteAllOpts snapshots and replaces the complete options value.
func WithDeleteAllOpts(value DeleteAllOpts) DeleteAllOption {
	owned := copyDeleteAll(value)
	return func(config *DeleteAllOpts) error { *config = copyDeleteAll(owned); return nil }
}
func WithDeleteAllHeader(key, value string) DeleteAllOption {
	return func(config *DeleteAllOpts) error { config.Headers = setHeader(config.Headers, key, value); return nil }
}
func WithDeleteAllHeaders(value map[string]string) DeleteAllOption {
	owned := copyHeaders(value)
	return func(config *DeleteAllOpts) error {
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
func copyProperties(value map[string]json.RawMessage) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	owned := make(map[string]json.RawMessage, len(value))
	for key, raw := range value {
		owned[key] = slices.Clone(raw)
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
			return config, invalid("nil object option")
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
	if err = payload(value.Description, value.Properties, value.Required); err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareUpdate(options []UpdateOption) (UpdateOpts, error) {
	value, err := apply(options, copyUpdate)
	if err != nil {
		return value, err
	}
	if value.Name != nil {
		if err = literal(*value.Name); err != nil {
			return value, err
		}
	}
	if err = payload(value.Description, value.Properties, value.Required); err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareGet(options []GetOption) (GetOpts, error) {
	value, err := apply(options, copyGet)
	if err != nil {
		return value, err
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
func prepareDeleteAll(options []DeleteAllOption) (DeleteAllOpts, error) {
	value, err := apply(options, copyDeleteAll)
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
