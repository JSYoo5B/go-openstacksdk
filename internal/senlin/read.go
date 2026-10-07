// Package senlin supplies private validation and inherited read options for
// concrete Senlin APIs. Service packages own their models and endpoint contracts.
package senlin

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ListOpts reflects inherited Python Resource list controls. MaxItems and
// Paginated are local controls; a positive cap may supply a limit hint for
// catalogs that opt in. A zero Limit and empty Marker omit those wire options.
type ListOpts struct {
	Limit     int
	Marker    string
	MaxItems  int
	Paginated *bool
}

type ListOption = request.Option[ListOpts]

func WithMaxItems(value int) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.MaxItems = value; return nil }
}

func WithPaginated(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.Paginated = &copy
		return nil
	}
}

func Validate(ctx context.Context, client *gophercloud.ServiceClient) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: Senlin service client is required", resource.ErrInvalidOption)
	}
	if client.Type != "" && client.Type != "clustering" {
		return fmt.Errorf("%w: Senlin requires the clustering service type", resource.ErrInvalidOption)
	}
	if client.Microversion != "" {
		if client.Type != "clustering" {
			return fmt.Errorf("%w: Senlin microversion requires the clustering service type", resource.ErrInvalidOption)
		}
		if _, err := Minor(client); err != nil {
			return err
		}
	}
	for key, value := range client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") {
			expected := "clustering " + client.Microversion
			if client.Microversion == "" || value != expected {
				return fmt.Errorf("%w: Senlin version header conflicts with selected microversion", resource.ErrInvalidOption)
			}
		}
	}
	return nil
}

// Minor validates a selected numeric 1.N version. An empty selection uses the
// server's 1.0 default; a symbolic latest cannot prove a version requirement.
func Minor(client *gophercloud.ServiceClient) (int, error) {
	if client == nil {
		return 0, fmt.Errorf("%w: Senlin service client is required", resource.ErrInvalidOption)
	}
	selected := client.Microversion
	if selected == "" {
		return 0, nil
	}
	if selected == "latest" {
		return 0, fmt.Errorf("%w: select a numeric Senlin microversion", resource.ErrUnsupported)
	}
	parts := strings.Split(selected, ".")
	if len(parts) != 2 || parts[0] != "1" {
		return 0, fmt.Errorf("%w: Senlin microversion must be 1.N", resource.ErrInvalidOption)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 || strconv.Itoa(minor) != parts[1] {
		return 0, fmt.Errorf("%w: invalid Senlin microversion %q", resource.ErrInvalidOption, selected)
	}
	return minor, nil
}

func RequireVersion(ctx context.Context, client *gophercloud.ServiceClient, minimum int) error {
	if err := Validate(ctx, client); err != nil {
		return err
	}
	minor, err := Minor(client)
	if err != nil {
		return err
	}
	if minor < minimum {
		return fmt.Errorf("%w: operation requires Senlin microversion 1.%d or newer", resource.ErrUnsupported, minimum)
	}
	return nil
}

func Identifier(value string) error { return resource.ID(value).Validate() }

func Query(config request.Config[ListOpts], allowedArguments ...string) (url.Values, error) {
	if err := request.ValidateCapabilities(config, false, true, false, allowedArguments...); err != nil {
		return nil, err
	}
	if config.Options.Limit < 0 {
		return nil, fmt.Errorf("%w: page limit must be non-negative", resource.ErrInvalidOption)
	}
	if config.Options.MaxItems < 0 {
		return nil, fmt.Errorf("%w: maximum items must be non-negative", resource.ErrInvalidOption)
	}
	query := make(url.Values)
	if config.Options.Limit != 0 {
		query.Set("limit", strconv.Itoa(config.Options.Limit))
	}
	if config.Options.Marker != "" {
		if err := Identifier(config.Options.Marker); err != nil {
			return nil, err
		}
		query.Set("marker", config.Options.Marker)
	}
	for key, values := range config.Query {
		if key == "limit" || key == "marker" || key == "max_items" || key == "paginated" {
			return nil, fmt.Errorf("%w: extension %q is a concrete list option", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}

// ListWithBodyFilters applies owned raw Body filters after REST row validation
// and local pagination controls. Filtered rows still consume the raw row cap.
func ListWithBodyFilters[T any](ctx context.Context, spec rest.CollectionSpec[T], filters BodyFilterSpec, options ...ListOption) iter.Seq2[*T, error] {
	options = append([]ListOption(nil), options...)
	filters = snapshotBodyFilterSpec(filters)
	return func(yield func(*T, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		if err != nil {
			yield(nil, request.Wrap("List", spec.Kind, err))
			return
		}
		query, err := Query(config, filters.Namespace)
		if err == nil {
			err = RejectBodyFilterQuery(query, filters)
		}
		var prepared map[string]json.RawMessage
		if err == nil {
			prepared, err = PrepareBodyFilters(config, filters)
		}
		if err != nil {
			yield(nil, request.Wrap("List", spec.Kind, err))
			return
		}
		control := rest.ListControl{MaxItems: config.Options.MaxItems,
			SinglePage: config.Options.Paginated != nil && !*config.Options.Paginated,
			LimitHint:  spec.Paging.MaxItemsLimitHint}
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if err != nil {
				yield(nil, request.Wrap("List", spec.Kind, err))
				return
			}
			matches, err := MatchFilters(spec.Metadata(value).Body, prepared)
			if err != nil {
				yield(nil, request.Wrap("List", spec.Kind, err))
				return
			}
			if matches && !yield(value, nil) {
				return
			}
		}
	}
}

// ListWithClientBodyFilters prepares each iteration's owned request controls
// before rebuilding the concrete collection gates for its effective client.
func ListWithClientBodyFilters[T any](ctx context.Context, source *gophercloud.ServiceClient, factory func(*gophercloud.ServiceClient) rest.CollectionSpec[T], filters BodyFilterSpec, options ...ListOption) iter.Seq2[*T, error] {
	options = append([]ListOption(nil), options...)
	filters = snapshotBodyFilterSpec(filters)
	return func(yield func(*T, error) bool) {
		kind := "clustering.list"
		if factory != nil {
			kind = factory(source).Kind
		}
		config, err := request.Apply(ListOpts{}, options...)
		if err == nil {
			err = ValidateListCapabilities(config, filters.Namespace)
		}
		var effective *gophercloud.ServiceClient
		if err == nil {
			effective, err = PrepareListClient(ctx, source, config)
		}
		if err == nil && factory == nil {
			err = fmt.Errorf("%w: list collection factory is required", resource.ErrInvalidOption)
		}
		if err != nil {
			yield(nil, request.Wrap("List", kind, err))
			return
		}
		spec := ListSpec(source, effective, factory)
		kind = spec.Kind
		data := ListQueryConfig(config)
		query, err := Query(data, filters.Namespace)
		if err == nil {
			err = RejectListControlQuery(query)
		}
		if err == nil {
			err = RejectBodyFilterQuery(query, filters)
		}
		var prepared map[string]json.RawMessage
		if err == nil {
			prepared, err = PrepareBodyFilters(data, filters)
		}
		if err != nil {
			yield(nil, request.Wrap("List", kind, err))
			return
		}
		control := rest.ListControl{MaxItems: config.Options.MaxItems,
			SinglePage: config.Options.Paginated != nil && !*config.Options.Paginated,
			LimitHint:  spec.Paging.MaxItemsLimitHint}
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if err != nil {
				yield(nil, request.Wrap("List", kind, err))
				return
			}
			matched, err := MatchFilters(spec.Metadata(value).Body, prepared)
			if err != nil {
				yield(nil, request.Wrap("List", kind, err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func List[T any](ctx context.Context, spec rest.CollectionSpec[T], options ...ListOption) iter.Seq2[*T, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*T, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		if err != nil {
			yield(nil, request.Wrap("List", spec.Kind, err))
			return
		}
		query, err := Query(config)
		if err != nil {
			yield(nil, request.Wrap("List", spec.Kind, err))
			return
		}
		control := rest.ListControl{MaxItems: config.Options.MaxItems,
			SinglePage: config.Options.Paginated != nil && !*config.Options.Paginated, LimitHint: true}
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if !yield(value, request.Wrap("List", spec.Kind, err)) {
				return
			}
		}
	}
}

func All[T any](values iter.Seq2[*T, error]) ([]*T, error) {
	all := make([]*T, 0)
	for value, err := range values {
		if err != nil {
			return nil, err
		}
		all = append(all, value)
	}
	return all, nil
}
