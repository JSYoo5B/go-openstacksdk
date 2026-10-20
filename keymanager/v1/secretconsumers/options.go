package secretconsumers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ConsumerOpts identifies an association, not a URL or a Consumer resource ID.
// ResourceType/ResourceID may be service-specific strings; no UUID is guessed.
type ConsumerOpts struct {
	Service      string `json:"service"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

type CreateOption = request.Option[ConsumerOpts]
type DeleteOpts struct {
	ConsumerOpts
	IgnoreMissing *bool `json:"-"`
}
type DeleteOption = request.Option[DeleteOpts]

func WithCreateOptions(value ConsumerOpts) CreateOption { return request.WithOptions(value) }
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[ConsumerOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[ConsumerOpts](key, value)
}

func WithDeleteOptions(value DeleteOpts) DeleteOption {
	owned := copyDelete(value)
	return func(config *request.Config[DeleteOpts]) error { config.Options = copyDelete(owned); return nil }
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(config *request.Config[DeleteOpts]) error {
		owned := value
		config.Options.IgnoreMissing = &owned
		return nil
	}
}
func WithDeleteField(key string, value any) DeleteOption {
	return request.WithField[DeleteOpts](key, value)
}
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}

func copyDelete(value DeleteOpts) DeleteOpts {
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	return value
}

func consumerBody[T any](config request.Config[T], value ConsumerOpts) (json.RawMessage, error) {
	if err := request.ValidateCapabilities(config, true, false, true); err != nil {
		return nil, err
	}
	if err := validateHeaders(config.Headers); err != nil {
		return nil, err
	}
	if value.Service == "" || value.ResourceType == "" || value.ResourceID == "" {
		return nil, fmt.Errorf("%w: service, resource_type and resource_id are required", resource.ErrInvalidOption)
	}
	for key := range config.Fields {
		switch strings.ToLower(key) {
		case "service", "resource_type", "resource_id", "secret_id", "id", "secret_ref", "consumers", "name", "status", "created", "updated":
			return nil, fmt.Errorf("%w: body field %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	fields := map[string]any{"service": value.Service, "resource_type": value.ResourceType, "resource_id": value.ResourceID}
	body, err := request.MergeFieldsFor(fields, config.Fields, value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(body)
}

// ListOpts requests a server page. Zero Limit/Offset omit those query fields,
// preserving server defaults (10 and 0). MaxItems and Paginated are local.
type ListOpts struct {
	Limit     int
	Offset    int
	MaxItems  int
	Paginated *bool
}
type ListOption = request.Option[ListOpts]

func copyList(value ListOpts) ListOpts {
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	return value
}
func WithListOptions(value ListOpts) ListOption {
	owned := copyList(value)
	return func(config *request.Config[ListOpts]) error { config.Options = copyList(owned); return nil }
}
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }
func WithListMaxItems(value int) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.MaxItems = value; return nil }
}
func WithListPaginated(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		owned := value
		config.Options.Paginated = &owned
		return nil
	}
}

func prepareList(options []ListOption) (url.Values, rest.ListControl, error) {
	config, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, false)
	}
	if err != nil {
		return nil, rest.ListControl{}, err
	}
	value := copyList(config.Options)
	if value.Limit < 0 || value.Offset < 0 || value.MaxItems < 0 {
		return nil, rest.ListControl{}, fmt.Errorf("%w: limit, offset and max items must be non-negative", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	if value.Limit > 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.Offset > 0 {
		query.Set("offset", strconv.Itoa(value.Offset))
	}
	for key, values := range config.Query {
		if strings.TrimSpace(key) == "" {
			return nil, rest.ListControl{}, fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
		}
		if (key == "limit" && value.Limit > 0) || (key == "offset" && value.Offset > 0) {
			return nil, rest.ListControl{}, fmt.Errorf("%w: query %q conflicts with the concrete list option", resource.ErrInvalidOption, key)
		}
		switch strings.ToLower(key) {
		case "marker", "max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters", "secret_id":
			return nil, rest.ListControl{}, fmt.Errorf("%w: query %q is not a server paging option", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated}, nil
}

func validateHeaders(headers map[string]string) error {
	seen := make(map[string]string, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, " \t\r\n:") || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%w: invalid consumer header", resource.ErrInvalidOption)
		}
		canonical := strings.ToLower(key)
		if previous, exists := seen[canonical]; exists && previous != value {
			return fmt.Errorf("%w: conflicting case variants of header %q", resource.ErrInvalidOption, key)
		}
		seen[canonical] = value
		switch canonical {
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-type", "content-length", "openstack-api-version":
			return fmt.Errorf("%w: header %q is owned by the SDK", resource.ErrInvalidOption, key)
		}
	}
	return nil
}
