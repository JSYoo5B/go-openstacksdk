package metadefnamespaces

import (
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/internal/glancecalls"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts configures the bounded namespace create operation.
type CreateOpts struct {
	Headers                  map[string]string
	DisplayName              *string
	Description              *string
	Visibility               *string
	Owner                    *string
	Protected                *bool
	Properties               map[string]PropertyDefinition
	Objects                  []ObjectDefinition
	Tags                     []TagDefinition
	ResourceTypeAssociations []ResourceTypeAssociationDefinition
}

// CreateOption is one option of the operation: WithCreateOpts,
// WithCreateHeaders, WithCreateDisplayName, WithCreateDescription,
// WithCreateVisibility, WithCreateOwner, WithCreateProtected, request.Header,
// or request.Custom over *CreateOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func copyCreate(value CreateOpts) CreateOpts {
	value.Headers = copyHeaders(value.Headers)
	value.DisplayName = copyPointer(value.DisplayName)
	value.Description = copyPointer(value.Description)
	value.Visibility = copyPointer(value.Visibility)
	value.Owner = copyPointer(value.Owner)
	value.Protected = copyPointer(value.Protected)
	value.Properties = copyPropertyDefinitions(value.Properties)
	value.Objects = copyObjectDefinitions(value.Objects)
	value.Tags = slices.Clone(value.Tags)
	value.ResourceTypeAssociations = copyAssociationDefinitions(value.ResourceTypeAssociations)
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

func WithCreateDisplayName(value string) CreateOption {
	return request.Custom(func(config *CreateOpts) error { owned := value; config.DisplayName = &owned; return nil })
}

func WithCreateDescription(value string) CreateOption {
	return request.Custom(func(config *CreateOpts) error { owned := value; config.Description = &owned; return nil })
}

func WithCreateVisibility(value string) CreateOption {
	return request.Custom(func(config *CreateOpts) error { owned := value; config.Visibility = &owned; return nil })
}

func WithCreateOwner(value string) CreateOption {
	return request.Custom(func(config *CreateOpts) error { owned := value; config.Owner = &owned; return nil })
}

func WithCreateProtected(value bool) CreateOption {
	return request.Custom(func(config *CreateOpts) error { owned := value; config.Protected = &owned; return nil })
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

// UpdateOption is one option of the operation: WithUpdateOpts,
// WithUpdateHeaders, WithUpdateNamespace, WithUpdateDisplayName,
// WithUpdateDescription, WithUpdateVisibility, WithUpdateOwner,
// WithUpdateProtected, request.Header, or request.Custom over *UpdateOpts.
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

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
	return request.Custom(func(config *UpdateOpts) error { *config = copyUpdate(owned); return nil })
}

func WithUpdateHeaders(value map[string]string) UpdateOption {
	owned := copyHeaders(value)
	return request.Custom(func(config *UpdateOpts) error {
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

func WithUpdateNamespace(value string) UpdateOption {
	return request.Custom(func(config *UpdateOpts) error { owned := value; config.Namespace = &owned; return nil })
}

func WithUpdateDisplayName(value string) UpdateOption {
	return request.Custom(func(config *UpdateOpts) error { owned := value; config.DisplayName = &owned; return nil })
}

func WithUpdateDescription(value string) UpdateOption {
	return request.Custom(func(config *UpdateOpts) error { owned := value; config.Description = &owned; return nil })
}

func WithUpdateVisibility(value string) UpdateOption {
	return request.Custom(func(config *UpdateOpts) error { owned := value; config.Visibility = &owned; return nil })
}

func WithUpdateOwner(value string) UpdateOption {
	return request.Custom(func(config *UpdateOpts) error { owned := value; config.Owner = &owned; return nil })
}

func WithUpdateProtected(value bool) UpdateOption {
	return request.Custom(func(config *UpdateOpts) error { owned := value; config.Protected = &owned; return nil })
}

// GetOpts configures the bounded namespace get operation.
type GetOpts struct {
	Headers      map[string]string
	ResourceType *string
}

// GetOption is one option of the operation: WithGetOpts, WithGetHeaders,
// WithGetResourceType, request.Header, or request.Custom over *GetOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

func copyGet(value GetOpts) GetOpts {
	value.Headers = copyHeaders(value.Headers)
	value.ResourceType = copyPointer(value.ResourceType)
	return value
}

// WithGetOpts snapshots and replaces the complete options value.
func WithGetOpts(value GetOpts) GetOption {
	owned := copyGet(value)
	return request.Custom(func(config *GetOpts) error { *config = copyGet(owned); return nil })
}

func WithGetHeaders(value map[string]string) GetOption {
	owned := copyHeaders(value)
	return request.Custom(func(config *GetOpts) error {
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

func WithGetResourceType(value string) GetOption {
	return request.Custom(func(config *GetOpts) error { owned := value; config.ResourceType = &owned; return nil })
}

// DeleteOpts configures the bounded namespace delete operation.
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

// ListOption is one option of the operation: WithListOpts, WithListHeaders,
// WithListVisibility, WithListSortKey, WithListSortDir,
// WithListResourceTypes, WithListSinglePage, request.Header, request.Limit,
// request.Marker, request.MaxItems, or request.Custom over *ListOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func copyList(value ListOpts) ListOpts {
	value.Headers = copyHeaders(value.Headers)
	value.Limit = copyPointer(value.Limit)
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

func WithListVisibility(value string) ListOption {
	return request.Custom(func(config *ListOpts) error { config.Visibility = value; return nil })
}

func WithListSortKey(value string) ListOption {
	return request.Custom(func(config *ListOpts) error { config.SortKey = value; return nil })
}

func WithListSortDir(value string) ListOption {
	return request.Custom(func(config *ListOpts) error { config.SortDir = value; return nil })
}

func WithListResourceTypes(value string) ListOption {
	return request.Custom(func(config *ListOpts) error { config.ResourceTypes = value; return nil })
}

func WithListSinglePage(value bool) ListOption {
	return request.Custom(func(config *ListOpts) error { config.SinglePage = value; return nil })
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
	value, err := apply(calls.Translate(options, createCalls), copyCreate)
	if err != nil {
		return value, err
	}
	if err = scalars(value.DisplayName, value.Description, value.Visibility, value.Owner); err != nil {
		return value, err
	}
	if err = validateDefinitions(value); err != nil {
		return value, err
	}
	value.Headers, err = validateHeaders(value.Headers, false, "")
	return value, err
}
func prepareUpdate(options []UpdateOption) (UpdateOpts, error) {
	value, err := apply(calls.Translate(options, updateCalls), copyUpdate)
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
	value, err := apply(calls.Translate(options, getCalls), copyGet)
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
	value, err := apply(calls.Translate(options, deleteCalls), copyDelete)
	if err == nil {
		value.Headers, err = validateHeaders(value.Headers, false, "")
	}
	return value, err
}
func prepareList(options []ListOption) (ListOpts, url.Values, error) {
	value, err := apply(calls.Translate(options, listCalls), copyList)
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

var (
	createCalls = glancecalls.HeaderSpec(func(o *CreateOpts) *map[string]string { return &o.Headers })
	updateCalls = glancecalls.HeaderSpec(func(o *UpdateOpts) *map[string]string { return &o.Headers })
	getCalls    = glancecalls.HeaderSpec(func(o *GetOpts) *map[string]string { return &o.Headers })
	deleteCalls = calls.Spec[DeleteOpts]{
		Header:        glancecalls.SetHeader(func(o *DeleteOpts) *map[string]string { return &o.Headers }),
		IgnoreMissing: calls.Pointer(func(o *DeleteOpts) **bool { return &o.IgnoreMissing }),
	}
	listCalls = calls.Spec[ListOpts]{
		Header: glancecalls.SetHeader(func(o *ListOpts) *map[string]string { return &o.Headers }),
		Paging: calls.Paging[ListOpts]{
			Limit:    calls.Pointer(func(o *ListOpts) **int { return &o.Limit }),
			Marker:   calls.Value(func(o *ListOpts) *string { return &o.Marker }),
			MaxItems: calls.Value(func(o *ListOpts) *int { return &o.MaxItems }),
		},
	}
)
