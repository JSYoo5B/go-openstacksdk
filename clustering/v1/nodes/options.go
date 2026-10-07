package nodes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
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

type CreateOption = request.Option[CreateOpts]
type UpdateOption = request.Option[UpdateOpts]
type GetOption = request.Option[GetOpts]
type ListOption = request.Option[ListOpts]
type DeleteOption = request.Option[DeleteOpts]

func WithCreateOptions(value CreateOpts) CreateOption { return senlin.Snapshot(value) }
func WithUpdateOptions(value UpdateOpts) UpdateOption { return senlin.Snapshot(value) }
func WithGetOptions(value GetOpts) GetOption          { return senlin.Snapshot(value) }
func WithListOptions(value ListOpts) ListOption       { return senlin.Snapshot(value) }
func WithDeleteOptions(value DeleteOpts) DeleteOption {
	value.IgnoreMissing = senlin.Bool(value.IgnoreMissing)
	return func(config *request.Config[DeleteOpts]) error {
		config.Options = DeleteOpts{Force: value.Force, IgnoreMissing: senlin.Bool(value.IgnoreMissing)}
		return nil
	}
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
	return withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Metadata = raw })
}
func WithUpdateMetadata(value any) UpdateOption {
	return withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Metadata = raw })
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
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}
func WithGetHeader(key, value string) GetOption { return request.WithHeader[GetOpts](key, value) }
func WithGetQuery(key, value string) GetOption  { return request.WithQuery[GetOpts](key, value) }

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

func WithGetDetails(value bool) GetOption {
	return func(config *request.Config[GetOpts]) error { copy := value; config.Options.Details = &copy; return nil }
}
func WithListGlobalProject(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
		return nil
	}
}
func WithListShowDetails(value bool) ListOption {
	return func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.ShowDetails = &copy
		return nil
	}
}
func WithDeleteForce(value bool) DeleteOption {
	return func(config *request.Config[DeleteOpts]) error {
		config.Options.Force = value
		return nil
	}
}
func WithDeleteIgnoreMissing(value bool) DeleteOption {
	return func(config *request.Config[DeleteOpts]) error {
		copy := value
		config.Options.IgnoreMissing = &copy
		return nil
	}
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
