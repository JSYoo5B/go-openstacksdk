package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListProjectRecordsOpts supplies query and local paging controls for the
// projects visible to one user. Nil Paginated uses all advertised pages.
// Nil Microversion preserves the selected Identity client's policy.
type ListProjectRecordsOpts struct {
	Limit        int
	Marker       string
	MaxItems     int
	Paginated    *bool
	Microversion *string
}

type ListProjectRecordsOption = request.Option[ListProjectRecordsOpts]

func copyProjectListOptions(value ListProjectRecordsOpts) ListProjectRecordsOpts {
	if value.Paginated != nil {
		owned := *value.Paginated
		value.Paginated = &owned
	}
	if value.Microversion != nil {
		owned := *value.Microversion
		value.Microversion = &owned
	}
	return value
}

// WithProjectListOptions replaces typed options while retaining headers, raw
// queries and semantic filters. Pointer values are owned now and per iteration.
func WithProjectListOptions(value ListProjectRecordsOpts) ListProjectRecordsOption {
	owned := copyProjectListOptions(value)
	return func(config *request.Config[ListProjectRecordsOpts]) error {
		config.Options = copyProjectListOptions(owned)
		return nil
	}
}

// WithProjectListMaxItems caps raw rows before local filtering. Zero is unlimited.
func WithProjectListMaxItems(value int) ListProjectRecordsOption {
	return func(config *request.Config[ListProjectRecordsOpts]) error {
		config.Options.MaxItems = value
		return nil
	}
}

// WithProjectListPaginated(false) consumes only the first page.
func WithProjectListPaginated(value bool) ListProjectRecordsOption {
	return func(config *request.Config[ListProjectRecordsOpts]) error {
		owned := value
		config.Options.Paginated = &owned
		return nil
	}
}

// WithProjectListMicroversion selects an operation-local version, including an
// explicit empty value. It never modifies the original service client.
func WithProjectListMicroversion(value string) ListProjectRecordsOption {
	return func(config *request.Config[ListProjectRecordsOpts]) error {
		owned := value
		config.Options.Microversion = &owned
		return nil
	}
}

func WithProjectListHeader(key, value string) ListProjectRecordsOption {
	return request.WithHeader[ListProjectRecordsOpts](key, value)
}

// WithProjectListQuery passes a wire query without attribute classification.
// It overrides the same typed query; local controls need dedicated options.
func WithProjectListQuery(key, value string) ListProjectRecordsOption {
	return request.WithQuery[ListProjectRecordsOpts](key, value)
}

const projectListFiltersArgument = "user_project_filters"

type projectListFilterCapture struct{ options []resource.ListOption }

func capturedProjectListFilters(arguments map[string]any) ([]resource.ListOption, error) {
	value, exists := arguments[projectListFiltersArgument]
	if !exists {
		return nil, nil
	}
	capture, ok := value.(projectListFilterCapture)
	if !ok {
		return nil, fmt.Errorf("%w: invalid SDK user-project filter capture", resource.ErrInvalidOption)
	}
	return append([]resource.ListOption(nil), capture.options...), nil
}

func projectListFilterOption(option resource.ListOption) ListProjectRecordsOption {
	return func(config *request.Config[ListProjectRecordsOpts]) error {
		options, err := capturedProjectListFilters(config.Arguments)
		if err != nil {
			return err
		}
		if config.Arguments == nil {
			config.Arguments = make(map[string]any)
		}
		config.Arguments[projectListFiltersArgument] = projectListFilterCapture{options: append(options, option)}
		return nil
	}
}

// WithProjectListFilter adds a declared query attribute or local Body filter.
// Unknown attributes are discarded. Repeated attributes use the final value.
func WithProjectListFilter(field string, value any) ListProjectRecordsOption {
	return projectListFilterOption(resource.WithFilter(field, value))
}

// WithProjectListFilters replaces semantic filters; nil/empty clears them.
// Canonical attributes take precedence over wire aliases within one bulk map.
// Semantic query keys cannot overlap typed or raw query keys, even if equal.
func WithProjectListFilters(values map[string]any) ListProjectRecordsOption {
	return projectListFilterOption(resource.WithFilters(values))
}

// Own every mutable request carrier between callbacks. Semantic filter options
// already retain immutable JSON snapshots; only their captured slice is copied.
func copyProjectListConfig(config *request.Config[ListProjectRecordsOpts]) {
	config.Options = copyProjectListOptions(config.Options)
	config.Headers = maps.Clone(config.Headers)
	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}
	config.Query = maps.Clone(config.Query)
	if config.Query == nil {
		config.Query = make(url.Values)
	}
	for key, values := range config.Query {
		config.Query[key] = append([]string(nil), values...)
	}
	config.Fields = maps.Clone(config.Fields)
	if config.Fields == nil {
		config.Fields = make(map[string]json.RawMessage)
	}
	for key, raw := range config.Fields {
		config.Fields[key] = append(json.RawMessage(nil), raw...)
	}
	config.Arguments = maps.Clone(config.Arguments)
	if config.Arguments == nil {
		config.Arguments = make(map[string]any)
	}
	if value, exists := config.Arguments[projectListFiltersArgument]; exists {
		if capture, ok := value.(projectListFilterCapture); ok {
			config.Arguments[projectListFiltersArgument] = projectListFilterCapture{options: append([]resource.ListOption(nil), capture.options...)}
		}
	}
}

type projectListParameters struct {
	query        url.Values
	control      rest.ListControl
	filters      []resource.ListOption
	microversion *string
	headers      map[string]string
}

func prepareProjectList(ctx context.Context, guard func(context.Context) error, options []ListProjectRecordsOption) (projectListParameters, error) {
	guarded := make([]ListProjectRecordsOption, len(options))
	for index, option := range options {
		apply := option
		guarded[index] = func(config *request.Config[ListProjectRecordsOpts]) error {
			if err := guard(ctx); err != nil {
				return err
			}
			if apply == nil {
				return fmt.Errorf("%w: nil project-list option", resource.ErrInvalidOption)
			}
			copyProjectListConfig(config)
			applyErr := apply(config)
			copyProjectListConfig(config)
			return errors.Join(applyErr, guard(ctx))
		}
	}
	config, err := request.Apply(ListProjectRecordsOpts{}, guarded...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, true, projectListFiltersArgument)
	}
	if err != nil {
		return projectListParameters{}, err
	}
	filters, err := capturedProjectListFilters(config.Arguments)
	if err != nil {
		return projectListParameters{}, err
	}
	value := copyProjectListOptions(config.Options)
	if value.Limit < 0 || value.MaxItems < 0 {
		return projectListParameters{}, fmt.Errorf("%w: limit and max items must be non-negative", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	if value.Limit != 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.Marker != "" {
		query.Set("marker", value.Marker)
	}
	for key, values := range config.Query {
		if strings.TrimSpace(key) == "" {
			return projectListParameters{}, fmt.Errorf("%w: empty query key", resource.ErrInvalidOption)
		}
		switch strings.ToLower(key) {
		case "user_id", "max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters":
			return projectListParameters{}, fmt.Errorf("%w: query %q requires its dedicated SDK input", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return projectListParameters{
		query:   query,
		control: rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true},
		filters: filters, microversion: value.Microversion, headers: maps.Clone(config.Headers),
	}, nil
}
