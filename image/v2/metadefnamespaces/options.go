package metadefnamespaces

import (
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CreateOpts configures the bounded namespace create operation.
type CreateOpts struct {
	Headers     map[string]string
	DisplayName *string
	Description *string
	Visibility  *string
	Owner       *string
	Protected   *bool
}
type CreateOption func(*CreateOpts) error

func copyCreate(value CreateOpts) CreateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.DisplayName = copyPointer(value.DisplayName)
	value.Description = copyPointer(value.Description)
	value.Visibility = copyPointer(value.Visibility)
	value.Owner = copyPointer(value.Owner)
	value.Protected = copyPointer(value.Protected)
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

func WithCreateDisplayName(value string) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.DisplayName = &owned; return nil }
}

func WithCreateDescription(value string) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.Description = &owned; return nil }
}

func WithCreateVisibility(value string) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.Visibility = &owned; return nil }
}

func WithCreateOwner(value string) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.Owner = &owned; return nil }
}

func WithCreateProtected(value bool) CreateOption {
	return func(config *CreateOpts) error { owned := value; config.Protected = &owned; return nil }
}

// UpdateOpts configures the bounded namespace update operation.
type UpdateOpts struct {
	Headers     map[string]string
	Namespace   *string
	DisplayName *string
	Description *string
	Visibility  *string
	Owner       *string
	Protected   *bool
}
type UpdateOption func(*UpdateOpts) error

func copyUpdate(value UpdateOpts) UpdateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Namespace = copyPointer(value.Namespace)
	value.DisplayName = copyPointer(value.DisplayName)
	value.Description = copyPointer(value.Description)
	value.Visibility = copyPointer(value.Visibility)
	value.Owner = copyPointer(value.Owner)
	value.Protected = copyPointer(value.Protected)
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

func WithUpdateNamespace(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Namespace = &owned; return nil }
}

func WithUpdateDisplayName(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.DisplayName = &owned; return nil }
}

func WithUpdateDescription(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Description = &owned; return nil }
}

func WithUpdateVisibility(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Visibility = &owned; return nil }
}

func WithUpdateOwner(value string) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Owner = &owned; return nil }
}

func WithUpdateProtected(value bool) UpdateOption {
	return func(config *UpdateOpts) error { owned := value; config.Protected = &owned; return nil }
}

// GetOpts configures the bounded namespace get operation.
type GetOpts struct {
	Headers      map[string]string
	ResourceType *string
}
type GetOption func(*GetOpts) error

func copyGet(value GetOpts) GetOpts {
	value.Headers = copyHeaders(value.Headers)
	value.ResourceType = copyPointer(value.ResourceType)
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

func WithGetResourceType(value string) GetOption {
	return func(config *GetOpts) error { owned := value; config.ResourceType = &owned; return nil }
}

// DeleteOpts configures the bounded namespace delete operation.
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

// ListOpts configures the bounded namespace list operation.
type ListOpts struct {
	Headers       map[string]string
	Limit         *int
	Marker        string
	Visibility    string
	SortKey       string
	SortDir       string
	ResourceTypes string
	MaxItems      int
	SinglePage    bool
}
type ListOption func(*ListOpts) error

func copyList(value ListOpts) ListOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Limit = copyPointer(value.Limit)
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

func WithListLimit(value int) ListOption {
	return func(config *ListOpts) error { owned := value; config.Limit = &owned; return nil }
}

func WithListMarker(value string) ListOption {
	return func(config *ListOpts) error { config.Marker = value; return nil }
}

func WithListVisibility(value string) ListOption {
	return func(config *ListOpts) error { config.Visibility = value; return nil }
}

func WithListSortKey(value string) ListOption {
	return func(config *ListOpts) error { config.SortKey = value; return nil }
}

func WithListSortDir(value string) ListOption {
	return func(config *ListOpts) error { config.SortDir = value; return nil }
}

func WithListResourceTypes(value string) ListOption {
	return func(config *ListOpts) error { config.ResourceTypes = value; return nil }
}

func WithListMaxItems(value int) ListOption {
	return func(config *ListOpts) error { config.MaxItems = value; return nil }
}

func WithListSinglePage(value bool) ListOption {
	return func(config *ListOpts) error { config.SinglePage = value; return nil }
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
			return config, invalid("nil namespace option")
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
	if err = scalars(value.DisplayName, value.Description, value.Visibility, value.Owner); err != nil {
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
	if value.Namespace != nil {
		if err = literal(*value.Namespace); err != nil {
			return value, err
		}
	}
	if err = scalars(value.DisplayName, value.Description, value.Visibility, value.Owner); err != nil {
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
	if value.ResourceType != nil {
		if err = queryText(*value.ResourceType); err != nil {
			return value, err
		}
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareDelete(options []DeleteOption) (DeleteOpts, error) {
	value, err := apply(options, copyDelete)
	if err == nil {
		value.Headers, err = validateHeaders(value.Headers, false, "")
	}
	return value, err
}
func prepareList(options []ListOption) (ListOpts, url.Values, error) {
	value, err := apply(options, copyList)
	if err != nil {
		return value, nil, err
	}
	if value.Limit != nil && *value.Limit < 0 || value.MaxItems < 0 {
		return value, nil, invalid("limit and max items must be nonnegative")
	}
	if value.Marker != "" {
		if err = literal(value.Marker); err != nil {
			return value, nil, err
		}
	}
	if value.Visibility != "" && value.Visibility != "public" && value.Visibility != "private" {
		return value, nil, invalid("visibility must be public or private")
	}
	if value.SortDir != "" && value.SortDir != "asc" && value.SortDir != "desc" {
		return value, nil, invalid("sort direction must be asc or desc")
	}
	for _, text := range []string{value.SortKey, value.ResourceTypes} {
		if err = queryText(text); err != nil {
			return value, nil, err
		}
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	if err != nil {
		return value, nil, err
	}
	query := make(url.Values)
	if value.Limit != nil {
		query.Set("limit", strconv.Itoa(*value.Limit))
	}
	for _, field := range []struct{ key, value string }{{"marker", value.Marker}, {"visibility", value.Visibility}, {"sort_key", value.SortKey}, {"sort_dir", value.SortDir}, {"resource_types", value.ResourceTypes}} {
		if field.value != "" {
			query.Set(field.key, field.value)
		}
	}
	return value, query, nil
}
