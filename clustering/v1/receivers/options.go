package receivers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
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

// CreateOption is one option of Create: WithCreateOptions, the WithCreate
// field options, request.Header, request.Field or request.Custom over
// *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = senlin.WriteCalls[CreateOpts]()

// UpdateOption is one option of Update, TrackedReceiver.Edit and
// TrackedReceiver.Commit: WithUpdateOptions, the WithUpdate field options,
// request.Header, request.Field or request.Custom over
// *request.Config[UpdateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

var updateCalls = senlin.WriteCalls[UpdateOpts]()

// ListOption is one option of List and All: WithListOptions,
// WithListGlobalProject, WithListUserID, WithListFilter, request.Header,
// request.Microversion, request.Query (a string or []string value),
// request.MaxItems, request.Paginated or request.Custom over
// *request.Config[ListOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = senlin.ListCalls(func(o *ListOpts) *int { return &o.MaxItems }, func(o *ListOpts) **bool { return &o.Paginated })

func WithCreateOptions(value CreateOpts) CreateOption { return request.Custom(senlin.Snapshot(value)) }
func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.Custom(senlin.Snapshot(value)) }
func WithListOptions(value ListOpts) ListOption       { return request.Custom(senlin.Snapshot(value)) }
func WithCreateClusterID(value string) CreateOption {
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		config.Options.ClusterID = request.Present(value)
		return nil
	})
}
func WithCreateClusterIDNull() CreateOption {
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		config.Options.ClusterID = request.Null[string]()
		return nil
	})
}
func WithCreateAction(value string) CreateOption {
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		config.Options.Action = request.Present(value)
		return nil
	})
}
func WithCreateActionNull() CreateOption {
	return request.Custom(func(config *request.Config[CreateOpts]) error {
		config.Options.Action = request.Null[string]()
		return nil
	})
}
func WithUpdateName(value string) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Name = request.Present(value)
		return nil
	})
}
func WithUpdateNameNull() UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Name = request.Null[string]()
		return nil
	})
}
func WithUpdateAction(value string) UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Action = request.Present(value)
		return nil
	})
}
func WithUpdateActionNull() UpdateOption {
	return request.Custom(func(config *request.Config[UpdateOpts]) error {
		config.Options.Action = request.Null[string]()
		return nil
	})
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
	return request.Custom(withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Actor = raw }))
}
func WithCreateParams(value any) CreateOption {
	return request.Custom(withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Params = raw }))
}
func WithUpdateParams(value any) UpdateOption {
	return request.Custom(withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Params = raw }))
}

func WithListGlobalProject(value bool) ListOption {
	return request.Custom(func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	})
}
func WithListUserID(value string) ListOption {
	return request.Custom(func(config *request.Config[ListOpts]) error { config.Options.UserID = value; return nil })
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
