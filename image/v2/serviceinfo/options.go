package serviceinfo

import (
	"encoding/json"
	"maps"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
)

// ListStoresOpts selects basic or detailed discovery. Limit and Marker are
// caller wire preferences, which Glance's discovery handlers may ignore.
// MaxItems and Paginated are local controls and never generate a wire limit
// or cursor. Only advertised links trigger another page.
type ListStoresOpts struct {
	Details   bool
	Limit     int
	Marker    string
	MaxItems  int
	Paginated *bool
}

type ListStoresOption = request.Option[ListStoresOpts]

type GetImportInfoOpts struct{}
type GetImportInfoOption = request.Option[GetImportInfoOpts]

func copyListStoresOpts(value ListStoresOpts) ListStoresOpts {
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	return value
}

// WithListStoresOptions snapshots and replaces the complete typed policy at
// creation and on every application. Separate query and header options remain.
func WithListStoresOptions(value ListStoresOpts) ListStoresOption {
	owned := copyListStoresOpts(value)
	return func(config *request.Config[ListStoresOpts]) error {
		config.Options = copyListStoresOpts(owned)
		return nil
	}
}

func WithListStoresDetails(value bool) ListStoresOption {
	return func(config *request.Config[ListStoresOpts]) error { config.Options.Details = value; return nil }
}

func WithListStoresMaxItems(value int) ListStoresOption {
	return func(config *request.Config[ListStoresOpts]) error { config.Options.MaxItems = value; return nil }
}

func WithListStoresPaginated(value bool) ListStoresOption {
	return func(config *request.Config[ListStoresOpts]) error {
		owned := value
		config.Options.Paginated = &owned
		return nil
	}
}

// WithListStoresQuery sends an exact wire query. It overrides an identically
// named typed query and does not enable Python local Body filtering.
func WithListStoresQuery(key, value string) ListStoresOption {
	return request.WithQuery[ListStoresOpts](key, value)
}

func WithListStoresHeader(key, value string) ListStoresOption {
	return request.WithHeader[ListStoresOpts](key, value)
}

func WithGetImportInfoHeader(key, value string) GetImportInfoOption {
	return request.WithHeader[GetImportInfoOpts](key, value)
}

func copyConfig[T any](value request.Config[T], copyOptions func(T) T) request.Config[T] {
	value.Options = copyOptions(value.Options)
	fields := make(map[string]json.RawMessage, len(value.Fields))
	for key, raw := range value.Fields {
		fields[key] = append(json.RawMessage(nil), raw...)
	}
	value.Fields = fields
	query := make(url.Values, len(value.Query))
	for key, values := range value.Query {
		query[key] = append([]string(nil), values...)
	}
	value.Query = query
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	value.Arguments = maps.Clone(value.Arguments)
	if value.Arguments == nil {
		value.Arguments = make(map[string]any)
	}
	return value
}

func applyOptions[T any](base T, options []request.Option[T], copyOptions func(T) T) (request.Config[T], error) {
	config := copyConfig(request.Config[T]{Options: base}, copyOptions)
	for _, apply := range options {
		if apply == nil {
			return config, infoInvalid("nil discovery option")
		}
		// Each callback receives a separate handle. Retaining it cannot affect
		// later callbacks or the final prepared request and iteration policy.
		working := copyConfig(config, copyOptions)
		if err := apply(&working); err != nil {
			return config, err
		}
		config = copyConfig(working, copyOptions)
	}
	return config, nil
}

func prepareListStores(options []ListStoresOption) (ListStoresOpts, url.Values, map[string]string, rest.ListControl, error) {
	config, err := applyOptions(ListStoresOpts{}, options, copyListStoresOpts)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, true)
	}
	if err != nil {
		return ListStoresOpts{}, nil, nil, rest.ListControl{}, err
	}
	value := config.Options
	if value.Limit < 0 || value.MaxItems < 0 || !utf8.ValidString(value.Marker) {
		return value, nil, nil, rest.ListControl{}, infoInvalid("limit and max items must be nonnegative and marker must be valid UTF-8")
	}
	query := make(url.Values)
	if value.Limit != 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.Marker != "" {
		query.Set("marker", value.Marker)
	}
	for key, values := range config.Query {
		if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
			return value, nil, nil, rest.ListControl{}, infoInvalid("invalid discovery query key")
		}
		switch strings.ToLower(key) {
		case "details", "max_items", "maxitems", "paginated", "session", "resource_type", "resourcetype", "base_path", "basepath", "list_base_path", "listbasepath", "allow_unknown_params", "allowunknownparams", "microversion", "headers", "jmespath_filters", "jmespathfilters":
			return value, nil, nil, rest.ListControl{}, infoInvalid("query %q requires a dedicated SDK option", key)
		}
		for _, text := range values {
			if !utf8.ValidString(text) {
				return value, nil, nil, rest.ListControl{}, infoInvalid("discovery query values must be valid UTF-8")
			}
		}
		query[key] = append([]string(nil), values...)
	}
	headers, err := infoHeaders(config.Headers, false, "")
	return value, query, headers, rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated}, err
}

func prepareGetImportInfo(options []GetImportInfoOption) (map[string]string, error) {
	config, err := applyOptions(GetImportInfoOpts{}, options, func(value GetImportInfoOpts) GetImportInfoOpts { return value })
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, true)
	}
	if err != nil {
		return nil, err
	}
	return infoHeaders(config.Headers, false, "")
}
