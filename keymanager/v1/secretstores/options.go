package secretstores

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// ListOpts exposes the pinned SecretStore query names. Empty string fields are
// omitted; WithListQuery can transmit an explicit empty value. GlobalDefault
// preserves omitted versus false. MaxItems/Paginated are local controls.
type ListOpts struct {
	Limit             int
	Marker            string
	Name              string
	Status            string
	GlobalDefault     *bool
	CryptoPlugin      string
	SecretStorePlugin string
	Created           string
	Updated           string
	MaxItems          int
	Paginated         *bool
}

type ListOption = request.Option[ListOpts]

func copyOptions(value ListOpts) ListOpts {
	if value.GlobalDefault != nil {
		owned := *value.GlobalDefault
		value.GlobalDefault = &owned
	}
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	return value
}

// WithListOptions owns both pointer values at creation and on each application.
func WithListOptions(value ListOpts) ListOption {
	owned := copyOptions(value)
	return func(config *request.Config[ListOpts]) error {
		config.Options = copyOptions(owned)
		return nil
	}
}

func WithListGlobalDefault(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		owned := value
		config.Options.GlobalDefault = &owned
		return nil
	}
}

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

// WithListQuery is a wire-only extension. It overrides an identically named
// typed query and does not enable Python Body filtering or attribute aliases.
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func prepareList(options []ListOption) (url.Values, rest.ListControl, []resource.ListOption, error) {
	config, err := request.Apply(ListOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, false, listFiltersArgument)
	}
	if err != nil {
		return nil, rest.ListControl{}, nil, err
	}
	filters, err := capturedListFilters(config.Arguments)
	if err != nil {
		return nil, rest.ListControl{}, nil, err
	}
	value := copyOptions(config.Options)
	if value.Limit < 0 || value.MaxItems < 0 {
		return nil, rest.ListControl{}, nil, fmt.Errorf("%w: limit and max items must be non-negative", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	for key, text := range map[string]string{
		"marker": value.Marker, "name": value.Name, "status": value.Status,
		"crypto_plugin": value.CryptoPlugin, "secret_store_plugin": value.SecretStorePlugin,
		"created": value.Created, "updated": value.Updated,
	} {
		if text != "" {
			query.Set(key, text)
		}
	}
	if value.Limit != 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.GlobalDefault != nil {
		query.Set("global_default", strconv.FormatBool(*value.GlobalDefault))
	}
	for key, values := range config.Query {
		if strings.TrimSpace(key) == "" {
			return nil, rest.ListControl{}, nil, fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
		}
		switch strings.ToLower(key) {
		case "max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters":
			return nil, rest.ListControl{}, nil, fmt.Errorf("%w: query %q requires a dedicated SDK option", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true}, filters, nil
}
