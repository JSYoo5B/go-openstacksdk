package flavors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FlavorListOpts owns summary/detail, conditional enrichment and local row
// controls. Nil Details is true; GetExtraSpecs is false. Nil Microversion
// retains the selected version or discovers at most 2.61.
type FlavorListOpts struct {
	Details       *bool
	GetExtraSpecs bool
	Limit         int
	Marker        string
	MaxItems      int
	Paginated     *bool
	Microversion  *string
}
type FlavorListOption = request.Option[FlavorListOpts]

func copyFlavorListOptions(value FlavorListOpts) FlavorListOpts {
	if value.Details != nil {
		owned := *value.Details
		value.Details = &owned
	}
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
func WithFlavorListOptions(value FlavorListOpts) FlavorListOption {
	owned := copyFlavorListOptions(value)
	return func(c *request.Config[FlavorListOpts]) error { c.Options = copyFlavorListOptions(owned); return nil }
}
func WithFlavorListDetails(value bool) FlavorListOption {
	return func(c *request.Config[FlavorListOpts]) error { owned := value; c.Options.Details = &owned; return nil }
}
func WithFlavorListExtraSpecs(value bool) FlavorListOption {
	return func(c *request.Config[FlavorListOpts]) error { c.Options.GetExtraSpecs = value; return nil }
}
func WithFlavorListLimit(value int) FlavorListOption {
	return func(c *request.Config[FlavorListOpts]) error { c.Options.Limit = value; return nil }
}
func WithFlavorListMarker(value string) FlavorListOption {
	return func(c *request.Config[FlavorListOpts]) error { c.Options.Marker = value; return nil }
}
func WithFlavorListMaxItems(value int) FlavorListOption {
	return func(c *request.Config[FlavorListOpts]) error { c.Options.MaxItems = value; return nil }
}
func WithFlavorListPaginated(value bool) FlavorListOption {
	return func(c *request.Config[FlavorListOpts]) error {
		owned := value
		c.Options.Paginated = &owned
		return nil
	}
}
func WithFlavorListMicroversion(value string) FlavorListOption {
	return func(c *request.Config[FlavorListOpts]) error {
		owned := value
		c.Options.Microversion = &owned
		return nil
	}
}
func WithFlavorListHeader(key, value string) FlavorListOption {
	return request.WithHeader[FlavorListOpts](key, value)
}
func WithFlavorListQuery(key, value string) FlavorListOption {
	return request.WithQuery[FlavorListOpts](key, value)
}

const flavorRecordFiltersArgument = "flavor_record_filters"

type flavorFilterCapture struct{ options []resource.ListOption }

func capturedFlavorFilters(arguments map[string]any) ([]resource.ListOption, error) {
	value, present := arguments[flavorRecordFiltersArgument]
	if !present {
		return nil, nil
	}
	capture, ok := value.(flavorFilterCapture)
	if !ok {
		return nil, fmt.Errorf("%w: invalid SDK flavor filter capture", resource.ErrInvalidOption)
	}
	return slices.Clone(capture.options), nil
}
func flavorFilterOption[T any](option resource.ListOption) request.Option[T] {
	return func(c *request.Config[T]) error {
		selected, err := capturedFlavorFilters(c.Arguments)
		if err != nil {
			return err
		}
		if c.Arguments == nil {
			c.Arguments = make(map[string]any)
		}
		c.Arguments[flavorRecordFiltersArgument] = flavorFilterCapture{options: append(selected, option)}
		return nil
	}
}
func WithFlavorListFilter(field string, value any) FlavorListOption {
	return flavorFilterOption[FlavorListOpts](resource.WithFilter(field, value))
}
func WithFlavorListFilters(values map[string]any) FlavorListOption {
	return flavorFilterOption[FlavorListOpts](resource.WithFilters(values))
}

func ownFlavorReadConfig[T any](config *request.Config[T], clone func(T) T) {
	config.Options = clone(config.Options)
	cloudread.OwnReadConfig(config)
	if value, present := config.Arguments[flavorRecordFiltersArgument]; present {
		if capture, ok := value.(flavorFilterCapture); ok {
			config.Arguments[flavorRecordFiltersArgument] = flavorFilterCapture{options: slices.Clone(capture.options)}
		}
	}
}
func prepareFlavorReadConfig[T any](ctx context.Context, check func(context.Context) error, base T, clone func(T) T, options []request.Option[T]) (request.Config[T], error) {
	config, err := cloudread.ApplyReadOptions(ctx, base, options, func(c *request.Config[T]) { ownFlavorReadConfig(c, clone) }, check)
	if err == nil {
		err = request.ValidateCapabilities(config, false, true, true, flavorRecordFiltersArgument)
	}
	return config, errors.Join(err, check(ctx))
}

func flavorRecordControl(key string) bool {
	switch strings.ToLower(key) {
	case "details", "get_extra_specs", "ignore_missing", "all_projects", "max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters":
		return true
	}
	return false
}
func flavorRecordFilterDescriptor() *resource.FilterDescriptor {
	body := make(map[string]string, 12)
	for _, name := range []string{"id", "name", "original_name", "description", "disk", "ram", "vcpus", "swap", "ephemeral", "is_disabled", "rxtx_factor", "extra_specs"} {
		body[name] = name
	}
	return &resource.FilterDescriptor{
		Query: map[string]string{"limit": "limit", "marker": "marker", "sort_key": "sort_key", "sort_dir": "sort_dir", "is_public": "is_public", "min_disk": "minDisk", "min_ram": "minRam"}, Body: body,
		Reserved: []string{"details", "get_extra_specs", "ignore_missing", "all_projects", "max_items", "paginated", "session", "resource_type", "base_path", "list_base_path", "allow_unknown_params", "microversion", "headers", "jmespath_filters"},
	}
}
func baseFlavorQuery(value FlavorListOpts, rawQuery url.Values) (url.Values, error) {
	if value.Limit < 0 || value.MaxItems < 0 {
		return nil, fmt.Errorf("%w: flavor limit and raw row cap must be non-negative", resource.ErrInvalidOption)
	}
	query := make(url.Values, len(rawQuery)+2)
	if value.Limit > 0 {
		query.Set("limit", strconv.Itoa(value.Limit))
	}
	if value.Marker != "" {
		query.Set("marker", value.Marker)
	}
	for key, values := range rawQuery {
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("%w: empty flavor query key", resource.ErrInvalidOption)
		}
		if flavorRecordControl(key) {
			return nil, fmt.Errorf("%w: flavor control %q requires a dedicated option", resource.ErrUnsupported, key)
		}
		query[key] = slices.Clone(values)
	}
	return query, nil
}

type flavorListParameters struct {
	query         url.Values
	control       rest.ListControl
	selection     *resource.FilterSelection
	headers       map[string]string
	microversion  *string
	details       bool
	getExtraSpecs bool
}

func flavorReadParameters(value FlavorListOpts, rawQuery url.Values, headers map[string]string, selection *resource.FilterSelection) (flavorListParameters, error) {
	query, err := baseFlavorQuery(value, rawQuery)
	if err != nil {
		return flavorListParameters{}, err
	}
	for key, values := range selection.Query {
		if _, present := query[key]; present {
			return flavorListParameters{}, fmt.Errorf("%w: semantic flavor query %q conflicts with typed/raw query", resource.ErrInvalidOption, key)
		}
		query[key] = slices.Clone(values)
	}
	isPublic, present, err := selection.Attribute("is_public")
	if err != nil {
		return flavorListParameters{}, err
	}
	if _, selected := query["is_public"]; !selected || present && bytes.Equal(bytes.TrimSpace(isPublic), []byte("null")) {
		query.Set("is_public", "None")
	}
	// Null/empty-array values are absent from the actual URL. Only the
	// flavor-specific is_public null has the special None spelling above.
	for key, values := range query {
		if len(values) == 0 {
			delete(query, key)
		}
	}
	if values, present := query["limit"]; present {
		limit, err := strconv.Atoi(query.Get("limit"))
		if len(values) != 1 || err != nil || limit < 1 {
			return flavorListParameters{}, fmt.Errorf("%w: flavor limit must be one positive integer", resource.ErrInvalidOption)
		}
	}
	if values, present := query["marker"]; present && (len(values) != 1 || strings.TrimSpace(values[0]) == "") {
		return flavorListParameters{}, fmt.Errorf("%w: flavor marker must be one nonempty value", resource.ErrInvalidOption)
	}
	return flavorListParameters{query: query, control: rest.ListControl{MaxItems: value.MaxItems, SinglePage: value.Paginated != nil && !*value.Paginated, LimitHint: true}, selection: selection, headers: maps.Clone(headers), microversion: value.Microversion, details: value.Details == nil || *value.Details, getExtraSpecs: value.GetExtraSpecs}, nil
}
func prepareFlavorList(ctx context.Context, check func(context.Context) error, options []FlavorListOption) (flavorListParameters, error) {
	config, err := prepareFlavorReadConfig(ctx, check, FlavorListOpts{}, copyFlavorListOptions, options)
	if err != nil {
		return flavorListParameters{}, err
	}
	filters, err := capturedFlavorFilters(config.Arguments)
	if err != nil {
		return flavorListParameters{}, err
	}
	selection, err := resource.PrepareFilterSelection(flavorRecordFilterDescriptor(), filters...)
	if err != nil {
		return flavorListParameters{}, err
	}
	return flavorReadParameters(config.Options, config.Query, config.Headers, selection)
}
