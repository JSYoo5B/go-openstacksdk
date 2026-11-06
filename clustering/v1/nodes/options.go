package nodes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type CreateOpts struct {
	Name      string                   `json:"name"`
	ProfileID string                   `json:"profile_id"`
	ClusterID request.Optional[string] `json:"cluster_id,omitzero"`
	Role      request.Optional[string] `json:"role,omitzero"`
	Metadata  json.RawMessage          `json:"metadata,omitempty"`
}

// A zero Optional omits its field; Present preserves false/empty values and
// Null emits JSON null. The deployment validates nullable field semantics.
type UpdateOpts struct {
	Name      request.Optional[string] `json:"name,omitzero"`
	ProfileID request.Optional[string] `json:"profile_id,omitzero"`
	Role      request.Optional[string] `json:"role,omitzero"`
	Metadata  json.RawMessage          `json:"metadata,omitempty"`
	Tainted   request.Optional[bool]   `json:"tainted,omitzero"`
}

type GetOpts struct{ Details *bool }

type ListOpts struct {
	// MaxItems counts wire rows before local filtering; zero is unlimited.
	MaxItems int
	// Paginated nil uses all pages; an explicit false returns one page.
	Paginated     *bool
	Limit         int
	Marker        string
	Name          string
	ClusterID     string
	Status        string
	Sort          string
	GlobalProject *bool
	ShowDetails   *bool
}

type DeleteOpts struct {
	Force         bool  `json:"force"`
	IgnoreMissing *bool `json:"-"`
}

// CreateOption is one option of Create: WithCreateOptions, the WithCreate
// field options, request.Header, request.Field or request.Custom over
// *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = senlin.WriteCalls[CreateOpts]()

// UpdateOption is one option of Update, TrackedNode.Edit and
// TrackedNode.Commit: WithUpdateOptions, the WithUpdate field options,
// request.Header, request.Field or request.Custom over
// *request.Config[UpdateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

var updateCalls = senlin.WriteCalls[UpdateOpts]()

// GetOption is one option of Get: WithGetOptions, WithGetDetails,
// request.Header, request.Query (a string or []string value) or request.Custom
// over *request.Config[GetOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetQueryCall

var getCalls = senlin.GetQueryCalls[GetOpts]()

// ListOption is one option of List and All: WithListOptions,
// WithListGlobalProject, WithListShowDetails, WithListFilter, request.Header,
// request.Microversion, request.Query (a string or []string value),
// request.MaxItems, request.Paginated or request.Custom over
// *request.Config[ListOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = senlin.ListCalls(func(o *ListOpts) *int { return &o.MaxItems }, func(o *ListOpts) **bool { return &o.Paginated })

// DeleteOption is one option of Delete: WithDeleteOptions, WithDeleteForce,
// request.IgnoreMissing, request.Header or request.Custom over
// *request.Config[DeleteOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

var deleteCalls = senlin.DeleteCalls(func(o *DeleteOpts) **bool { return &o.IgnoreMissing })

func WithCreateOptions(value CreateOpts) CreateOption { return request.Custom(senlin.Snapshot(value)) }
func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.Custom(senlin.Snapshot(value)) }
func WithGetOptions(value GetOpts) GetOption          { return request.Custom(senlin.Snapshot(value)) }
func WithListOptions(value ListOpts) ListOption       { return request.Custom(senlin.Snapshot(value)) }
func WithDeleteOptions(value DeleteOpts) DeleteOption {
	value.IgnoreMissing = senlin.Bool(value.IgnoreMissing)
	return request.Custom(func(config *request.Config[DeleteOpts]) error {
		config.Options = DeleteOpts{Force: value.Force, IgnoreMissing: senlin.Bool(value.IgnoreMissing)}
		return nil
	})
}

func withJSON[T any](value any, set func(*T, json.RawMessage)) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: node JSON input: %v", resource.ErrInvalidOption, err)
		}
		set(&config.Options, append(json.RawMessage(nil), encoded...))
		return nil
	}
}

func WithCreateMetadata(value any) CreateOption {
	return request.Custom(withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Metadata = raw }))
}
func WithUpdateMetadata(value any) UpdateOption {
	return request.Custom(withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Metadata = raw }))
}

func WithGetDetails(value bool) GetOption {
	return request.Custom(func(config *request.Config[GetOpts]) error { copy := value; config.Options.Details = &copy; return nil })
}
func WithListGlobalProject(value bool) ListOption {
	return request.Custom(func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	})
}
func WithListShowDetails(value bool) ListOption {
	return request.Custom(func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.ShowDetails = &copy
		return nil
	})
}
func WithDeleteForce(value bool) DeleteOption {
	return request.Custom(func(config *request.Config[DeleteOpts]) error {
		config.Options.Force = value
		return nil
	})
}

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
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
	if err := senlin.SortGrammar(value.Sort); err != nil {
		return nil, err
	}
	query, err := senlin.Query(request.Config[senlin.ListOpts]{Options: senlin.ListOpts{Limit: value.Limit, Marker: value.Marker}})
	if err != nil {
		return nil, err
	}
	for key, field := range map[string]string{"name": value.Name, "cluster_id": value.ClusterID, "status": value.Status, "sort": value.Sort} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	if value.ShowDetails != nil {
		query.Set("show_details", strconv.FormatBool(*value.ShowDetails))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "max_items", "paginated", "name", "cluster_id", "status", "sort", "global_project", "show_details", "details", "data", "metadata", "id", "project", "project_id", "domain", "domain_id", "user", "user_id", "created_at", "updated_at", "init_at", "physical_id", "profile_id", "profile_name", "index", "role", "status_reason", "dependents", "tainted":
			return nil, fmt.Errorf("%w: query %q is a concrete option or local node filter", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}
