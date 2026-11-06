package profiles

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
	Name     string          `json:"name"`
	Spec     json.RawMessage `json:"spec"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// UpdateOpts only exposes mutable fields. Metadata nil omits it; raw JSON null
// and an empty object remain explicit values. The server validates null support.
type UpdateOpts struct {
	Name     *string         `json:"name,omitempty"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type ValidateOpts struct {
	Spec json.RawMessage `json:"spec"`
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
	Sort          string
	GlobalProject *bool
}

// CreateOption is one option of Create: WithCreateOptions, WithCreateSpec,
// WithCreateMetadata, request.Header, request.Field or request.Custom over
// *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = senlin.WriteCalls[CreateOpts]()

// UpdateOption is one option of Update, TrackedProfile.Edit and
// TrackedProfile.Commit: WithUpdateOptions, WithUpdateName,
// WithUpdateMetadata, request.Header, request.Field or request.Custom over
// *request.Config[UpdateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

var updateCalls = senlin.WriteCalls[UpdateOpts]()

// ValidateOption is one option of Validate: WithValidateOptions,
// WithValidateSpec, request.Header, request.Field or request.Custom over
// *request.Config[ValidateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ValidateOption = request.WriteCall

var validateCalls = senlin.WriteCalls[ValidateOpts]()

// ListOption is one option of List and All: WithListOptions,
// WithListGlobalProject, WithListFilter, request.Header, request.Microversion,
// request.Query (a string or []string value), request.MaxItems,
// request.Paginated or request.Custom over *request.Config[ListOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = senlin.ListCalls(func(o *ListOpts) *int { return &o.MaxItems }, func(o *ListOpts) **bool { return &o.Paginated })

func WithCreateOptions(value CreateOpts) CreateOption { return request.Custom(senlin.Snapshot(value)) }
func WithUpdateOptions(value UpdateOpts) UpdateOption { return request.Custom(senlin.Snapshot(value)) }
func WithValidateOptions(value ValidateOpts) ValidateOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithListOptions(value ListOpts) ListOption { return request.Custom(senlin.Snapshot(value)) }

func withJSON[T any](value any, set func(*T, json.RawMessage)) request.Option[T] {
	encoded, err := json.Marshal(value)
	return func(config *request.Config[T]) error {
		if err != nil {
			return fmt.Errorf("%w: profile JSON input: %v", resource.ErrInvalidOption, err)
		}
		set(&config.Options, append(json.RawMessage(nil), encoded...))
		return nil
	}
}

func WithCreateSpec(value any) CreateOption {
	return request.Custom(withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Spec = raw }))
}
func WithCreateMetadata(value any) CreateOption {
	return request.Custom(withJSON(value, func(opts *CreateOpts, raw json.RawMessage) { opts.Metadata = raw }))
}
func WithUpdateMetadata(value any) UpdateOption {
	return request.Custom(withJSON(value, func(opts *UpdateOpts, raw json.RawMessage) { opts.Metadata = raw }))
}
func WithValidateSpec(value any) ValidateOption {
	return request.Custom(withJSON(value, func(opts *ValidateOpts, raw json.RawMessage) { opts.Spec = raw }))
}

func WithListGlobalProject(value bool) ListOption {
	return request.Custom(func(config *request.Config[ListOpts]) error {
		copy := value
		config.Options.GlobalProject = &copy
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
	for key, field := range map[string]string{"name": value.Name, "type": value.Type, "sort": value.Sort} {
		if field != "" {
			query.Set(key, field)
		}
	}
	if value.GlobalProject != nil {
		query.Set("global_project", strconv.FormatBool(*value.GlobalProject))
	}
	for key, values := range config.Query {
		switch key {
		case "limit", "marker", "max_items", "paginated", "name", "type", "sort", "global_project", "metadata", "spec", "id", "project", "project_id", "domain", "domain_id", "user", "user_id", "created_at", "updated_at":
			return nil, fmt.Errorf("%w: query %q is a concrete option or local profile filter", resource.ErrInvalidOption, key)
		}
		query[key] = append([]string(nil), values...)
	}
	return query, nil
}
