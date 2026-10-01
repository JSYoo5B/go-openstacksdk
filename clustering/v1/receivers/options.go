package receivers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type CreateOpts struct {
	Name      string                   `json:"name"`
	Type      string                   `json:"type"`
	ClusterID request.Optional[string] `json:"cluster_id,omitzero"`
	Action    request.Optional[string] `json:"action,omitzero"`
	Actor     json.RawMessage          `json:"actor,omitempty"`
	Params    json.RawMessage          `json:"params,omitempty"`
}

type UpdateOpts struct {
	Name   request.Optional[string] `json:"name,omitzero"`
	Action request.Optional[string] `json:"action,omitzero"`
	Params json.RawMessage          `json:"params,omitempty"`
}

type ListOpts struct {
	// MaxItems counts wire rows before local filtering; zero is unlimited.
	MaxItems int
	// Paginated nil uses all pages; an explicit false returns one page.
	Paginated     *bool
	Limit         int
	Marker        string
	Name          string
	Type          string
	ClusterID     string
	Action        string
	Sort          string
	GlobalProject *bool
	UserID        string
}

type CreateOption = request.Option[CreateOpts]
type UpdateOption = request.Option[UpdateOpts]
type ListOption = request.Option[ListOpts]

func WithCreateOptions(value CreateOpts) CreateOption { return senlin.Snapshot(value) }
func WithUpdateOptions(value UpdateOpts) UpdateOption { return senlin.Snapshot(value) }
func WithListOptions(value ListOpts) ListOption       { return senlin.Snapshot(value) }
func WithCreateClusterID(value string) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.ClusterID = request.Present(value)
		return nil
	}
}
func WithCreateClusterIDNull() CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.ClusterID = request.Null[string]()
		return nil
	}
}
func WithCreateAction(value string) CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.Action = request.Present(value)
		return nil
	}
}
func WithCreateActionNull() CreateOption {
	return func(config *request.Config[CreateOpts]) error {
		config.Options.Action = request.Null[string]()
		return nil
	}
}
func WithUpdateName(value string) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Name = request.Present(value)
		return nil
	}
}
func WithUpdateNameNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Name = request.Null[string]()
		return nil
	}
}
func WithUpdateAction(value string) UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Action = request.Present(value)
		return nil
	}
}
func WithUpdateActionNull() UpdateOption {
	return func(config *request.Config[UpdateOpts]) error {
		config.Options.Action = request.Null[string]()
		return nil
	}
}

func withJSON[T any](value any, set func(*T, json.RawMessage)) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: receiver JSON input: %v", resource.ErrInvalidOption, err)
		}
		set(&config.Options, append(json.RawMessage(nil), encoded...))
		return nil
	}
}
func WithCreateActor(value any) CreateOption {
	return withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Actor = raw })
}
func WithCreateParams(value any) CreateOption {
	return withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Params = raw })
}
func WithUpdateParams(value any) UpdateOption {
	return withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Params = raw })
}
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithUpdateHeader(key, value string) UpdateOption {
	return request.WithHeader[UpdateOpts](key, value)
}

// WithListMaxItems limits wire rows before local filtering. Zero is unlimited.
func WithListMaxItems(value int) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.MaxItems = value; return nil }
}

// WithListPaginated controls continuation without changing server query fields.
func WithListPaginated(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.Paginated = &copy
		return nil
	}
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }
func WithListGlobalProject(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	}
}
func WithListUserID(value string) ListOption {
	return func(config *request.Config[ListOpts]) error { config.Options.UserID = value; return nil }
}

func validateQuery(ctx context.Context, client *gophercloud.ServiceClient, query url.Values) error {
	if err := senlin.Sort(query.Get("sort"), "name", "type", "action", "cluster_id", "created_at", "user"); err != nil {
		return err
	}
	if query.Has("user_id") {
		return fmt.Errorf("%w: receiver user_id is an SDK alias; wire queries use user", resource.ErrInvalidOption)
	}
	if query.Has("user") {
		return senlin.RequireVersion(ctx, client, 4)
	}
	return nil
}

func listQuery(ctx context.Context, client *gophercloud.ServiceClient, config request.Config[ListOpts]) (url.Values, error) {
	if err := senlin.ValidateListCapabilities(config, localFiltersKey); err != nil {
		return nil, err
	}
	if err := senlin.RejectListControlQuery(config.Query); err != nil {
		return nil, err
	}
	value := config.Options
	if value.MaxItems < 0 {
		return nil, fmt.Errorf("%w: max items must be non-negative", resource.ErrInvalidOption)
	}
	query, err := senlin.Query(request.Config[senlin.ListOpts]{Options: senlin.ListOpts{Limit: value.Limit, Marker: value.Marker}})
	if err != nil {
		return nil, err
	}
	for key, field := range map[string]string{"name": value.Name, "type": value.Type, "cluster_id": value.ClusterID, "action": value.Action, "sort": value.Sort, "user": value.UserID} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "max_items", "paginated", "name", "type", "cluster_id", "action", "sort", "global_project", "user", "user_id", "id", "project", "project_id", "domain", "domain_id", "actor", "params", "channel", "created_at", "updated_at":
			return nil, fmt.Errorf("%w: query %q is a concrete receiver option or local Body filter", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	if err := validateQuery(ctx, client, query); err != nil {
		return nil, err
	}
	return query, nil
}
